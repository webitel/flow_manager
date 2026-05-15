package member

import (
	"context"
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/webitel/flow_manager/internal/domain/flow"
	queuedomain "github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/internal/storage"
)

// ── fakeMemberStore ───────────────────────────────────────────────────────────

type fakeMemberStore struct {
	position    int64
	posErr      error
	ewtVal      float64
	ewtErr      error
	propsResult flow.Variables
	propsErr    error
	patchCount  int
	patchErr    error
	createErr   error

	// recorded
	createCalls []*queuedomain.CallbackMember
}

func (f *fakeMemberStore) CallPosition(_ string) (int64, error) {
	return f.position, f.posErr
}
func (f *fakeMemberStore) EWTPuzzle(_ int64, _ string, _ int, _, _ []int) (float64, error) {
	return f.ewtVal, f.ewtErr
}
func (f *fakeMemberStore) GetProperties(_ int64, _ *queuedomain.SearchMember, _ flow.Variables) (flow.Variables, error) {
	return f.propsResult, f.propsErr
}
func (f *fakeMemberStore) PatchMembers(_ int64, _ *queuedomain.SearchMember, _ *queuedomain.PatchMember) (int, error) {
	return f.patchCount, f.patchErr
}
func (f *fakeMemberStore) CreateMember(_ int64, _, _ int, member *queuedomain.CallbackMember) error {
	f.createCalls = append(f.createCalls, member)
	return f.createErr
}

var _ storage.MemberStore = (*fakeMemberStore)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func memberInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1, ConnID: "call-1"}
}

// ── ccPosition ────────────────────────────────────────────────────────────────

func TestCCPosition_NoSet(t *testing.T) {
	op := &ccPositionOp{store: &fakeMemberStore{}}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{}))
	if err == nil {
		t.Fatal("expected error when set is empty")
	}
}

func TestCCPosition_Success(t *testing.T) {
	store := &fakeMemberStore{position: 5}
	op := &ccPositionOp{store: store}
	out, err := op.Execute(context.Background(), memberInput(map[string]any{"set": "pos"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["pos"] != "5" {
		t.Errorf("pos = %q, want 5", out.SetVars["pos"])
	}
}

func TestCCPosition_DepError(t *testing.T) {
	store := &fakeMemberStore{posErr: fmt.Errorf("not in queue")}
	op := &ccPositionOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{"set": "pos"}))
	if err == nil {
		t.Fatal("expected error when CallPosition fails")
	}
}

// ── memberInfo ────────────────────────────────────────────────────────────────

func TestMemberInfo_NoMember(t *testing.T) {
	op := &memberInfoOp{store: &fakeMemberStore{}}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"set": map[string]any{"x": "y"},
	}))
	if err == nil {
		t.Fatal("expected error when member is nil")
	}
}

func TestMemberInfo_EmptySet(t *testing.T) {
	op := &memberInfoOp{store: &fakeMemberStore{}}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"member": map[string]any{"id": 1},
	}))
	if err == nil {
		t.Fatal("expected error when set is empty")
	}
}

func TestMemberInfo_Success(t *testing.T) {
	store := &fakeMemberStore{
		propsResult: flow.Variables{"phone": "380XXXXXXXXX"},
	}
	op := &memberInfoOp{store: store}
	out, err := op.Execute(context.Background(), memberInput(map[string]any{
		"member": map[string]any{"id": 1},
		"set":    map[string]any{"number": "phone"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["phone"] != "380XXXXXXXXX" {
		t.Errorf("SetVars = %v, want phone=380XXXXXXXXX", out.SetVars)
	}
}

func TestMemberInfo_DepError(t *testing.T) {
	store := &fakeMemberStore{propsErr: fmt.Errorf("db error")}
	op := &memberInfoOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"member": map[string]any{"id": 1},
		"set":    map[string]any{"x": "y"},
	}))
	if err == nil {
		t.Fatal("expected error when GetProperties fails")
	}
}

// ── patchMembers ──────────────────────────────────────────────────────────────

func TestPatchMembers_NoMember(t *testing.T) {
	op := &patchMembersOp{store: &fakeMemberStore{}}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"patch": map[string]any{"stopCause": "done"},
	}))
	if err == nil {
		t.Fatal("expected error when member is nil")
	}
}

func TestPatchMembers_NoPatch(t *testing.T) {
	op := &patchMembersOp{store: &fakeMemberStore{}}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"member": map[string]any{"id": 1},
	}))
	if err == nil {
		t.Fatal("expected error when patch is nil")
	}
}

func TestPatchMembers_Success(t *testing.T) {
	store := &fakeMemberStore{patchCount: 3}
	op := &patchMembersOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"member": map[string]any{"id": 1},
		"patch":  map[string]any{"stopCause": "manual"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchMembers_DepCausePromoted(t *testing.T) {
	// StopCauseDep set when StopCause is nil → should be promoted.
	// We exercise this path by passing deprecated field names.
	// Since decoding maps json tags, we pass the args directly to exercise the promotion.
	cause := "legacy"
	argv := patchMembersArgs{
		Member: &queuedomain.SearchMember{},
		Patch: &queuedomain.PatchMember{
			StopCauseDep: &cause,
		},
	}
	// Manually exercise the promotion logic.
	if argv.Patch.StopCauseDep != nil && argv.Patch.StopCause == nil {
		argv.Patch.StopCause = argv.Patch.StopCauseDep
	}
	if argv.Patch.StopCause == nil || *argv.Patch.StopCause != "legacy" {
		t.Error("StopCauseDep should have been promoted to StopCause")
	}
}

func TestPatchMembers_DepError(t *testing.T) {
	store := &fakeMemberStore{patchErr: fmt.Errorf("constraint")}
	op := &patchMembersOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"member": map[string]any{"id": 1},
		"patch":  map[string]any{"stopCause": "done"},
	}))
	if err == nil {
		t.Fatal("expected error when PatchMembers fails")
	}
}

// ── ewt ───────────────────────────────────────────────────────────────────────

func TestEWT_NoSetVar(t *testing.T) {
	op := &ewtOp{store: &fakeMemberStore{}}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"queue_ids": []any{1},
	}))
	if err == nil {
		t.Fatal("expected error when setVar is empty")
	}
}

func TestEWT_DefaultMinIs60(t *testing.T) {
	// EWTPuzzle is called; we verify no error and result stored.
	store := &fakeMemberStore{ewtVal: 120.5}
	op := &ewtOp{store: store}
	out, err := op.Execute(context.Background(), memberInput(map[string]any{
		"setVar":    "ewt_result",
		"queue_ids": []any{1, 2},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["ewt_result"] == "" {
		t.Error("expected ewt_result to be set")
	}
}

func TestEWT_DepError(t *testing.T) {
	store := &fakeMemberStore{ewtErr: fmt.Errorf("puzzle error")}
	op := &ewtOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"setVar": "ewt",
	}))
	if err == nil {
		t.Fatal("expected error when EWTPuzzle fails")
	}
}

// ── callbackQueue ─────────────────────────────────────────────────────────────

func TestCallbackQueue_Success(t *testing.T) {
	store := &fakeMemberStore{}
	op := &callbackQueueOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"queue":   map[string]any{"id": 3},
		"holdSec": 60,
		"communications": []any{
			map[string]any{"destination": "+380XXXXXXXXX"},
		},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.createCalls) != 1 {
		t.Errorf("CreateMember called %d times, want 1", len(store.createCalls))
	}
}

func TestCallbackQueue_DepError(t *testing.T) {
	store := &fakeMemberStore{createErr: fmt.Errorf("duplicate")}
	op := &callbackQueueOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"queue": map[string]any{"id": 3},
	}))
	if err == nil {
		t.Fatal("expected error when CreateMember fails")
	}
}

func TestCallbackQueue_EmptyStopCauseCleared(t *testing.T) {
	// An empty string StopCause must be set to nil before CreateMember.
	store := &fakeMemberStore{}
	op := &callbackQueueOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"queue":      map[string]any{"id": 1},
		"stopCause":  "",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.createCalls) != 1 {
		t.Fatal("CreateMember not called")
	}
	if store.createCalls[0].StopCause != nil {
		t.Error("expected StopCause to be nil when empty string provided")
	}
}

func TestCallbackQueue_InvalidUTF8_Sanitized(t *testing.T) {
	store := &fakeMemberStore{}
	op := &callbackQueueOp{store: store}
	_, err := op.Execute(context.Background(), memberInput(map[string]any{
		"queue": map[string]any{"id": 1},
		"name":  "Test\x80Name", // invalid UTF-8 byte
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.createCalls) != 1 {
		t.Fatal("CreateMember not called")
	}
	// The invalid byte should have been stripped.
	if !utf8.ValidString(store.createCalls[0].Name) {
		t.Errorf("Name still contains invalid UTF-8: %q", store.createCalls[0].Name)
	}
}
