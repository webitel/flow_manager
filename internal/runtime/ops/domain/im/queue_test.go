package im

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/webitel/wlog"

	genpb "github.com/webitel/flow_manager/api/gen/cc"
	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── stubDialog ────────────────────────────────────────────────────────────────

// stubDialog is a minimal hand-written stub for chatdomain.IMDialog.
// Only methods called by cancelQueueOp and joinQueueOp are implemented.
type stubDialog struct {
	id       string
	domainId int64
	threadId string
	from     chatdomain.ImEndpoint
	to       chatdomain.ImEndpoint
	lastMsg  chatdomain.Message
	queueKey *queue.InQueueKey
	mu       sync.Mutex
	vars     map[string]string

	// recorded side-effects
	setQueueCalls []*queue.InQueueKey
}

// flow.Connection
func (d *stubDialog) Type() flow.ConnectionType      { return flow.ConnectionTypeIM }
func (d *stubDialog) Id() string                      { return d.id }
func (d *stubDialog) NodeId() string                  { return "" }
func (d *stubDialog) DomainId() int64                 { return d.domainId }
func (d *stubDialog) Context() context.Context        { return context.Background() }
func (d *stubDialog) Close() error                    { return nil }
func (d *stubDialog) Log() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}
func (d *stubDialog) Variables() map[string]string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.vars
}
func (d *stubDialog) Get(key string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	v, ok := d.vars[key]
	return v, ok
}
func (d *stubDialog) Set(_ context.Context, vars flow.Variables) (flow.Response, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.vars == nil {
		d.vars = make(map[string]string)
	}
	for k, v := range vars {
		d.vars[k] = fmt.Sprintf("%v", v)
	}
	return nil, nil
}
func (d *stubDialog) ParseText(text string, _ ...flow.ParseOption) string { return text }

// chatdomain.IMDialog
func (d *stubDialog) ThreadId() string                    { return d.threadId }
func (d *stubDialog) From() chatdomain.ImEndpoint         { return d.from }
func (d *stubDialog) To() chatdomain.ImEndpoint           { return d.to }
func (d *stubDialog) LastMessage() chatdomain.Message     { return d.lastMsg }
func (d *stubDialog) SchemaId() int                       { return 0 }
func (d *stubDialog) Stop(_ error)                        {}
func (d *stubDialog) IsTransfer() bool                    { return false }
func (d *stubDialog) DumpExportVariables() map[string]string { return nil }

func (d *stubDialog) GetQueueKey() *queue.InQueueKey {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.queueKey
}

func (d *stubDialog) SetQueue(k *queue.InQueueKey) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.setQueueCalls = append(d.setQueueCalls, k)
	d.queueKey = k
	return true
}

func (d *stubDialog) SendMessage(_ context.Context, _ chatdomain.ChatMessageOutbound) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) SendTextMessage(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) SendImageMessage(_ context.Context, _ chatdomain.ChatMessageOutbound) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) SendDocumentMessage(_ context.Context, _ chatdomain.ChatMessageOutbound) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) SendFile(_ context.Context, _ string, _ *files.File, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) SendMenu(_ context.Context, _ *chatdomain.ChatMenuArgs) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) Export(_ context.Context, _ []string) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) UnSet(_ context.Context, _ []string) (flow.Response, error) {
	panic("not implemented")
}
func (d *stubDialog) LastMessages(_ int) []chatdomain.ChatMessage { panic("not implemented") }

// Compile-time check
var _ chatdomain.IMDialog = (*stubDialog)(nil)

// ── fakeQueueDeps ─────────────────────────────────────────────────────────────

type fakeQueueDeps struct {
	cancelErr        error
	findQueueId      int32
	findQueueErr     error
	agentId          *int32
	joinAttId        int64
	joinCh           chan domcc.QueueEvent
	joinErr          error
	leavingCalled    bool
	leavingMu        sync.Mutex
}

func (f *fakeQueueDeps) CancelAttempt(_ context.Context, _ queue.InQueueKey, _ string) error {
	return f.cancelErr
}
func (f *fakeQueueDeps) FindQueueByName(_ int64, _ string) (int32, error) {
	return f.findQueueId, f.findQueueErr
}
func (f *fakeQueueDeps) GetAgentIdByExtension(_ int64, _ string) (*int32, error) {
	return f.agentId, nil
}
func (f *fakeQueueDeps) JoinIMToInboundQueue(_ context.Context, _ *genpb.IMJoinToQueueRequest) (int64, <-chan domcc.QueueEvent, error) {
	return f.joinAttId, f.joinCh, f.joinErr
}
func (f *fakeQueueDeps) LeavingIMToInboundQueue(_ int64) {
	f.leavingMu.Lock()
	f.leavingCalled = true
	f.leavingMu.Unlock()
}

// ── fakeCoord ─────────────────────────────────────────────────────────────────

type fakeCoord struct {
	mu       sync.Mutex
	calls    []dispatchCall
	dispatchErr error
}

type dispatchCall struct {
	key     string
	payload map[string]string
}

func (f *fakeCoord) Dispatch(_ context.Context, key string, payload map[string]string) error {
	f.mu.Lock()
	f.calls = append(f.calls, dispatchCall{key: key, payload: payload})
	f.mu.Unlock()
	return f.dispatchErr
}

func (f *fakeCoord) waitForN(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		got := len(f.calls)
		f.mu.Unlock()
		if got >= n {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// ── helpers ───────────────────────────────────────────────────────────────────

func ctxWithDialog(d chatdomain.IMDialog) context.Context {
	return connctx.WithConnection(context.Background(), d)
}

func freshInput(connID string, args map[string]any) ops.OpInput {
	return ops.OpInput{
		ConnID: connID,
		Node:   &tree.Node{Args: args},
	}
}

func resumeInput(connID string, payload map[string]string, vars map[string]string) ops.OpInput {
	return ops.OpInput{
		ConnID:        connID,
		Node:          &tree.Node{},
		ResumePayload: payload,
		Variables:     vars,
	}
}

// ── cancelQueue tests ─────────────────────────────────────────────────────────

func TestCancelQueue_NoDialog(t *testing.T) {
	op := &cancelQueueOp{deps: &fakeQueueDeps{}}
	_, err := op.Execute(context.Background(), freshInput("c1", nil))
	if err == nil {
		t.Fatal("expected error when no dialog in context")
	}
}

func TestCancelQueue_NoQueueKey(t *testing.T) {
	dialog := &stubDialog{} // queueKey is nil
	op := &cancelQueueOp{deps: &fakeQueueDeps{}}
	_, err := op.Execute(ctxWithDialog(dialog), freshInput("c1", nil))
	if err == nil {
		t.Fatal("expected error when no active queue key")
	}
}

func TestCancelQueue_DepError(t *testing.T) {
	key := &queue.InQueueKey{AttemptId: 99}
	dialog := &stubDialog{queueKey: key}
	deps := &fakeQueueDeps{cancelErr: fmt.Errorf("cc unavailable")}
	op := &cancelQueueOp{deps: deps}
	_, err := op.Execute(ctxWithDialog(dialog), freshInput("c1", nil))
	if err == nil {
		t.Fatal("expected error when CancelAttempt fails")
	}
}

func TestCancelQueue_Success(t *testing.T) {
	key := &queue.InQueueKey{AttemptId: 42}
	dialog := &stubDialog{queueKey: key}
	op := &cancelQueueOp{deps: &fakeQueueDeps{}}

	out, err := op.Execute(ctxWithDialog(dialog), freshInput("c1", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["cc_cancel"] != "true" {
		t.Errorf("cc_cancel = %q, want %q", out.SetVars["cc_cancel"], "true")
	}
	// SetQueue(nil) must have been called to clear the key.
	dialog.mu.Lock()
	calls := dialog.setQueueCalls
	dialog.mu.Unlock()
	if len(calls) != 1 || calls[0] != nil {
		t.Errorf("SetQueue calls = %v, want [nil]", calls)
	}
}

// ── joinQueue fresh-path tests ────────────────────────────────────────────────

func TestJoinIMQueue_NoDialog(t *testing.T) {
	op := &joinQueueOp{deps: &fakeQueueDeps{}, coord: &fakeCoord{}}
	_, err := op.Execute(context.Background(), freshInput("c1", map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err == nil {
		t.Fatal("expected error when no dialog in context")
	}
}

func TestJoinIMQueue_FindQueueByName_Fails(t *testing.T) {
	dialog := &stubDialog{}
	deps := &fakeQueueDeps{
		findQueueErr: fmt.Errorf("not found"),
		joinCh:       make(chan domcc.QueueEvent),
	}
	op := &joinQueueOp{deps: deps, coord: &fakeCoord{}}
	_, err := op.Execute(ctxWithDialog(dialog), freshInput("c1", map[string]any{
		"queue": map[string]any{"name": "sales"},
	}))
	if err == nil {
		t.Fatal("expected error when FindQueueByName fails")
	}
}

func TestJoinIMQueue_JoinFails(t *testing.T) {
	dialog := &stubDialog{}
	deps := &fakeQueueDeps{joinErr: fmt.Errorf("cc down")}
	op := &joinQueueOp{deps: deps, coord: &fakeCoord{}}
	_, err := op.Execute(ctxWithDialog(dialog), freshInput("c1", map[string]any{
		"queue": map[string]any{"id": 5},
	}))
	if err == nil {
		t.Fatal("expected error when JoinIMToInboundQueue fails")
	}
}

func TestJoinIMQueue_FreshPath_Suspends(t *testing.T) {
	dialog := &stubDialog{id: "conn-1", threadId: "thread-1"}
	ch := make(chan domcc.QueueEvent, 1)
	deps := &fakeQueueDeps{joinAttId: 77, joinCh: ch}
	coord := &fakeCoord{}
	op := &joinQueueOp{deps: deps, coord: coord}

	out, err := op.Execute(ctxWithDialog(dialog), freshInput("conn-1", map[string]any{
		"queue": map[string]any{"id": 3},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey == "" {
		t.Error("expected SuspendKey to be set")
	}
	if out.Pending == nil {
		t.Error("expected Pending to be set")
	}
	if out.SetVars["cc_attempt_id"] != "77" {
		t.Errorf("cc_attempt_id = %q, want %q", out.SetVars["cc_attempt_id"], "77")
	}
	if !out.ReenterOnResume {
		t.Error("expected ReenterOnResume=true")
	}
	close(ch) // signal goroutine to stop
}

func TestJoinIMQueue_FreshPath_QueueByName(t *testing.T) {
	// When queue.id == 0 and queue.name is set, FindQueueByName is called.
	dialog := &stubDialog{}
	ch := make(chan domcc.QueueEvent, 1)
	deps := &fakeQueueDeps{findQueueId: 9, joinAttId: 1, joinCh: ch}
	op := &joinQueueOp{deps: deps, coord: &fakeCoord{}}

	out, err := op.Execute(ctxWithDialog(dialog), freshInput("c1", map[string]any{
		"queue": map[string]any{"name": "support"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey == "" {
		t.Error("expected suspension")
	}
	close(ch)
}

func TestJoinIMQueue_Goroutine_DispatchesEvents(t *testing.T) {
	// Verifies the CC goroutine dispatches events through coord.
	dialog := &stubDialog{id: "conn-2"}
	ch := make(chan domcc.QueueEvent, 3)
	ch <- domcc.QueueEvent{Event: "offering"}
	ch <- domcc.QueueEvent{Event: "bridged"}
	ch <- domcc.QueueEvent{Event: "leaving", Result: "success"}
	close(ch)

	coord := &fakeCoord{}
	deps := &fakeQueueDeps{joinAttId: 5, joinCh: ch}
	op := &joinQueueOp{deps: deps, coord: coord}

	_, err := op.Execute(ctxWithDialog(dialog), freshInput("conn-2", map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !coord.waitForN(3, time.Second) {
		t.Fatalf("expected 3 coord.Dispatch calls, got %d", len(coord.calls))
	}

	events := make([]string, len(coord.calls))
	for i, c := range coord.calls {
		events[i] = c.payload[ccEventKey]
	}
	want := []string{"offering", "bridged", "leaving"}
	for i, w := range want {
		if events[i] != w {
			t.Errorf("event[%d] = %q, want %q", i, events[i], w)
		}
	}

	// "leaving" event carries cc_result.
	if coord.calls[2].payload[ccResultKey] != "success" {
		t.Errorf("cc_result = %q, want %q", coord.calls[2].payload[ccResultKey], "success")
	}
}

func TestJoinIMQueue_Goroutine_LeavingCallsCancelAndCleanup(t *testing.T) {
	// After the channel closes, LeavingIMToInboundQueue and SetQueue(nil)
	// must both be called.
	dialog := &stubDialog{id: "conn-3"}
	ch := make(chan domcc.QueueEvent, 1)
	ch <- domcc.QueueEvent{Event: "leaving"}
	close(ch)

	deps := &fakeQueueDeps{joinAttId: 11, joinCh: ch}
	coord := &fakeCoord{}
	op := &joinQueueOp{deps: deps, coord: coord}

	_, err := op.Execute(ctxWithDialog(dialog), freshInput("conn-3", map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for goroutine to finish.
	if !coord.waitForN(1, time.Second) {
		t.Fatal("goroutine did not dispatch in time")
	}
	// Give goroutine time for deferred cleanup.
	time.Sleep(20 * time.Millisecond)

	deps.leavingMu.Lock()
	called := deps.leavingCalled
	deps.leavingMu.Unlock()
	if !called {
		t.Error("expected LeavingIMToInboundQueue to be called")
	}

	dialog.mu.Lock()
	setCalls := dialog.setQueueCalls
	dialog.mu.Unlock()
	if len(setCalls) == 0 || setCalls[len(setCalls)-1] != nil {
		t.Error("expected SetQueue(nil) to be called by goroutine cleanup")
	}
}

// ── joinQueue resume-path tests ───────────────────────────────────────────────

func TestJoinIMQueue_Resume_Leaving(t *testing.T) {
	op := &joinQueueOp{deps: &fakeQueueDeps{}, coord: &fakeCoord{}}
	out, err := op.Execute(context.Background(), resumeInput("c1",
		map[string]string{ccEventKey: "leaving"},
		nil,
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey != "" {
		t.Error("leaving should exit the op (no re-suspension)")
	}
	if out.ReSuspend {
		t.Error("leaving should not re-suspend")
	}
}

func TestJoinIMQueue_Resume_Leaving_WithResult(t *testing.T) {
	op := &joinQueueOp{deps: &fakeQueueDeps{}, coord: &fakeCoord{}}
	out, err := op.Execute(context.Background(), resumeInput("c1",
		map[string]string{ccEventKey: "leaving", ccResultKey: "abandoned"},
		nil,
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["cc_result"] != "abandoned" {
		t.Errorf("cc_result = %q, want %q", out.SetVars["cc_result"], "abandoned")
	}
}

func TestJoinIMQueue_Resume_UnknownEvent_ReSuspends(t *testing.T) {
	op := &joinQueueOp{deps: &fakeQueueDeps{}, coord: &fakeCoord{}}
	out, err := op.Execute(context.Background(), resumeInput("c1",
		map[string]string{ccEventKey: "bridged"},
		nil,
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.ReSuspend {
		t.Error("unknown event should re-suspend")
	}
	if !out.ReenterOnResume {
		t.Error("unknown event should set ReenterOnResume")
	}
}

func TestJoinIMQueue_Resume_EmptyEvent_NoTrigger_ReSuspends(t *testing.T) {
	op := &joinQueueOp{deps: &fakeQueueDeps{}, coord: &fakeCoord{}}
	in := resumeInput("c1",
		map[string]string{ccEventKey: "", "msg": "hello"},
		map[string]string{"cc_attempt_id": "5"},
	)
	in.Triggers = map[string]*tree.Node{} // empty triggers

	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.ReSuspend {
		t.Error("plain message with no matching trigger should re-suspend")
	}
}

func TestJoinIMQueue_Resume_EmptyEvent_TriggerMatch(t *testing.T) {
	triggerNode := &tree.Node{ID: "cmd-node"}
	op := &joinQueueOp{deps: &fakeQueueDeps{}, coord: &fakeCoord{}}
	in := resumeInput("c1",
		map[string]string{ccEventKey: "", "msg": "stop"},
		nil,
	)
	in.Triggers = map[string]*tree.Node{
		"commands-stop": triggerNode,
	}

	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != triggerNode {
		t.Error("expected trigger branch to be returned")
	}
	if !out.ReenterOnResume {
		t.Error("trigger dispatch should set ReenterOnResume to keep waiting")
	}
	if out.ReSuspend {
		t.Error("trigger dispatch should not immediately re-suspend")
	}
}
