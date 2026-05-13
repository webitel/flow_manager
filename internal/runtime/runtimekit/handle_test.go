package runtimekit_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/webitel/wlog"

	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ─── fakes ──────────────────────────────────────────────────────────────────

// fakeConn satisfies flow.Connection but NOT sessionmgr.Connection.
type fakeConn struct {
	id   string
	vars map[string]string
	ctx  context.Context
}

func newFakeConn(id string) *fakeConn {
	return &fakeConn{id: id, vars: map[string]string{}, ctx: context.Background()}
}

func (c *fakeConn) Id() string                   { return c.id }
func (c *fakeConn) Context() context.Context     { return c.ctx }
func (c *fakeConn) Variables() map[string]string { return c.vars }
func (c *fakeConn) Type() flow.ConnectionType   { return 0 }
func (c *fakeConn) NodeId() string               { return "" }
func (c *fakeConn) DomainId() int64              { return 0 }
func (c *fakeConn) Get(_ string) (string, bool)  { return "", false }
func (c *fakeConn) Set(_ context.Context, _ flow.Variables) (flow.Response, error) {
	return nil, nil
}
func (c *fakeConn) ParseText(text string, _ ...flow.ParseOption) string { return text }
func (c *fakeConn) Close() error                                         { return nil }
func (c *fakeConn) Log() *wlog.Logger                                    { return testLogger() }

// fakeConnInbound embeds fakeConn and adds OnInboundMessage, satisfying
// sessionmgr.Connection.
type fakeConnInbound struct {
	fakeConn
}

func newFakeConnInbound(id string) *fakeConnInbound {
	return &fakeConnInbound{fakeConn: fakeConn{id: id, vars: map[string]string{}, ctx: context.Background()}}
}

func (c *fakeConnInbound) OnInboundMessage(_ func(string)) func() {
	return func() {}
}

// fakeRepo implements persistence.Repository. Only Create is functional.
type fakeRepo struct {
	createErr error
	created   *persistence.Record
}

func (r *fakeRepo) Create(_ context.Context, rec *persistence.Record) error {
	if r.createErr != nil {
		return r.createErr
	}
	rec.ID = uuid.New()
	r.created = rec
	return nil
}
func (r *fakeRepo) Load(_ context.Context, _ uuid.UUID) (*persistence.Record, error) {
	return nil, nil
}
func (r *fakeRepo) LoadByResumeKey(_ context.Context, _ string) (*persistence.Record, error) {
	return nil, nil
}
func (r *fakeRepo) LoadByConnectionID(_ context.Context, _ string) (*persistence.Record, error) {
	return nil, nil
}
func (r *fakeRepo) Update(_ context.Context, _ *persistence.Record) error  { return nil }
func (r *fakeRepo) Suspend(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *fakeRepo) Complete(_ context.Context, _ uuid.UUID) error          { return nil }
func (r *fakeRepo) Fail(_ context.Context, _ uuid.UUID, _ string) error    { return nil }
func (r *fakeRepo) Touch(_ context.Context, _ string) error                { return nil }
func (r *fakeRepo) ClaimOrphaned(_ context.Context, _ string, _ time.Duration) ([]*persistence.Record, error) {
	return nil, nil
}
func (r *fakeRepo) ClaimTimerExpired(_ context.Context, _ int16, _ string) ([]*persistence.Record, error) {
	return nil, nil
}

// fakeRunner implements FlowRunner. afterStatus is applied to rec.Status when
// non-empty; simulates the driver completing or suspending the flow.
type fakeRunner struct {
	afterStatus state.Status
	runErr      error
	called      atomic.Bool
}

func (r *fakeRunner) Run(_ context.Context, rec *persistence.Record, _ *tree.Tree, _ map[string]string) error {
	r.called.Store(true)
	if r.afterStatus != "" {
		rec.Status = r.afterStatus
	}
	return r.runErr
}

// fakeWatcher implements SessionWatcher and records Watch invocations.
type watchCall struct {
	rec        *persistence.Record
	initialMsg string
}

type fakeWatcher struct {
	calls []watchCall
}

func (w *fakeWatcher) Watch(
	_ sessionmgr.Connection,
	rec *persistence.Record,
	initialMsg string,
	_ sessionmgr.ContextDecorator,
	_ sessionmgr.TeardownFunc,
) {
	w.calls = append(w.calls, watchCall{rec: rec, initialMsg: initialMsg})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func testLogger() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}

func minimalTree() *tree.Tree { return &tree.Tree{} }

func baseConfig(conn flow.Connection, repo *fakeRepo, runner *fakeRunner, watcher *fakeWatcher) runtimekit.HandleConfig {
	return runtimekit.HandleConfig{
		ChannelName: "test",
		ChannelType: 9,
		Conn:        conn,
		Tr:          minimalTree(),
		Tags:        map[string]string{},
		SchemaID:    42,
		DomainID:    1,
		AppID:       "app-1",
		Repo:        repo,
		Driver:      runner,
		SessionMgr:  watcher,
		Decorator:   func(ctx context.Context) context.Context { return ctx },
		Teardown:    func() {},
		Log:         testLogger(),
	}
}

// teardownCounter wraps a bool pointer into a closure for tracking calls.
func teardownTracker() (func(), *int32) {
	var n int32
	return func() { atomic.AddInt32(&n, 1) }, &n
}

// ─── tests ───────────────────────────────────────────────────────────────────

// conn does not implement sessionmgr.Connection → teardown called, no watch.
func TestRunSession_NoSessionmgrConnection(t *testing.T) {
	conn := newFakeConn("c1")
	repo := &fakeRepo{}
	runner := &fakeRunner{}
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Teardown = td

	watching, err := runtimekit.RunSession(nil, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if watching {
		t.Fatal("expected watching=false")
	}
	if atomic.LoadInt32(n) != 1 {
		t.Fatal("teardown must be called exactly once")
	}
	if runner.called.Load() {
		t.Fatal("driver must not run when connection fails sessionmgr.Connection assertion")
	}
	if len(watcher.calls) != 0 {
		t.Fatal("Watch must not be called")
	}
}

// rec is already suspended → Watch called with initialMsg from conn variables.
func TestRunSession_SuspendedRecovery(t *testing.T) {
	conn := newFakeConnInbound("c2")
	conn.vars[chatdomain.ConversationStartMessageVariable] = "hello"

	repo := &fakeRepo{}
	runner := &fakeRunner{}
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn
	cfg.Teardown = td

	rec := &persistence.Record{
		ID:           uuid.New(),
		ConnectionID: "c2",
		Status:       state.StatusSuspended,
		State:        state.ExecState{},
	}

	watching, err := runtimekit.RunSession(rec, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !watching {
		t.Fatal("expected watching=true")
	}
	if atomic.LoadInt32(n) != 0 {
		t.Fatal("teardown must NOT be called — sessionmgr owns it")
	}
	if runner.called.Load() {
		t.Fatal("driver must not run on recovery path")
	}
	if len(watcher.calls) != 1 {
		t.Fatalf("expected 1 Watch call, got %d", len(watcher.calls))
	}
	if watcher.calls[0].initialMsg != "hello" {
		t.Fatalf("expected initialMsg='hello', got %q", watcher.calls[0].initialMsg)
	}
}

// Fresh start: Repo.Create succeeds, driver completes → Teardown called.
func TestRunSession_FreshStart_Completes(t *testing.T) {
	conn := newFakeConnInbound("c3")
	repo := &fakeRepo{}
	runner := &fakeRunner{} // afterStatus=0 → rec.Status stays StatusRunning (set by RunSession)
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn
	cfg.Teardown = td

	watching, err := runtimekit.RunSession(nil, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if watching {
		t.Fatal("expected watching=false")
	}
	if atomic.LoadInt32(n) != 1 {
		t.Fatal("teardown must be called once")
	}
	if !runner.called.Load() {
		t.Fatal("driver must run for fresh start")
	}
	if repo.created == nil {
		t.Fatal("Repo.Create must be called")
	}
	if repo.created.SchemaID != 42 || repo.created.Channel != 9 || repo.created.AppID != "app-1" {
		t.Fatalf("record fields mismatch: %+v", repo.created)
	}
	if len(watcher.calls) != 0 {
		t.Fatal("Watch must not be called")
	}
}

// Fresh start: driver suspends the record → Watch called, teardown deferred.
func TestRunSession_FreshStart_Suspends(t *testing.T) {
	conn := newFakeConnInbound("c4")
	repo := &fakeRepo{}
	runner := &fakeRunner{afterStatus: state.StatusSuspended}
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn
	cfg.Teardown = td

	watching, err := runtimekit.RunSession(nil, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !watching {
		t.Fatal("expected watching=true")
	}
	if atomic.LoadInt32(n) != 0 {
		t.Fatal("teardown must NOT be called")
	}
	if len(watcher.calls) != 1 {
		t.Fatalf("expected 1 Watch call, got %d", len(watcher.calls))
	}
	if watcher.calls[0].initialMsg != "" {
		t.Fatalf("expected empty initialMsg for mid-run suspend, got %q", watcher.calls[0].initialMsg)
	}
}

// Repo.Create fails → returns error, teardown NOT called.
func TestRunSession_FreshStart_CreateFails(t *testing.T) {
	conn := newFakeConnInbound("c5")
	createErr := errors.New("db down")
	repo := &fakeRepo{createErr: createErr}
	runner := &fakeRunner{}
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn
	cfg.Teardown = td

	watching, err := runtimekit.RunSession(nil, cfg)

	if !errors.Is(err, createErr) {
		t.Fatalf("expected createErr, got %v", err)
	}
	if watching {
		t.Fatal("expected watching=false")
	}
	if atomic.LoadInt32(n) != 0 {
		t.Fatal("teardown must NOT be called on create failure — caller owns connection stop")
	}
	if runner.called.Load() {
		t.Fatal("driver must not run when Create fails")
	}
}

// Existing running record: driver completes → Teardown called, no Watch.
func TestRunSession_ExistingRecord_Completes(t *testing.T) {
	conn := newFakeConnInbound("c6")
	repo := &fakeRepo{}
	runner := &fakeRunner{} // stays running
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn
	cfg.Teardown = td

	rec := &persistence.Record{
		ID:           uuid.New(),
		ConnectionID: "c6",
		Status:       state.StatusRunning,
		State:        state.ExecState{},
	}

	watching, err := runtimekit.RunSession(rec, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if watching {
		t.Fatal("expected watching=false")
	}
	if atomic.LoadInt32(n) != 1 {
		t.Fatal("teardown must be called once")
	}
	if !runner.called.Load() {
		t.Fatal("driver must run")
	}
	if repo.created != nil {
		t.Fatal("Repo.Create must NOT be called for existing record")
	}
}

// Existing running record: driver suspends → Watch called, teardown deferred.
func TestRunSession_ExistingRecord_Suspends(t *testing.T) {
	conn := newFakeConnInbound("c7")
	repo := &fakeRepo{}
	runner := &fakeRunner{afterStatus: state.StatusSuspended}
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn
	cfg.Teardown = td

	rec := &persistence.Record{
		ID:           uuid.New(),
		ConnectionID: "c7",
		Status:       state.StatusRunning,
		State:        state.ExecState{},
	}

	watching, err := runtimekit.RunSession(rec, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !watching {
		t.Fatal("expected watching=true")
	}
	if atomic.LoadInt32(n) != 0 {
		t.Fatal("teardown must NOT be called")
	}
	if len(watcher.calls) != 1 {
		t.Fatalf("expected 1 Watch call, got %d", len(watcher.calls))
	}
}

// Nil Decorator must not panic; flow runs normally.
func TestRunSession_NilDecorator(t *testing.T) {
	conn := newFakeConnInbound("c8")
	repo := &fakeRepo{}
	runner := &fakeRunner{}
	watcher := &fakeWatcher{}

	td, n := teardownTracker()
	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn
	cfg.Decorator = nil // explicitly nil
	cfg.Teardown = td

	watching, err := runtimekit.RunSession(nil, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if watching {
		t.Fatal("expected watching=false")
	}
	if atomic.LoadInt32(n) != 1 {
		t.Fatal("teardown must be called")
	}
}

// Connection variables are seeded into the ExecState for fresh-start records.
func TestRunSession_FreshStart_SeedsVariables(t *testing.T) {
	conn := newFakeConnInbound("c9")
	conn.vars["foo"] = "bar"
	conn.vars["baz"] = "qux"

	repo := &fakeRepo{}
	runner := &fakeRunner{}
	watcher := &fakeWatcher{}

	cfg := baseConfig(conn, repo, runner, watcher)
	cfg.Conn = conn

	runtimekit.RunSession(nil, cfg)

	if repo.created == nil {
		t.Fatal("Repo.Create must be called")
	}
	es := repo.created.State
	if es.Variables["foo"] != "bar" || es.Variables["baz"] != "qux" {
		t.Fatalf("expected conn variables in ExecState, got %v", es.Variables)
	}
}
