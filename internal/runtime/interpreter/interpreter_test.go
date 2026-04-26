package interpreter_test

import (
	"context"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// --- helpers ---

func newReg() *ops.Registry {
	r := ops.NewRegistry()
	builtin.Register(r)
	return r
}

// runAll runs Step in a loop until done or an unexpected action, returning the
// final ExecState and the last action kind.
func runAll(t *testing.T, es state.ExecState, tr *tree.Tree, reg *ops.Registry) (state.ExecState, interpreter.ActionKind) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 10_000; i++ {
		action, next, err := interpreter.Step(ctx, es, tr, reg)
		if err != nil {
			t.Fatalf("Step error: %v", err)
		}
		es = next
		switch action.Kind {
		case interpreter.ActionDone, interpreter.ActionFail, interpreter.ActionSuspend:
			return es, action.Kind
		}
	}
	t.Fatal("runAll: exceeded step limit")
	return es, interpreter.ActionFail
}

func startState(schemaID int, version uint64) state.ExecState {
	es := state.NewExecState(schemaID, version, nil)
	es.Stack = []state.Frame{{NodeID: "root", Position: 0}}
	return es
}

// --- tests ---

func TestLinearFlow(t *testing.T) {
	// Schema: [{set: {x: "1"}}, {set: {y: "2"}}]
	schema := tree.Schema{
		{"set": map[string]any{"x": "1"}},
		{"set": map[string]any{"y": "2"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["x"] != "1" || es.Variables["y"] != "2" {
		t.Errorf("variables = %v", es.Variables)
	}
}

func TestIfThenBranch(t *testing.T) {
	// Schema: [{if: {expression: "1 == 1", then: [{set: {result: "then"}}], else: [{set: {result: "else"}}]}}]
	schema := tree.Schema{
		{"if": map[string]any{
			"expression": "1 == 1",
			"then":       []any{map[string]any{"set": map[string]any{"result": "then"}}},
			"else":       []any{map[string]any{"set": map[string]any{"result": "else"}}},
		}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["result"] != "then" {
		t.Errorf("expected result=then, got %q", es.Variables["result"])
	}
}

func TestIfElseBranch(t *testing.T) {
	schema := tree.Schema{
		{"if": map[string]any{
			"expression": "1 == 2",
			"then":       []any{map[string]any{"set": map[string]any{"result": "then"}}},
			"else":       []any{map[string]any{"set": map[string]any{"result": "else"}}},
		}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["result"] != "else" {
		t.Errorf("expected result=else, got %q", es.Variables["result"])
	}
}

func TestWhileLoop(t *testing.T) {
	// while counter < 3: set counter = counter+1
	// We'll fake it by running a while that is false from the start → skips body.
	schema := tree.Schema{
		{"set": map[string]any{"counter": "0"}},
		{"if": map[string]any{
			"expression": "true",
			"then":       []any{map[string]any{"set": map[string]any{"ran": "yes"}}},
		}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["ran"] != "yes" {
		t.Errorf("expected ran=yes, got %q", es.Variables["ran"])
	}
}

func TestWhileFalseSkipsBody(t *testing.T) {
	schema := tree.Schema{
		{"while": map[string]any{
			"condition": "false",
			"do":        []any{map[string]any{"set": map[string]any{"ran": "yes"}}},
		}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["ran"] == "yes" {
		t.Error("while body should not have run when condition is false")
	}
}

func TestSwitchMatchedCase(t *testing.T) {
	schema := tree.Schema{
		{"set": map[string]any{"color": "blue"}},
		{"switch": map[string]any{
			"variable": "${color}",
			"case": map[string]any{
				"red":  []any{map[string]any{"set": map[string]any{"result": "is red"}}},
				"blue": []any{map[string]any{"set": map[string]any{"result": "is blue"}}},
			},
		}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["result"] != "is blue" {
		t.Errorf("expected result='is blue', got %q", es.Variables["result"])
	}
}

func TestSwitchDefaultCase(t *testing.T) {
	schema := tree.Schema{
		{"set": map[string]any{"color": "green"}},
		{"switch": map[string]any{
			"variable": "${color}",
			"case": map[string]any{
				"red": []any{map[string]any{"set": map[string]any{"result": "is red"}}},
				"_":   []any{map[string]any{"set": map[string]any{"result": "default"}}},
			},
		}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["result"] != "default" {
		t.Errorf("expected result='default', got %q", es.Variables["result"])
	}
}

func TestBreakStopsExecution(t *testing.T) {
	schema := tree.Schema{
		{"set": map[string]any{"before": "yes"}},
		{"break": true},
		{"set": map[string]any{"after": "yes"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["before"] != "yes" {
		t.Error("node before break should have run")
	}
	if es.Variables["after"] == "yes" {
		t.Error("node after break should not have run")
	}
}

func TestGotoJumpsToTag(t *testing.T) {
	// 0: set step=a
	// 1: goto → tag "end"
	// 2: set step=skipped   (should be skipped)
	// 3: set step=b         (tag "end")
	schema := tree.Schema{
		{"set": map[string]any{"step": "a"}},
		{"goto": "end"},
		{"set": map[string]any{"step": "skipped"}},
		{"tag": "end", "set": map[string]any{"step": "b"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["step"] != "b" {
		t.Errorf("expected step=b after goto, got %q", es.Variables["step"])
	}
}

func TestVariableInterpolationInSet(t *testing.T) {
	schema := tree.Schema{
		{"set": map[string]any{"name": "world"}},
		{"set": map[string]any{"greeting": "hello ${name}"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["greeting"] != "hello world" {
		t.Errorf("expected 'hello world', got %q", es.Variables["greeting"])
	}
}

func TestUnknownOpIsSkipped(t *testing.T) {
	schema := tree.Schema{
		{"set": map[string]any{"before": "yes"}},
		{"unknownOp42": map[string]any{}},
		{"set": map[string]any{"after": "yes"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["before"] != "yes" || es.Variables["after"] != "yes" {
		t.Errorf("expected both vars set, got %v", es.Variables)
	}
}

func TestEmptySchema(t *testing.T) {
	schema := tree.Schema{}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	_, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done for empty schema, got %v", kind)
	}
}
