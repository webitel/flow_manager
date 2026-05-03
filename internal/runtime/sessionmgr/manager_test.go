package sessionmgr_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/state"
)

// --- fakes ---

type dispatchCall struct {
	ctx     context.Context
	key     string
	payload map[string]string
}

type fakeDispatcher struct {
	mu    sync.Mutex
	calls []dispatchCall
	err   error
}

func (f *fakeDispatcher) Dispatch(ctx context.Context, key string, payload map[string]string) error {
	f.mu.Lock()
	cp := make(map[string]string, len(payload))
	for k, v := range payload {
		cp[k] = v
	}
	f.calls = append(f.calls, dispatchCall{ctx: ctx, key: key, payload: cp})
	err := f.err
	f.mu.Unlock()
	return err
}

func (f *fakeDispatcher) Calls() []dispatchCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dispatchCall, len(f.calls))
	copy(out, f.calls)
	return out
}

type fakeLoader struct {
	mu  sync.Mutex
	rec *persistence.Record
	err error
}

func (f *fakeLoader) LoadByConnectionID(_ context.Context, _ string) (*persistence.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rec, f.err
}

func (f *fakeLoader) SetRecord(rec *persistence.Record) {
	f.mu.Lock()
	f.rec = rec
	f.mu.Unlock()
}

type fakeConn struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc

	mu           sync.Mutex
	handler      func(text string)
	unregistered bool
}

func newFakeConn(id string) *fakeConn {
	ctx, cancel := context.WithCancel(context.Background())
	return &fakeConn{id: id, ctx: ctx, cancel: cancel}
}

func (f *fakeConn) Id() string               { return f.id }
func (f *fakeConn) Context() context.Context { return f.ctx }
func (f *fakeConn) Close()                   { f.cancel() }

func (f *fakeConn) OnInboundMessage(h func(text string)) (unregister func()) {
	f.mu.Lock()
	f.handler = h
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		f.unregistered = true
		f.mu.Unlock()
	}
}

func (f *fakeConn) Deliver(text string) {
	f.mu.Lock()
	h := f.handler
	f.mu.Unlock()
	if h != nil {
		h(text)
	}
}

func (f *fakeConn) Unregistered() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.unregistered
}

// --- helpers ---

func testLogger() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("waitFor timeout: %s", msg)
}

func suspendedRec(connID string) *persistence.Record {
	return &persistence.Record{
		ID:           uuid.New(),
		ConnectionID: connID,
		Status:       state.StatusSuspended,
		State: state.ExecState{
			Pending: &state.PendingIntent{
				OpName:    "recv_message",
				ResumeKey: "msg:" + connID,
			},
		},
	}
}

// --- tests ---

func TestWatch_InboundMessage_StaysSuspended(t *testing.T) {
	conn := newFakeConn("c1")
	defer conn.Close()
	disp := &fakeDispatcher{}
	rec := suspendedRec("c1")
	loader := &fakeLoader{rec: rec}

	var mu sync.Mutex
	teardownCalls := 0
	teardown := func() {
		mu.Lock()
		teardownCalls++
		mu.Unlock()
	}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, teardown)

	conn.Deliver("hello")

	waitFor(t, func() bool { return len(disp.Calls()) == 1 }, "dispatch called once")

	call := disp.Calls()[0]
	if call.key != "msg:c1" {
		t.Errorf("key = %q, want msg:c1", call.key)
	}
	if call.payload["msg"] != "hello" {
		t.Errorf("payload msg = %q, want hello", call.payload["msg"])
	}

	// Loader still returns suspended → teardown not called.
	time.Sleep(20 * time.Millisecond)
	mu.Lock()
	got := teardownCalls
	mu.Unlock()
	if got != 0 {
		t.Errorf("teardownCalls = %d, want 0", got)
	}
}

func TestWatch_InboundMessage_FlowCompletes(t *testing.T) {
	conn := newFakeConn("c2")
	defer conn.Close()
	disp := &fakeDispatcher{}
	rec := suspendedRec("c2")

	completed := *rec
	completed.Status = state.StatusCompleted
	loader := &fakeLoader{rec: &completed}

	var mu sync.Mutex
	teardownCalls := 0
	teardown := func() {
		mu.Lock()
		teardownCalls++
		mu.Unlock()
	}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, teardown)

	conn.Deliver("done")

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return teardownCalls == 1
	}, "teardown called once after completion")

	if !conn.Unregistered() {
		t.Errorf("expected unregister to be called")
	}
}

func TestWatch_InitialMessageReplay(t *testing.T) {
	conn := newFakeConn("c3")
	defer conn.Close()
	disp := &fakeDispatcher{}
	rec := suspendedRec("c3")
	loader := &fakeLoader{rec: rec}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "recovery-msg", nil, nil)

	waitFor(t, func() bool { return len(disp.Calls()) >= 1 }, "initial dispatch")
	if got := disp.Calls()[0].payload["msg"]; got != "recovery-msg" {
		t.Errorf("initial msg payload = %q, want recovery-msg", got)
	}
}

func TestWatch_WakeAtPast_DispatchesTimeout(t *testing.T) {
	conn := newFakeConn("c4")
	defer conn.Close()
	disp := &fakeDispatcher{}
	past := time.Now().Add(-time.Second).UTC().Format(time.RFC3339)
	rec := &persistence.Record{
		ID:           uuid.New(),
		ConnectionID: "c4",
		Status:       state.StatusSuspended,
		State: state.ExecState{
			Pending: &state.PendingIntent{
				OpName:    "recv_message",
				ResumeKey: "msg:c4",
				Args:      map[string]string{"wake_at": past},
			},
		},
	}
	loader := &fakeLoader{rec: rec}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, nil)

	waitFor(t, func() bool {
		for _, c := range disp.Calls() {
			if c.payload["timeout"] == "true" {
				return true
			}
		}
		return false
	}, "wake_at past dispatched timeout")
}

func TestWatch_WakeAtFuture_DispatchesAfterDelay(t *testing.T) {
	conn := newFakeConn("c5")
	defer conn.Close()
	disp := &fakeDispatcher{}
	// RFC3339 has second precision; truncate to align so the parsed-back delay
	// is between ~1s and ~2s regardless of where in the current second we are.
	future := time.Now().Truncate(time.Second).Add(2 * time.Second).UTC().Format(time.RFC3339)
	rec := &persistence.Record{
		ID:           uuid.New(),
		ConnectionID: "c5",
		Status:       state.StatusSuspended,
		State: state.ExecState{
			Pending: &state.PendingIntent{
				OpName:    "recv_message",
				ResumeKey: "msg:c5",
				Args:      map[string]string{"wake_at": future},
			},
		},
	}
	loader := &fakeLoader{rec: rec}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, nil)

	// Earliest possible fire is ~1s from now. 100ms must still be quiet.
	time.Sleep(100 * time.Millisecond)
	if len(disp.Calls()) != 0 {
		t.Fatalf("dispatch fired before wake_at deadline (calls=%d)", len(disp.Calls()))
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, c := range disp.Calls() {
			if c.payload["timeout"] == "true" {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("wake_at future dispatch never fired")
}

func TestWatch_ContextCancel_TriggersTeardown(t *testing.T) {
	conn := newFakeConn("c6")
	disp := &fakeDispatcher{}
	rec := suspendedRec("c6")
	loader := &fakeLoader{rec: rec}

	var mu sync.Mutex
	teardownCalls := 0
	teardown := func() {
		mu.Lock()
		teardownCalls++
		mu.Unlock()
	}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, teardown)

	conn.Close()

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return teardownCalls == 1
	}, "teardown after context cancel")

	if !conn.Unregistered() {
		t.Errorf("expected unregister to be called on context cancel")
	}
}

func TestWatch_TeardownInvokedOnlyOnce(t *testing.T) {
	conn := newFakeConn("c7")
	disp := &fakeDispatcher{}
	rec := suspendedRec("c7")

	completed := *rec
	completed.Status = state.StatusCompleted
	loader := &fakeLoader{rec: &completed}

	var mu sync.Mutex
	teardownCalls := 0
	teardown := func() {
		mu.Lock()
		teardownCalls++
		mu.Unlock()
	}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, teardown)

	conn.Deliver("done")

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return teardownCalls == 1
	}, "first teardown")

	conn.Close() // would otherwise trigger a second teardown via ctx.Done watcher.

	time.Sleep(40 * time.Millisecond)
	mu.Lock()
	if teardownCalls != 1 {
		t.Errorf("teardownCalls = %d, want 1", teardownCalls)
	}
	mu.Unlock()
}

func TestWatch_DecoratorAppliedToDispatchContext(t *testing.T) {
	type ctxKey struct{}

	conn := newFakeConn("c8")
	defer conn.Close()
	disp := &fakeDispatcher{}
	rec := suspendedRec("c8")
	loader := &fakeLoader{rec: rec}

	decorator := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, ctxKey{}, "decorated")
	}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", decorator, nil)

	conn.Deliver("hi")
	waitFor(t, func() bool { return len(disp.Calls()) == 1 }, "dispatch")

	val, _ := disp.Calls()[0].ctx.Value(ctxKey{}).(string)
	if val != "decorated" {
		t.Errorf("decorator value = %q, want decorated", val)
	}
}

func TestWatch_FallbackResumeKey_NoPending(t *testing.T) {
	conn := newFakeConn("c9")
	defer conn.Close()
	disp := &fakeDispatcher{}
	rec := &persistence.Record{
		ID:           uuid.New(),
		ConnectionID: "c9",
		Status:       state.StatusSuspended,
		State:        state.ExecState{},
	}
	loader := &fakeLoader{rec: rec}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, nil)

	conn.Deliver("hello")
	waitFor(t, func() bool { return len(disp.Calls()) == 1 }, "dispatch")
	if got := disp.Calls()[0].key; got != "msg:c9" {
		t.Errorf("resume key = %q, want msg:c9", got)
	}
}

func TestWatch_DispatchError_LeavesSuspendedAlone(t *testing.T) {
	conn := newFakeConn("c10")
	defer conn.Close()
	disp := &fakeDispatcher{err: errors.New("boom")}
	rec := suspendedRec("c10")
	loader := &fakeLoader{rec: rec}

	var mu sync.Mutex
	teardownCalls := 0
	teardown := func() {
		mu.Lock()
		teardownCalls++
		mu.Unlock()
	}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, teardown)

	conn.Deliver("x")
	waitFor(t, func() bool { return len(disp.Calls()) == 1 }, "dispatch attempted")

	time.Sleep(30 * time.Millisecond)
	mu.Lock()
	if teardownCalls != 0 {
		t.Errorf("teardownCalls = %d, want 0 (loader still suspended)", teardownCalls)
	}
	mu.Unlock()
}

func TestWatch_LoaderError_TriggersTeardown(t *testing.T) {
	conn := newFakeConn("c11")
	defer conn.Close()
	disp := &fakeDispatcher{}
	rec := suspendedRec("c11")
	loader := &fakeLoader{rec: rec, err: errors.New("db down")}

	var mu sync.Mutex
	teardownCalls := 0
	teardown := func() {
		mu.Lock()
		teardownCalls++
		mu.Unlock()
	}

	m := sessionmgr.New(disp, loader, testLogger())
	m.Watch(conn, rec, "", nil, teardown)

	conn.Deliver("x")

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return teardownCalls == 1
	}, "teardown after loader error")
}
