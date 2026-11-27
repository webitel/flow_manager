package im

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/webitel/engine/pkg/wbt"
	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/model"
)

const ownerSessionID = testThreadID + ".1"

// memSessionStore mirrors the flow.session statements of SQLSessionStore in memory, so
// several test nodes can share one store the way flow_manager nodes share the table.
type memSessionStore struct {
	mu          sync.Mutex
	claims      map[string]string
	seq         map[string]int
	consumed    map[string]int64
	consumedErr error
	releaseErr  error
	removeErr   error
}

func newMemSessionStore() *memSessionStore {
	return &memSessionStore{
		claims:   map[string]string{},
		seq:      map[string]int{},
		consumed: map[string]int64{},
	}
}

func (m *memSessionStore) Touch(id, appID string) (*int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if owner, ok := m.claims[id]; ok {
		if owner != appID {
			return nil, nil //nolint:nilnil // mirrors SQLSessionStore.Touch: no seq means another node holds the claim
		}

		m.seq[id]++
	} else {
		m.claims[id] = appID
		m.seq[id] = 1
	}

	seq := m.seq[id]

	return &seq, nil
}

func (m *memSessionStore) Remove(id, appID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.removeErr != nil {
		return m.removeErr
	}

	m.removeLocked(id, appID)

	return nil
}

func (m *memSessionStore) removeLocked(id, appID string) {
	if m.claims[id] == appID {
		delete(m.claims, id)
		delete(m.seq, id)
	}
}

func consumedKey(id, kind string) string { return id + "|" + kind }

func (m *memSessionStore) Release(id, appID string, consumedUpTo model.IMConsumedUpTo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.releaseErr != nil {
		return m.releaseErr
	}

	for kind, upTo := range map[string]int64{
		model.IMEventTypeMessage:  consumedUpTo.Messages,
		model.IMEventTypeCallback: consumedUpTo.Callbacks,
	} {
		if upTo > 0 {
			m.consumed[consumedKey(id, kind)] = max(m.consumed[consumedKey(id, kind)], upTo)
		}
	}

	m.removeLocked(id, appID)

	return nil
}

func (m *memSessionStore) Consumed(id, kind string, at int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.consumedErr != nil {
		return false, m.consumedErr
	}

	upTo, ok := m.consumed[consumedKey(id, kind)]

	return ok && upTo >= at, nil
}

func (m *memSessionStore) PruneConsumed(time.Duration) error { return nil }

func (m *memSessionStore) RemoveByThread(threadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id := range m.claims {
		if strings.HasPrefix(id, threadID+".") {
			delete(m.claims, id)
		}
	}

	for id := range m.consumed {
		if strings.HasPrefix(id, threadID+".") {
			delete(m.consumed, id)
		}
	}

	return nil
}

func (m *memSessionStore) RemoveAll(appID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, owner := range m.claims {
		if owner == appID {
			delete(m.claims, id)
		}
	}

	return nil
}

func (m *memSessionStore) claimOf(id string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	owner, ok := m.claims[id]

	return owner, ok
}

func (m *memSessionStore) consumedOf(id string) (model.IMConsumedUpTo, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	upTo := model.IMConsumedUpTo{
		Messages:  m.consumed[consumedKey(id, model.IMEventTypeMessage)],
		Callbacks: m.consumed[consumedKey(id, model.IMEventTypeCallback)],
	}

	return upTo, !upTo.IsZero()
}

// newNode builds one flow_manager IM node on top of a store shared with other nodes.
func newNode(tb testing.TB, id string, store SessionStore) *server {
	tb.Helper()

	log := wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
	wlog.InitGlobalLogger(log)

	return &server{
		id:              id,
		log:             log,
		connectionStore: NewConnectionStore(log),
		sessionStore:    store,
		consume:         make(chan model.Connection, 10),
		client: &Client{
			ctx:           context.Background(),
			threadService: &wbt.Client[p.ThreadManagementClient]{Api: fakeGatewayThreads{}},
		},
	}
}

func messageFrom(sub, id string, createdAt int64) model.IMEventWrapper {
	return model.MessageWrapper[model.Message]{
		DomainID: 1,
		Type:     model.IMEventTypeMessage,
		Message: model.Message{
			ID:        id,
			ThreadID:  testThreadID,
			DomainID:  1,
			Text:      "qa-WMSG-519",
			CreatedAt: createdAt,
			From:      model.ImEndpoint{Sub: sub, Issuer: "webitel"},
			To:        []model.ImEndpoint{{Sub: "1", Issuer: IMUserTypeBot, MemberID: "m-bot-1"}},
		},
	}
}

func customerMessageAt(id string, createdAt int64) model.IMEventWrapper {
	return messageFrom("cust-1", id, createdAt)
}

func systemMessageAt(id string, createdAt int64) model.IMEventWrapper {
	msg := messageFrom("cust-1", id, createdAt).(model.MessageWrapper[model.Message])
	msg.Message.Type = model.IMMessageTypeSystem

	return msg
}

func callbackAt(reactedAt int64) model.IMEventWrapper {
	return model.MessageWrapper[model.InteractiveCallback]{
		DomainID: 1,
		Type:     model.IMEventTypeCallback,
		Message: model.InteractiveCallback{
			ThreadID:   testThreadID,
			InReplyTo:  "menu-1",
			ButtonCode: "no",
			ReactedBy:  model.ImEndpoint{Sub: "cust-1", Issuer: "webitel"},
			Receiver:   model.ImEndpoint{Sub: "1", Issuer: IMUserTypeBot, MemberID: "m-bot-1"},
			ReactedAt:  time.UnixMilli(reactedAt),
		},
	}
}

func deliver(tb testing.TB, s *server, msg model.IMEventWrapper) []string {
	tb.Helper()

	if err := s.nodeMessage(msg); err != nil {
		tb.Fatalf("nodeMessage on %s: %v", s.id, err)
	}

	return dispatched(s.consume)
}

func liveDialog(tb testing.TB, s *server) *Connection {
	tb.Helper()

	conn, ok := s.connectionStore.Get(ownerSessionID)
	if !ok {
		tb.Fatalf("no live dialog on %s", s.id)
	}

	return conn
}

func finishNaturally(conn *Connection) {
	conn.MarkCompleted()
	conn.Stop(nil)
}

// runFinishedDialog reproduces the cluster state behind WMSG-519: node a ran the dialog,
// which received events up to lastAt and then reached its natural end; node b receives the
// same events and has not processed them yet.
func runFinishedDialog(tb testing.TB, events ...model.IMEventWrapper) (a, b *server, store *memSessionStore) {
	tb.Helper()

	store = newMemSessionStore()
	a = newNode(tb, "fm-a", store)
	b = newNode(tb, "fm-b", store)

	if got := deliver(tb, a, events[0]); !slices.Equal(got, []string{ownerSessionID}) {
		tb.Fatalf("node a dispatched %v, want the owner dialog", got)
	}

	for _, ev := range events[1:] {
		if got := deliver(tb, a, ev); len(got) != 0 {
			tb.Fatalf("event for a live dialog dispatched %v", got)
		}
	}

	finishNaturally(liveDialog(tb, a))

	if owner, ok := store.claimOf(ownerSessionID); ok {
		tb.Fatalf("claim still held by %s after the natural end", owner)
	}

	return a, b, store
}

func TestLaggingNode_DoesNotRestartFinishedDialog(t *testing.T) {
	first := customerMessageAt("msg-1", 1000)
	last := customerMessageAt("msg-2", 2000)

	tests := []struct {
		name string
		node func(a, b *server) *server
		late model.IMEventWrapper
	}{
		{name: "other node, last event", node: func(_, b *server) *server { return b }, late: last},
		{name: "other node, earlier event", node: func(_, b *server) *server { return b }, late: first},
		{name: "same node, duplicate delivery", node: func(a, _ *server) *server { return a }, late: last},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b, store := runFinishedDialog(t, first, last)
			node := tt.node(a, b)

			if got := deliver(t, node, tt.late); len(got) != 0 {
				t.Fatalf("late copy restarted the finished dialog on %s: %v", node.id, got)
			}

			if owner, ok := store.claimOf(ownerSessionID); ok {
				t.Errorf("late copy left the claim held by %s", owner)
			}

			if _, ok := node.connectionStore.Get(ownerSessionID); ok {
				t.Error("late copy left a dialog in the connection store")
			}
		})
	}
}

func TestNewMessageAfterNaturalEnd_StartsDialog(t *testing.T) {
	_, b, store := runFinishedDialog(t, customerMessageAt("msg-1", 1000), customerMessageAt("msg-2", 2000))

	if got := deliver(t, b, customerMessageAt("msg-3", 2001)); !slices.Equal(got, []string{ownerSessionID}) {
		t.Fatalf("new customer message dispatched %v, want the owner dialog", got)
	}

	if owner, _ := store.claimOf(ownerSessionID); owner != "fm-b" {
		t.Errorf("claim owner = %q, want fm-b", owner)
	}
}

func TestEventIgnoredBySchema_StillCountsAsReceived(t *testing.T) {
	_, b, _ := runFinishedDialog(t,
		customerMessageAt("msg-1", 1000),
		messageFrom("operator-1", "msg-op", 2500),
	)

	if got := deliver(t, b, messageFrom("operator-1", "msg-op", 2500)); len(got) != 0 {
		t.Fatalf("late copy of an event the dialog ignored restarted it: %v", got)
	}
}

func TestLateCallbackCopy_DoesNotRestartFinishedDialog(t *testing.T) {
	_, b, _ := runFinishedDialog(t, customerMessageAt("msg-1", 1000), callbackAt(1500))

	if got := deliver(t, b, callbackAt(1500)); len(got) != 0 {
		t.Fatalf("late callback copy restarted the finished dialog: %v", got)
	}

	if got := deliver(t, b, callbackAt(1501)); !slices.Equal(got, []string{ownerSessionID}) {
		t.Fatalf("new callback dispatched %v, want the owner dialog", got)
	}
}

func TestGrantStartAfterNaturalEnd_IsNotFiltered(t *testing.T) {
	_, b, _ := runFinishedDialog(t, customerMessageAt("msg-1", 1000))

	err := b.handleBotControlGranted(model.IMBotControlGrantedEvent{
		ThreadID: testThreadID,
		DomainID: 1,
		MemberID: "m-bot-1",
		Sub:      1,
		Reason:   "transfer",
	})
	if err != nil {
		t.Fatalf("handleBotControlGranted: %v", err)
	}

	if got := dispatched(b.consume); !slices.Equal(got, []string{ownerSessionID}) {
		t.Fatalf("grant dispatched %v, want the owner dialog", got)
	}
}

func TestStop_ConsumptionRecord(t *testing.T) {
	tests := []struct {
		name       string
		stop       func(c *Connection)
		wantRecord bool
	}{
		{name: "natural end", stop: finishNaturally, wantRecord: true},
		{name: "break after natural end", stop: func(c *Connection) {
			c.MarkCompleted()
			c.Break()
			c.Stop(nil)
		}, wantRecord: true},
		{name: "error", stop: func(c *Connection) {
			c.MarkCompleted()
			c.Stop(model.NewAppError("test", "test.stop", nil, "routing not found", 400))
		}},
		{name: "not completed", stop: func(c *Connection) { c.Stop(nil) }},
		{name: "break without natural end", stop: func(c *Connection) {
			c.Break()
			c.Stop(nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMemSessionStore()
			a := newNode(t, "fm-a", store)

			deliver(t, a, customerMessageAt("msg-1", 1000))
			tt.stop(liveDialog(t, a))

			upTo, ok := store.consumedOf(ownerSessionID)
			if ok != tt.wantRecord {
				t.Errorf("consumption record written = %v (%+v), want %v", ok, upTo, tt.wantRecord)
			}

			if owner, held := store.claimOf(ownerSessionID); held {
				t.Errorf("claim still held by %s", owner)
			}
		})
	}
}

func TestReleaseFailure_FallsBackToRemove(t *testing.T) {
	store := newMemSessionStore()
	store.releaseErr = errors.New("test: release failed")
	a := newNode(t, "fm-a", store)

	deliver(t, a, customerMessageAt("msg-1", 1000))
	finishNaturally(liveDialog(t, a))

	if owner, ok := store.claimOf(ownerSessionID); ok {
		t.Fatalf("claim still held by %s after a failed release", owner)
	}
}

func TestSystemEventNewerThanLateCustomerMessage_DoesNotBlockIt(t *testing.T) {
	_, b, _ := runFinishedDialog(t, customerMessageAt("msg-1", 1000), systemMessageAt("msg-sys", 3000))

	if got := deliver(t, b, customerMessageAt("msg-2", 2000)); !slices.Equal(got, []string{ownerSessionID}) {
		t.Fatalf("customer message delivered after the end dispatched %v, want the owner dialog", got)
	}
}

func TestConsumedCheckError_StartsDialog(t *testing.T) {
	_, b, store := runFinishedDialog(t, customerMessageAt("msg-1", 1000))
	store.consumedErr = errors.New("test: store unavailable")

	if got := deliver(t, b, customerMessageAt("msg-1", 1000)); !slices.Equal(got, []string{ownerSessionID}) {
		t.Fatalf("dispatched %v, want the dialog started when the check fails", got)
	}
}

func TestConsumedEvent_RemoveFailureStillSkips(t *testing.T) {
	_, b, store := runFinishedDialog(t, customerMessageAt("msg-1", 1000))
	store.removeErr = errors.New("test: remove failed")

	if got := deliver(t, b, customerMessageAt("msg-1", 1000)); len(got) != 0 {
		t.Fatalf("late copy restarted the finished dialog when giving back the claim failed: %v", got)
	}
}

func TestRemoveByThread_DropsConsumptionRecord(t *testing.T) {
	_, b, store := runFinishedDialog(t, customerMessageAt("msg-1", 1000))

	if err := store.RemoveByThread(testThreadID); err != nil {
		t.Fatalf("RemoveByThread: %v", err)
	}

	if _, ok := store.consumedOf(ownerSessionID); ok {
		t.Fatal("consumption record survived RemoveByThread")
	}

	if got := deliver(t, b, customerMessageAt("msg-1", 1000)); !slices.Equal(got, []string{ownerSessionID}) {
		t.Fatalf("dispatched %v, want a start once the thread records are cleared", got)
	}
}

func TestLateMessageOlderThanObservedCallback_StartsDialog(t *testing.T) {
	_, b, _ := runFinishedDialog(t, customerMessageAt("msg-1", 1000), callbackAt(1500))

	if got := deliver(t, b, customerMessageAt("msg-2", 1490)); !slices.Equal(got, []string{ownerSessionID}) {
		t.Fatalf("message the dialog never received dispatched %v, want the owner dialog", got)
	}
}

func TestEventAfterStop_IsNeitherFedNorCounted(t *testing.T) {
	store := newMemSessionStore()
	a := newNode(t, "fm-a", store)

	deliver(t, a, customerMessageAt("msg-1", 1000))
	conn := liveDialog(t, a)
	finishNaturally(conn)

	conn.OnMessage(customerMessageAt("msg-late", 5000))

	if got := conn.LastMessage().ID; got != "msg-1" {
		t.Errorf("last message = %q, want msg-1: an event was fed to a stopped dialog", got)
	}

	if got := conn.observed.Messages; got != 1000 {
		t.Errorf("observed messages = %d, want 1000", got)
	}
}

func TestObserved_NeverMovesBack(t *testing.T) {
	s := newNode(t, "fm-a", newMemSessionStore())
	conn := newConnection(s, ownerSessionID, model.ImEndpoint{Sub: "1", Issuer: IMUserTypeBot, MemberID: "m-bot-1"}, customerMessageAt("msg-1", 1000), false)

	conn.OnMessage(customerMessageAt("msg-3", 3000))
	conn.OnMessage(customerMessageAt("msg-2", 2000))
	conn.OnMessage(callbackAt(2500))
	conn.OnMessage(callbackAt(1500))

	want := model.IMConsumedUpTo{Messages: 3000, Callbacks: 2500}
	if conn.observed != want {
		t.Errorf("observed = %+v, want %+v", conn.observed, want)
	}
}

func TestEventMark(t *testing.T) {
	tests := []struct {
		name     string
		msg      model.IMEventWrapper
		wantKind string
		wantAt   int64
	}{
		{name: "message", msg: customerMessageAt("msg-1", 1234), wantKind: model.IMEventTypeMessage, wantAt: 1234},
		{name: "callback", msg: callbackAt(5678), wantKind: model.IMEventTypeCallback, wantAt: 5678},
		{name: "system message", msg: systemMessageAt("msg-sys", 9999)},
		{name: "callback without time", msg: model.MessageWrapper[model.InteractiveCallback]{
			Type:    model.IMEventTypeCallback,
			Message: model.InteractiveCallback{ThreadID: testThreadID},
		}},
		{name: "grant start", msg: model.MessageWrapper[model.Message]{
			Type:    model.IMEventTypeMessage,
			Message: model.Message{ThreadID: testThreadID},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, at := eventMark(tt.msg)
			if kind != tt.wantKind || at != tt.wantAt {
				t.Errorf("eventMark = (%q, %d), want (%q, %d)", kind, at, tt.wantKind, tt.wantAt)
			}
		})
	}
}
