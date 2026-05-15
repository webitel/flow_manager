package builtin_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── break ─────────────────────────────────────────────────────────────────────

func TestBreak_SetsBreakFlag(t *testing.T) {
	out, err := builtin.Break().Execute(context.Background(), ops.OpInput{Node: &tree.Node{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Break {
		t.Error("expected Break=true")
	}
}

// ── dump ──────────────────────────────────────────────────────────────────────

func TestDump_EmptyVars_NoError(t *testing.T) {
	_, err := builtin.Dump().Execute(context.Background(), ops.OpInput{Node: &tree.Node{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDump_WithVars_NoError(t *testing.T) {
	_, err := builtin.Dump().Execute(context.Background(), ops.OpInput{
		Node:      &tree.Node{},
		Variables: map[string]string{"a": "1", "b": "2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── goto ──────────────────────────────────────────────────────────────────────

func TestGoto_ReturnsGotoTag(t *testing.T) {
	out, err := builtin.Goto().Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"goto": "menu-tag"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Goto != "menu-tag" {
		t.Errorf("Goto = %q, want menu-tag", out.Goto)
	}
}

func TestGoto_EmptyTag_ReturnsError(t *testing.T) {
	_, err := builtin.Goto().Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"goto": ""}},
	})
	if err == nil {
		t.Fatal("expected error when tag is empty")
	}
}

func TestGoto_MissingArg_ReturnsError(t *testing.T) {
	_, err := builtin.Goto().Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{}},
	})
	if err == nil {
		t.Fatal("expected error when goto arg is missing")
	}
}

// ── execute ───────────────────────────────────────────────────────────────────

func TestExecute_NoName_ReturnsError(t *testing.T) {
	_, err := builtin.Execute().Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{}},
	})
	if err == nil {
		t.Fatal("expected error when name is empty")
	}
}

func TestExecute_FunctionNotFound_ReturnsError(t *testing.T) {
	_, err := builtin.Execute().Execute(context.Background(), ops.OpInput{
		Node:      &tree.Node{Args: map[string]any{"name": "missing-fn"}},
		Functions: map[string]*tree.Node{},
	})
	if err == nil {
		t.Fatal("expected error when function not found")
	}
}

func TestExecute_Sync_ReturnsBranch(t *testing.T) {
	fnNode := &tree.Node{ID: "my-fn"}
	out, err := builtin.Execute().Execute(context.Background(), ops.OpInput{
		Node:      &tree.Node{Args: map[string]any{"name": "my-fn", "async": false}},
		Functions: map[string]*tree.Node{"my-fn": fnNode},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != fnNode {
		t.Errorf("Branch = %v, want fnNode", out.Branch)
	}
}

func TestExecute_Async_CallsRunBranch(t *testing.T) {
	fnNode := &tree.Node{ID: "async-fn"}
	called := make(chan string, 1)
	out, err := builtin.Execute().Execute(context.Background(), ops.OpInput{
		Node:      &tree.Node{Args: map[string]any{"name": "async-fn", "async": true}},
		Functions: map[string]*tree.Node{"async-fn": fnNode},
		RunBranch: func(_ context.Context, n *tree.Node, _ map[string]string) {
			called <- n.ID
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil {
		t.Error("expected nil Branch in async mode")
	}
	select {
	case id := <-called:
		if id != "async-fn" {
			t.Errorf("RunBranch called with node %q, want async-fn", id)
		}
	case <-time.After(time.Second):
		t.Error("RunBranch not called within 1s")
	}
}

// ── global ────────────────────────────────────────────────────────────────────

type fakeGlobalDeps struct {
	err   error
	calls []struct{ name, value string }
}

func (f *fakeGlobalDeps) SetGlobalVar(_ context.Context, _ int64, name, value string, _ bool) error {
	f.calls = append(f.calls, struct{ name, value string }{name, value})
	return f.err
}

func TestGlobal_SetsVar(t *testing.T) {
	deps := &fakeGlobalDeps{}
	op := builtin.GlobalOp(deps)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"greeting": map[string]any{"value": "hello", "encrypt": false},
		}},
		DomainID: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.calls) != 1 || deps.calls[0].name != "greeting" || deps.calls[0].value != "hello" {
		t.Errorf("calls = %v", deps.calls)
	}
}

func TestGlobal_BadArgType_ReturnsError(t *testing.T) {
	deps := &fakeGlobalDeps{}
	op := builtin.GlobalOp(deps)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"bad_var": "string-not-object",
		}},
	})
	if err == nil {
		t.Fatal("expected error when var value is not an object")
	}
}

func TestGlobal_DepError_Propagated(t *testing.T) {
	deps := &fakeGlobalDeps{err: fmt.Errorf("permission denied")}
	op := builtin.GlobalOp(deps)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"x": map[string]any{"value": "1"},
		}},
	})
	if err == nil {
		t.Fatal("expected error when SetGlobalVar fails")
	}
}

// ── generateLink ──────────────────────────────────────────────────────────────

type fakeGenerateLinkDeps struct {
	link string
	err  error
}

func (f *fakeGenerateLinkDeps) GeneratePreSignedLink(_ context.Context, _, _ string, _, _ int64, _ map[string]string) (string, error) {
	return f.link, f.err
}

func TestGenerateLink_NoSet_ReturnsError(t *testing.T) {
	op := builtin.GenerateLinkOp(&fakeGenerateLinkDeps{link: "/file/1"})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{}},
	})
	if err == nil {
		t.Fatal("expected error when set is missing")
	}
}

func TestGenerateLink_Success(t *testing.T) {
	op := builtin.GenerateLinkOp(&fakeGenerateLinkDeps{link: "/file/1"})
	out, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"set":    "link",
			"source": "media",
		}},
		ConnID:   "42",
		DomainID: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["link"] == "" {
		t.Error("expected link to be set")
	}
}

func TestGenerateLink_DepError(t *testing.T) {
	op := builtin.GenerateLinkOp(&fakeGenerateLinkDeps{err: fmt.Errorf("storage error")})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"set": "link"}},
	})
	if err == nil {
		t.Fatal("expected error when GeneratePreSignedLink fails")
	}
}

// ── openLink ──────────────────────────────────────────────────────────────────

type fakeOpenLinkDeps struct {
	err   error
	calls int
}

func (f *fakeOpenLinkDeps) PushOpenLink(_ int64, _ string, _ int64, _, _ string) error {
	f.calls++
	return f.err
}

func TestOpenLink_Success(t *testing.T) {
	deps := &fakeOpenLinkDeps{}
	op := builtin.OpenLinkOp(deps)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"url":     "https://example.com",
			"message": "Click here",
			"userId":  float64(7),
		}},
		Variables: map[string]string{"wbt_sock_id": "sock-1"},
		DomainID:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.calls != 1 {
		t.Errorf("PushOpenLink called %d times, want 1", deps.calls)
	}
}

func TestOpenLink_UserIdFromVariable(t *testing.T) {
	deps := &fakeOpenLinkDeps{}
	op := builtin.OpenLinkOp(deps)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node:      &tree.Node{Args: map[string]any{"url": "https://x.com"}},
		Variables: map[string]string{"user_id": "42", "wbt_sock_id": "s1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenLink_DepError(t *testing.T) {
	deps := &fakeOpenLinkDeps{err: fmt.Errorf("ws error")}
	op := builtin.OpenLinkOp(deps)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"url": "https://x.com"}},
	})
	if err == nil {
		t.Fatal("expected error when PushOpenLink fails")
	}
}

// ── list ──────────────────────────────────────────────────────────────────────

type fakeListDeps struct {
	found    bool
	checkErr error
	addErr   error
}

func (f *fakeListDeps) CheckList(_ int64, _ string, _ *int, _ *string) (bool, error) {
	return f.found, f.checkErr
}
func (f *fakeListDeps) AddToList(_ context.Context, _ int64, _ *int, _ *string, _ string, _ *string, _ int64) error {
	return f.addErr
}

var listBranchNode = &tree.Node{ID: "list-match"}

func TestList_NoDestination_ReturnsError(t *testing.T) {
	op := builtin.ListOp(&fakeListDeps{})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{}},
	})
	if err == nil {
		t.Fatal("expected error when destination is empty")
	}
}

func TestList_Found_ReturnsBranch(t *testing.T) {
	op := builtin.ListOp(&fakeListDeps{found: true})
	out, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{
			Args:     map[string]any{"destination": "380501234567", "list": map[string]any{"id": float64(1)}},
			Children: []*tree.Node{listBranchNode},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != listBranchNode {
		t.Errorf("Branch = %v, want listBranchNode", out.Branch)
	}
}

func TestList_NotFound_EmptyOutput(t *testing.T) {
	op := builtin.ListOp(&fakeListDeps{found: false})
	out, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{
			Args:     map[string]any{"destination": "380501234567"},
			Children: []*tree.Node{listBranchNode},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil {
		t.Error("expected nil Branch when not found")
	}
}

func TestList_DepError(t *testing.T) {
	op := builtin.ListOp(&fakeListDeps{checkErr: fmt.Errorf("db error")})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"destination": "380"}},
	})
	if err == nil {
		t.Fatal("expected error when CheckList fails")
	}
}

func TestListAdd_NoDestination_ReturnsError(t *testing.T) {
	op := builtin.ListAddOp(&fakeListDeps{})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{}},
	})
	if err == nil {
		t.Fatal("expected error when destination is empty")
	}
}

func TestListAdd_Success(t *testing.T) {
	op := builtin.ListAddOp(&fakeListDeps{})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"destination": "380501234567",
			"list":        map[string]any{"id": float64(5)},
			"description": "blocked",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListAdd_DepError(t *testing.T) {
	op := builtin.ListAddOp(&fakeListDeps{addErr: fmt.Errorf("constraint")})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"destination": "380"}},
	})
	if err == nil {
		t.Fatal("expected error when AddToList fails")
	}
}

// ── timezone ──────────────────────────────────────────────────────────────────

func TestTimezone_ByName_SetsTimezone(t *testing.T) {
	op := builtin.TimezoneOp(nil)
	out, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"name": "Europe/Kyiv"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetTimezone != "Europe/Kyiv" {
		t.Errorf("SetTimezone = %q, want Europe/Kyiv", out.SetTimezone)
	}
}

func TestTimezone_ById_SetsTimezone(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	op := builtin.TimezoneOp(func(id int) *time.Location {
		if id == 99 {
			return loc
		}
		return nil
	})
	out, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"id": 99}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetTimezone != "America/New_York" {
		t.Errorf("SetTimezone = %q, want America/New_York", out.SetTimezone)
	}
}

func TestTimezone_InvalidName_ReturnsError(t *testing.T) {
	op := builtin.TimezoneOp(nil)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"name": "Not/A/Timezone"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid timezone name")
	}
}

func TestTimezone_NoIdOrName_ReturnsError(t *testing.T) {
	op := builtin.TimezoneOp(nil)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{}},
	})
	if err == nil {
		t.Fatal("expected error when neither id nor name provided")
	}
}

// ── sql ───────────────────────────────────────────────────────────────────────

type fakeSqlDeps struct {
	result map[string]interface{}
	err    error
}

func (f *fakeSqlDeps) SqlQuery(_ context.Context, _, _, _ string, _ []interface{}) (map[string]interface{}, error) {
	return f.result, f.err
}

func TestSql_NoQuery_ReturnsError(t *testing.T) {
	op := builtin.SqlOp(&fakeSqlDeps{})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"driver": "postgres", "dns": "postgres://localhost/db"}},
	})
	if err == nil {
		t.Fatal("expected error when query is empty")
	}
}

func TestSql_NoDriver_ReturnsError(t *testing.T) {
	op := builtin.SqlOp(&fakeSqlDeps{})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"query": "SELECT 1", "dns": "postgres://localhost/db"}},
	})
	if err == nil {
		t.Fatal("expected error when driver is empty")
	}
}

func TestSql_NoDns_ReturnsError(t *testing.T) {
	op := builtin.SqlOp(&fakeSqlDeps{})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"query": "SELECT 1", "driver": "postgres"}},
	})
	if err == nil {
		t.Fatal("expected error when dns is empty")
	}
}

func TestSql_Success_SetsVarsFromRow(t *testing.T) {
	op := builtin.SqlOp(&fakeSqlDeps{
		result: map[string]interface{}{"client_name": "Alice", "client_id": 42},
	})
	out, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"driver": "postgres",
			"dns":    "postgres://localhost/db",
			"query":  "SELECT * FROM clients WHERE phone = $1",
			"params": []interface{}{"380501234567"},
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["client_name"] != "Alice" {
		t.Errorf("client_name = %q, want Alice", out.SetVars["client_name"])
	}
}

func TestSql_DepError_Propagated(t *testing.T) {
	op := builtin.SqlOp(&fakeSqlDeps{err: fmt.Errorf("connection refused")})
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"driver": "postgres", "dns": "x", "query": "SELECT 1",
		}},
	})
	if err == nil {
		t.Fatal("expected error when SqlQuery fails")
	}
}
