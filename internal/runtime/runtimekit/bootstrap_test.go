package runtimekit_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/webitel/wlog"

	contacts2 "github.com/webitel/flow_manager/api/gen/contacts"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	"github.com/webitel/flow_manager/internal/domain/email"
	"github.com/webitel/flow_manager/internal/domain/files"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/domain/notification"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/domain/routing"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/internal/storage"
)

// ─── fakeBootstrapDeps ────────────────────────────────────────────────────────

// fakeBootstrapDeps satisfies runtimekit.BootstrapDeps with no-op zero values.
// None of the op methods are called during Bootstrap itself (they are stored as
// closure references and only invoked when a flow actually executes).
type fakeBootstrapDeps struct {
	repo persistence.Repository
	log  *wlog.Logger
}

func newFakeBootstrapDeps() *fakeBootstrapDeps {
	return &fakeBootstrapDeps{
		repo: &fakeRepo{},
		log:  wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false}),
	}
}

// ── BootstrapDeps direct methods ─────────────────────────────────────────────

func (d *fakeBootstrapDeps) GetLocation(_ int) *time.Location { return time.UTC }
func (d *fakeBootstrapDeps) GetStore() storage.Store          { return &fakeStore{} }
func (d *fakeBootstrapDeps) GetSchemaById(_ int64, _ int) (*routing.Schema, error) {
	return nil, nil
}
func (d *fakeBootstrapDeps) Meeting() domainmeeting.Client            { return nil }
func (d *fakeBootstrapDeps) Cases() domcases.Client                   { return nil }
func (d *fakeBootstrapDeps) RuntimeStateRepo() persistence.Repository { return d.repo }
func (d *fakeBootstrapDeps) Log() *wlog.Logger                        { return d.log }
func (d *fakeBootstrapDeps) SchemaVariable(_ context.Context, _ int64, _ string) string {
	return ""
}

// ── builtin.CookieCache ───────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) GetCookieCache(_ context.Context, _ int64, _ string) (string, error) {
	return "", nil
}
func (d *fakeBootstrapDeps) SetCookieCache(_ context.Context, _ int64, _, _ string, _ int64) error {
	return nil
}

// ── builtin.GlobalDeps ────────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) SetGlobalVar(_ context.Context, _ int64, _, _ string, _ bool) error {
	return nil
}

// ── builtin.ListDeps ──────────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) CheckList(_ int64, _ string, _ *int, _ *string) (bool, error) {
	return false, nil
}
func (d *fakeBootstrapDeps) AddToList(_ context.Context, _ int64, _ *int, _ *string, _ string, _ *string, _ int64) error {
	return nil
}

// ── builtin.CacheDeps ────────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) CacheGet(_ context.Context, _ string, _ int64, _ string) (string, error) {
	return "", nil
}
func (d *fakeBootstrapDeps) CacheSet(_ context.Context, _ string, _ int64, _, _ string, _ int64) error {
	return nil
}
func (d *fakeBootstrapDeps) CacheDelete(_ context.Context, _ string, _ int64, _ string) error {
	return nil
}

// ── builtin.GenerateLinkDeps ─────────────────────────────────────────────────

func (d *fakeBootstrapDeps) GeneratePreSignedLink(_ context.Context, _, _ string, _, _ int64, _ map[string]string) (string, error) {
	return "", nil
}

// ── builtin.OpenLinkDeps ─────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) PushOpenLink(_ int64, _ string, _ int64, _, _ string) error {
	return nil
}

// ── builtin.SqlDeps ───────────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) SqlQuery(_ context.Context, _, _, _ string, _ []interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// ── emailop.EmailDeps ─────────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) SmtpSettings(_ int64, _ *queue.SearchEntity) (*email.SmtSettings, error) {
	return nil, nil
}
func (d *fakeBootstrapDeps) SmtpSettingsOAuthToken(_ *email.SmtSettings) (string, error) {
	return "", nil
}
func (d *fakeBootstrapDeps) GetFileMetadata(_ int64, _ []int64) ([]files.File, error) {
	return nil, nil
}
func (d *fakeBootstrapDeps) DownloadFile(_ int64, _ int64) (io.ReadCloser, error) {
	return nil, nil
}
func (d *fakeBootstrapDeps) SaveEmail(_ int64, _ *email.Email) error { return nil }

// ── contactsop.LinkDeps ───────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) CallSetContactId(_ int64, _ string, _ int64) error      { return nil }
func (d *fakeBootstrapDeps) ContactLinkToChat(_ context.Context, _, _ string) error { return nil }
func (d *fakeBootstrapDeps) MailSetContacts(_ context.Context, _ int64, _ string, _ []int64) error {
	return nil
}

// ── notifop.Deps ──────────────────────────────────────────────────────────────

func (d *fakeBootstrapDeps) UserNotification(_ notification.Notification) {}

// ─── fakeStore ────────────────────────────────────────────────────────────────

// fakeStore satisfies storage.Store; all sub-stores are nil — Bootstrap only
// passes them as references to op constructors, never calls their methods.
type fakeStore struct{}

func (s *fakeStore) Call() storage.CallStore                   { return nil }
func (s *fakeStore) Schema() storage.SchemaStore               { return nil }
func (s *fakeStore) CallRouting() storage.CallRoutingStore     { return nil }
func (s *fakeStore) Endpoint() storage.EndpointStore           { return nil }
func (s *fakeStore) Email() storage.EmailStore                 { return nil }
func (s *fakeStore) Media() storage.MediaStore                 { return nil }
func (s *fakeStore) Calendar() storage.CalendarStore           { return nil }
func (s *fakeStore) List() storage.ListStore                   { return nil }
func (s *fakeStore) Chat() storage.ChatStore                   { return nil }
func (s *fakeStore) Queue() storage.QueueStore                 { return nil }
func (s *fakeStore) Member() storage.MemberStore               { return nil }
func (s *fakeStore) User() storage.UserStore                   { return nil }
func (s *fakeStore) Log() storage.LogStore                     { return nil }
func (s *fakeStore) File() storage.FileStore                   { return nil }
func (s *fakeStore) WebHook() storage.WebHookStore             { return nil }
func (s *fakeStore) SystemcSettings() storage.SystemcSettings  { return nil }
func (s *fakeStore) SocketSession() storage.SocketSessionStore { return nil }
func (s *fakeStore) Session() storage.SessionStore             { return nil }

// ─── fakeContactsClient ───────────────────────────────────────────────────────

type fakeContactsClient struct{}

func (fakeContactsClient) Create(_ context.Context, _ string, _ *contacts2.InputContactRequest) (*contacts2.Contact, error) {
	return nil, nil
}
func (fakeContactsClient) Locate(_ context.Context, _ string, _ *contacts2.LocateContactRequest) (*contacts2.Contact, error) {
	return nil, nil
}
func (fakeContactsClient) Search(_ context.Context, _ string, _ *contacts2.SearchContactsRequest) (*contacts2.ContactList, error) {
	return nil, nil
}
func (fakeContactsClient) SearchNA(_ context.Context, _ *contacts2.SearchContactsNARequest) (*contacts2.ContactList, error) {
	return nil, nil
}
func (fakeContactsClient) Update(_ context.Context, _ string, _ *contacts2.InputContactRequest) (*contacts2.Contact, error) {
	return nil, nil
}
func (fakeContactsClient) MergeVariables(_ context.Context, _ string, _ *contacts2.MergeVariablesRequest) (*contacts2.VariableList, error) {
	return nil, nil
}
func (fakeContactsClient) MergePhones(_ context.Context, _ string, _ *contacts2.MergePhonesRequest) (*contacts2.PhoneList, error) {
	return nil, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func minimalBootstrapConfig(deps *fakeBootstrapDeps) runtimekit.Config {
	return runtimekit.Config{
		Deps:     deps,
		LoadTree: func(_ context.Context, _ int64, _ int) (*tree.Tree, error) { return nil, nil },
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestBootstrap_ReturnsNonNilKit(t *testing.T) {
	kit := runtimekit.Bootstrap(minimalBootstrapConfig(newFakeBootstrapDeps()))
	if kit == nil {
		t.Fatal("Bootstrap returned nil Kit")
	}
	if kit.Driver == nil {
		t.Fatal("Kit.Driver is nil")
	}
	if kit.Coord == nil {
		t.Fatal("Kit.Coord is nil")
	}
}

func TestBootstrap_ExtraOps_Called(t *testing.T) {
	called := false
	cfg := minimalBootstrapConfig(newFakeBootstrapDeps())
	cfg.ExtraOps = func(_ *ops.Registry) { called = true }

	runtimekit.Bootstrap(cfg)

	if !called {
		t.Fatal("ExtraOps was not called")
	}
}

func TestBootstrap_ExtraOps_Nil_DoesNotPanic(t *testing.T) {
	cfg := minimalBootstrapConfig(newFakeBootstrapDeps())
	cfg.ExtraOps = nil
	runtimekit.Bootstrap(cfg) // must not panic
}

func TestBootstrap_ExtraOps_ReceivesPopulatedRegistry(t *testing.T) {
	// ExtraOps is called after builtins are already registered.
	// Verify "if" (a core builtin) is present when ExtraOps runs.
	cfg := minimalBootstrapConfig(newFakeBootstrapDeps())
	cfg.ExtraOps = func(reg *ops.Registry) {
		if reg.Get("if") == nil {
			t.Error("builtin 'if' not in registry when ExtraOps is called")
		}
	}
	runtimekit.Bootstrap(cfg)
}

func TestBootstrap_ExtraOps_CanRegisterOp(t *testing.T) {
	// An op registered in ExtraOps must be reachable by the Driver.
	var opCalled bool
	probe := &probeOp{called: &opCalled}

	cfg := minimalBootstrapConfig(newFakeBootstrapDeps())
	cfg.ExtraOps = func(reg *ops.Registry) {
		reg.Register("testProbe", probe)
	}
	cfg.LoadTree = func(_ context.Context, _ int64, _ int) (*tree.Tree, error) {
		return tree.ParseJSON(0, []byte(`[{"testProbe": {}}]`))
	}

	kit := runtimekit.Bootstrap(cfg)

	tr, err := cfg.LoadTree(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("parse tree: %v", err)
	}
	rec := &persistence.Record{
		SchemaID: 0,
		Status:   state.StatusRunning,
		State: state.ExecState{
			Stack:     []state.Frame{{NodeID: "root", Position: 0}},
			Variables: map[string]string{},
		},
	}
	if runErr := kit.Driver.Run(context.Background(), rec, tr, nil); runErr != nil {
		t.Fatalf("Driver.Run: %v", runErr)
	}
	if !opCalled {
		t.Fatal("testProbe op was never called — ExtraOps registration did not reach the Driver")
	}
}

func TestBootstrap_ContactsClient_Nil_DoesNotPanic(t *testing.T) {
	cfg := minimalBootstrapConfig(newFakeBootstrapDeps())
	cfg.ContactsClient = nil
	runtimekit.Bootstrap(cfg) // must not panic
}

func TestBootstrap_ContactsClient_NonNil_DoesNotPanic(t *testing.T) {
	cfg := minimalBootstrapConfig(newFakeBootstrapDeps())
	cfg.ContactsClient = fakeContactsClient{}
	runtimekit.Bootstrap(cfg) // must not panic
}

func TestBootstrap_DeterministicIDs(t *testing.T) {
	// Two calls with the same config must return independent Kits (no shared
	// registry state — each Bootstrap creates its own ops.Registry).
	deps := newFakeBootstrapDeps()
	cfg := minimalBootstrapConfig(deps)

	kit1 := runtimekit.Bootstrap(cfg)
	kit2 := runtimekit.Bootstrap(cfg)

	if kit1 == kit2 {
		t.Error("Bootstrap must return a new Kit on each call")
	}
	if kit1.Driver == kit2.Driver {
		t.Error("each Kit must have its own Driver instance")
	}
}

// ─── probeOp ─────────────────────────────────────────────────────────────────

type probeOp struct{ called *bool }

func (o *probeOp) Kind() ops.OpKind { return ops.OpKindSync }
func (o *probeOp) Execute(_ context.Context, _ ops.OpInput) (ops.OpOutput, error) {
	*o.called = true
	return ops.OpOutput{}, nil
}
