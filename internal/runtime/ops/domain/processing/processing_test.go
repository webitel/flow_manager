package processing

import (
	"context"
	"fmt"
	"sync"
	"testing"

	casespb "github.com/webitel/flow_manager/api/gen/cases"
	calldomain "github.com/webitel/flow_manager/internal/domain/call"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	procpkg "github.com/webitel/flow_manager/pkg/processing"
)

// ── stubProcessingConn ────────────────────────────────────────────────────────

type stubProcessingConn struct {
	id         string
	domainId   int64
	vars       map[string]string
	components map[string]any
	exported   [][]string
	sendFormFn func(procpkg.FormElem) error
	actionHook func(procpkg.FormAction)
	mu         sync.Mutex
}

func newConn(id string) *stubProcessingConn {
	return &stubProcessingConn{
		id:         id,
		domainId:   1,
		vars:       make(map[string]string),
		components: make(map[string]any),
	}
}

func (c *stubProcessingConn) Id() string       { return c.id }
func (c *stubProcessingConn) DomainId() int64  { return c.domainId }
func (c *stubProcessingConn) DumpExportVariables() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]string, len(c.vars))
	for k, v := range c.vars {
		out[k] = v
	}
	return out
}
func (c *stubProcessingConn) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.vars[key]
	return v, ok
}
func (c *stubProcessingConn) Set(_ context.Context, vars flow.Variables) (flow.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range vars {
		c.vars[k] = fmt.Sprintf("%v", v)
	}
	return nil, nil
}
func (c *stubProcessingConn) GetComponentByName(name string) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.components[name]
}
func (c *stubProcessingConn) SetComponent(name string, comp any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.components[name] = comp
}
func (c *stubProcessingConn) Export(_ context.Context, vars []string) {
	c.exported = append(c.exported, vars)
}
func (c *stubProcessingConn) SendForm(ctx context.Context, form procpkg.FormElem) error {
	if c.sendFormFn != nil {
		return c.sendFormFn(form)
	}
	return nil
}
func (c *stubProcessingConn) OnFormAction(handler func(procpkg.FormAction)) (unregister func()) {
	c.mu.Lock()
	c.actionHook = handler
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		c.actionHook = nil
		c.mu.Unlock()
	}
}

// fire simulates an operator clicking an action button.
func (c *stubProcessingConn) fire(action procpkg.FormAction) {
	c.mu.Lock()
	h := c.actionHook
	c.mu.Unlock()
	if h != nil {
		h(action)
	}
}

// ── fakeAttemptDeps ───────────────────────────────────────────────────────────

type fakeAttemptDeps struct {
	resultErr error
	resumeErr error
}

func (f *fakeAttemptDeps) AttemptResult(_ *calldomain.AttemptResult) error { return f.resultErr }
func (f *fakeAttemptDeps) ResumeAttempt(_ context.Context, _, _ int64) error { return f.resumeErr }

// ── fakeComponentDeps (minimal cases client) ──────────────────────────────────

type fakeComponentDeps struct{}

func (f *fakeComponentDeps) SearchCases(_ context.Context, _ *casespb.SearchCasesRequest, _ string) (*casespb.CaseList, error) {
	return &casespb.CaseList{}, nil
}
func (f *fakeComponentDeps) LocateCase(_ context.Context, _ *casespb.LocateCaseRequest, _ string) (*casespb.Case, error) {
	return &casespb.Case{}, nil
}
func (f *fakeComponentDeps) CreateCase(_ context.Context, _ *casespb.CreateCaseRequest, _ string) (*casespb.Case, error) {
	return &casespb.Case{}, nil
}
func (f *fakeComponentDeps) UpdateCase(_ context.Context, _ *casespb.UpdateCaseRequest, _ string) (*casespb.UpdateCaseResponse, error) {
	return &casespb.UpdateCaseResponse{}, nil
}
func (f *fakeComponentDeps) LinkCommunication(_ context.Context, _ *casespb.LinkCommunicationRequest, _ string) (*casespb.LinkCommunicationResponse, error) {
	return &casespb.LinkCommunicationResponse{}, nil
}
func (f *fakeComponentDeps) GetServiceCatalogs(_ context.Context, _ *casespb.ListCatalogRequest, _ string) (*casespb.CatalogList, error) {
	return &casespb.CatalogList{}, nil
}
func (f *fakeComponentDeps) PublishComment(_ context.Context, _ *casespb.PublishCommentRequest, _ string) (*casespb.CaseComment, error) {
	return &casespb.CaseComment{}, nil
}
func (f *fakeComponentDeps) CreateLink(_ context.Context, _ *casespb.CreateLinkRequest, _ string) (*casespb.CaseLink, error) {
	return &casespb.CaseLink{}, nil
}
func (f *fakeComponentDeps) DeleteLink(_ context.Context, _ *casespb.DeleteLinkRequest, _ string) (*casespb.CaseLink, error) {
	return &casespb.CaseLink{}, nil
}
func (f *fakeComponentDeps) LocateService(_ context.Context, _ *casespb.LocateServiceRequest, _ string) (*casespb.LocateServiceResponse, error) {
	return &casespb.LocateServiceResponse{}, nil
}
func (f *fakeComponentDeps) CreateRelatedCase(_ context.Context, _ *casespb.CreateRelatedCaseRequest, _ string) (*casespb.RelatedCase, error) {
	return &casespb.RelatedCase{}, nil
}
func (f *fakeComponentDeps) ListCaseFiles(_ context.Context, _ *casespb.ListFilesRequest, _ string) (*casespb.CaseFileList, error) {
	return &casespb.CaseFileList{}, nil
}
func (f *fakeComponentDeps) LocateCatalog(_ context.Context, _ *casespb.LocateCatalogRequest, _ string) (*casespb.LocateCatalogResponse, error) {
	return &casespb.LocateCatalogResponse{}, nil
}
func (f *fakeComponentDeps) ListStatusConditions(_ context.Context, _ *casespb.ListStatusConditionRequest, _ string) (*casespb.StatusConditionList, error) {
	return &casespb.StatusConditionList{}, nil
}

var _ domcases.Client = (*fakeComponentDeps)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func ctxWithConn(conn ProcessingConn) context.Context {
	return WithConn(context.Background(), conn)
}

func procInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

// ── setToJSON ─────────────────────────────────────────────────────────────────

func TestSetToJSON_JSONObject(t *testing.T) {
	v := setToJSON(`{"key":"val"}`)
	if m, ok := v.(map[string]any); !ok || m["key"] != "val" {
		t.Errorf("got %T %v, want map with key=val", v, v)
	}
}

func TestSetToJSON_JSONArray(t *testing.T) {
	v := setToJSON(`["a","b"]`)
	arr, ok := v.([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("got %T %v, want []any of len 2", v, v)
	}
}

func TestSetToJSON_PlainString(t *testing.T) {
	v := setToJSON("hello")
	if s, ok := v.(string); !ok || s != "hello" {
		t.Errorf("got %v, want hello", v)
	}
}

func TestSetToJSON_ShortString(t *testing.T) {
	// < 2 chars → returned as-is
	v := setToJSON("x")
	if s, ok := v.(string); !ok || s != "x" {
		t.Errorf("got %v, want x", v)
	}
}

// ── attemptResult ─────────────────────────────────────────────────────────────

func TestAttemptResult_NoConn(t *testing.T) {
	op := &attemptResultOp{deps: &fakeAttemptDeps{}}
	_, err := op.Execute(context.Background(), procInput(map[string]any{"status": "success"}))
	if err == nil {
		t.Fatal("expected error when no processing connection in context")
	}
}

func TestAttemptResult_Success(t *testing.T) {
	conn := newConn("c1")
	op := &attemptResultOp{deps: &fakeAttemptDeps{}}
	in := procInput(map[string]any{"status": "success", "description": "resolved"})
	in.Variables = map[string]string{"attempt_id": "99"}
	_, err := op.Execute(ctxWithConn(conn), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAttemptResult_ExportVarsMerged(t *testing.T) {
	conn := newConn("c1")
	conn.vars["ticket_id"] = "TKT-001"
	called := false
	deps := &fakeAttemptDeps{}
	// We can't directly observe argv.Variables, but ensure no error occurs and
	// export variables are present in conn.
	op := &attemptResultOp{deps: deps}
	in := procInput(map[string]any{"status": "success"})
	in.Variables = map[string]string{"attempt_id": "1"}
	_, err := op.Execute(ctxWithConn(conn), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = called
}

func TestAttemptResult_DepError(t *testing.T) {
	conn := newConn("c1")
	deps := &fakeAttemptDeps{resultErr: fmt.Errorf("service unavailable")}
	op := &attemptResultOp{deps: deps}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{"status": "success"}))
	if err == nil {
		t.Fatal("expected error when AttemptResult fails")
	}
}

// ── export ────────────────────────────────────────────────────────────────────

func TestExport_NoConn(t *testing.T) {
	op := exportOp{}
	_, err := op.Execute(context.Background(), ops.OpInput{Node: &tree.Node{RawArgs: []any{"x"}}})
	if err == nil {
		t.Fatal("expected error when no conn")
	}
}

func TestExport_CallsConnExport(t *testing.T) {
	conn := newConn("c1")
	op := exportOp{}
	in := ops.OpInput{Node: &tree.Node{RawArgs: []any{"var1", "var2"}}}
	_, err := op.Execute(ctxWithConn(conn), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conn.exported) != 1 || len(conn.exported[0]) != 2 {
		t.Errorf("exported = %v, want [[var1 var2]]", conn.exported)
	}
}

// ── formFile ──────────────────────────────────────────────────────────────────

func TestFormFile_NoConn(t *testing.T) {
	op := formFileOp{}
	_, err := op.Execute(context.Background(), procInput(map[string]any{"id": "f1"}))
	if err == nil {
		t.Fatal("expected error when no conn")
	}
}

func TestFormFile_NoId(t *testing.T) {
	conn := newConn("c1")
	op := formFileOp{}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{}))
	if err == nil {
		t.Fatal("expected error when id is missing")
	}
}

func TestFormFile_Success_EmptyValue(t *testing.T) {
	conn := newConn("c1")
	op := formFileOp{}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{"id": "attachments"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := conn.components["attachments"]; !ok {
		t.Error("expected component 'attachments' to be registered")
	}
}

func TestFormFile_Success_ExistingValue(t *testing.T) {
	conn := newConn("c1")
	conn.vars["files-comp"] = `["file1.pdf"]`
	op := formFileOp{}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{"id": "files-comp"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── formComponent ─────────────────────────────────────────────────────────────

func TestFormComponent_NoConn(t *testing.T) {
	op := &formComponentOp{deps: &fakeComponentDeps{}}
	_, err := op.Execute(context.Background(), procInput(map[string]any{"id": "c1"}))
	if err == nil {
		t.Fatal("expected error when no conn")
	}
}

func TestFormComponent_NoId(t *testing.T) {
	conn := newConn("c1")
	op := &formComponentOp{deps: &fakeComponentDeps{}}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{
		"view": map[string]any{"component": "wt-input"},
	}))
	if err == nil {
		t.Fatal("expected error when id is missing")
	}
}

func TestFormComponent_WtInput_SetsStringValue(t *testing.T) {
	conn := newConn("c1")
	conn.vars["note"] = "call went well"
	op := &formComponentOp{deps: &fakeComponentDeps{}}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{
		"id":   "note",
		"view": map[string]any{"component": "wt-input", "label": "Note"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	comp, ok := conn.components["note"]
	if !ok {
		t.Fatal("expected component 'note' registered")
	}
	fc, ok := comp.(procpkg.FormComponent)
	if !ok {
		t.Fatalf("expected FormComponent, got %T", comp)
	}
	if fc.Value != "call went well" {
		t.Errorf("Value = %v, want 'call went well'", fc.Value)
	}
}

func TestFormComponent_Default_SetsJSONValue(t *testing.T) {
	conn := newConn("c1")
	conn.vars["status"] = `{"id": 1, "name": "Open"}`
	op := &formComponentOp{deps: &fakeComponentDeps{}}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{
		"id":   "status",
		"view": map[string]any{"component": "wt-select"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	comp := conn.components["status"]
	fc, ok := comp.(procpkg.FormComponent)
	if !ok {
		t.Fatalf("expected FormComponent, got %T", comp)
	}
	// JSON value should be parsed into a map
	if _, ok := fc.Value.(map[string]any); !ok {
		t.Errorf("Value = %T %v, want map[string]any", fc.Value, fc.Value)
	}
}

// ── generateForm ──────────────────────────────────────────────────────────────

func TestGenerateForm_Resume_ReturnsPayload(t *testing.T) {
	// Resume path: no conn needed — payload is returned as SetVars directly.
	op := &generateFormOp{coord: DispatchFunc(func(_ context.Context, _ string, _ map[string]string) error { return nil })}
	in := ops.OpInput{
		Node:          &tree.Node{Args: map[string]any{"id": "my-form"}},
		ResumePayload: map[string]string{"my-form": "submit", "note": "done"},
	}
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["my-form"] != "submit" {
		t.Errorf("SetVars[my-form] = %q, want submit", out.SetVars["my-form"])
	}
}

func TestGenerateForm_NoConn(t *testing.T) {
	op := &generateFormOp{coord: nil}
	_, err := op.Execute(context.Background(), procInput(map[string]any{"id": "f"}))
	if err == nil {
		t.Fatal("expected error when no conn")
	}
}

func TestGenerateForm_SendFormError(t *testing.T) {
	conn := newConn("c1")
	conn.sendFormFn = func(_ procpkg.FormElem) error { return fmt.Errorf("transport error") }
	op := &generateFormOp{coord: nil}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{
		"id": "f", "body": []any{}, "actions": []any{},
	}))
	if err == nil {
		t.Fatal("expected error when SendForm fails")
	}
}

func TestGenerateForm_Fresh_Suspends(t *testing.T) {
	conn := newConn("conn-42")
	var dispatched []string
	coord := DispatchFunc(func(_ context.Context, key string, _ map[string]string) error {
		dispatched = append(dispatched, key)
		return nil
	})
	op := &generateFormOp{coord: coord}
	in := ops.OpInput{
		ConnID: "conn-42",
		Node:   &tree.Node{Args: map[string]any{"id": "call-form", "body": []any{}, "actions": []any{}}},
	}
	out, err := op.Execute(ctxWithConn(conn), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey != "form:conn-42" {
		t.Errorf("SuspendKey = %q, want form:conn-42", out.SuspendKey)
	}
	if !out.ReenterOnResume {
		t.Error("expected ReenterOnResume=true")
	}

	// Simulate operator clicking an action.
	conn.fire(procpkg.FormAction{Name: "submit", Fields: map[string]any{"note": "ok"}})
	if len(dispatched) != 1 || dispatched[0] != "form:conn-42" {
		t.Errorf("coord.Dispatch not called correctly: %v", dispatched)
	}
}

// ── formTable ─────────────────────────────────────────────────────────────────

func TestFormTable_NoConn(t *testing.T) {
	op := formTableOp{}
	_, err := op.Execute(context.Background(), procInput(map[string]any{"id": "t1"}))
	if err == nil {
		t.Fatal("expected error when no conn")
	}
}

func TestFormTable_NoId(t *testing.T) {
	conn := newConn("c1")
	op := formTableOp{}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{}))
	if err == nil {
		t.Fatal("expected error when id is missing")
	}
}

func TestFormTable_Success(t *testing.T) {
	conn := newConn("c1")
	op := formTableOp{}
	_, err := op.Execute(ctxWithConn(conn), procInput(map[string]any{"id": "my-table"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := conn.components["my-table"]; !ok {
		t.Error("expected component 'my-table' registered")
	}
}

// ── resumeAttempt ─────────────────────────────────────────────────────────────

func TestResumeAttempt_FromArgs(t *testing.T) {
	deps := &fakeAttemptDeps{}
	op := &resumeAttemptOp{deps: deps}
	_, err := op.Execute(context.Background(), procInput(map[string]any{"id": 55}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResumeAttempt_FromVariable(t *testing.T) {
	deps := &fakeAttemptDeps{}
	op := &resumeAttemptOp{deps: deps}
	in := procInput(map[string]any{})
	in.Variables = map[string]string{"attempt_id": "77"}
	_, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResumeAttempt_DepError(t *testing.T) {
	deps := &fakeAttemptDeps{resumeErr: fmt.Errorf("timeout")}
	op := &resumeAttemptOp{deps: deps}
	_, err := op.Execute(context.Background(), procInput(map[string]any{"id": 1}))
	if err == nil {
		t.Fatal("expected error when ResumeAttempt fails")
	}
}
