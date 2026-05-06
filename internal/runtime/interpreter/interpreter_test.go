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
		action, next, err := interpreter.Step(ctx, nil, es, tr, reg, 0, "", nil, nil, nil)
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

func TestIfWithStringVarExpression(t *testing.T) {
	// expression uses ${var} which the old naive expand broke on string comparison.
	// parseExpression converts it to sys.getVariable("name") == "Alice".
	schema := tree.Schema{
		{"set": map[string]any{"name": "Alice"}},
		{"if": map[string]any{
			"expression": `${name} == "Alice"`,
			"then":       []any{map[string]any{"set": map[string]any{"result": "match"}}},
			"else":       []any{map[string]any{"set": map[string]any{"result": "no-match"}}},
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
	if es.Variables["result"] != "match" {
		t.Errorf("expected result=match, got %q", es.Variables["result"])
	}
}

func TestIfWithNumericVarExpression(t *testing.T) {
	// JS coerces the string "10" to number when compared with > 5.
	schema := tree.Schema{
		{"set": map[string]any{"count": "10"}},
		{"if": map[string]any{
			"expression": `${count} > 5`,
			"then":       []any{map[string]any{"set": map[string]any{"result": "big"}}},
			"else":       []any{map[string]any{"set": map[string]any{"result": "small"}}},
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
	if es.Variables["result"] != "big" {
		t.Errorf("expected result=big, got %q", es.Variables["result"])
	}
}

// TestGotoIntoNestedBranch verifies that goto to a tag inside a composite-op
// branch (e.g. if-then) properly continues execution after the branch is
// exhausted, rather than terminating the flow early.
//
// Schema (pseudocode):
//
//	0: set iter=0
//	1: if false → then:[{tag:t, set count=inc}], else:[]
//	2: set after_if=yes
//	3: if iter==0 → then:[set iter=1, goto t], else:[]
//	4: set final=yes
//
// Without the fix, goto "t" resets the stack to [{1.then, pos=0}], exhausting
// the then-container makes the stack empty → ActionDone before node 4 runs.
// With the fix the stack is [{root, pos=2}, {1.then, pos=0}], so after the
// then-container is done execution resumes at node 2 and reaches node 4.
func TestGotoIntoNestedBranch(t *testing.T) {
	schema := tree.Schema{
		{"set": map[string]any{"iter": "0"}},
		{"if": map[string]any{
			"expression": "false",
			"then": []any{
				map[string]any{"tag": "t", "set": map[string]any{"count": "incremented"}},
			},
		}},
		{"set": map[string]any{"after_if": "yes"}},
		{"if": map[string]any{
			"expression": `${iter} == 0`,
			"then": []any{
				map[string]any{"set": map[string]any{"iter": "1"}},
				map[string]any{"goto": "t"},
			},
		}},
		{"set": map[string]any{"final": "yes"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}

	es, kind := runAll(t, startState(1, tr.Version), tr, newReg())
	if kind != interpreter.ActionDone {
		t.Fatalf("expected Done, got %v", kind)
	}
	if es.Variables["count"] != "incremented" {
		t.Errorf("count not incremented: %q", es.Variables["count"])
	}
	if es.Variables["final"] != "yes" {
		t.Errorf("final not set — execution stopped before node 4 (old goto bug): final=%q", es.Variables["final"])
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

// --- inline trigger branch tests ---

// fakeWaitOp is a minimal suspendable op used in trigger tests.
// On first call it suspends with ReenterOnResume. On resume:
//   - payload["msg"] matches a commands-* trigger → inline branch + ReenterOnResume
//   - otherwise → sets the variable named by Node.Args["set"] and continues
type fakeWaitOp struct{}

func (fakeWaitOp) Kind() ops.OpKind { return ops.OpKindSuspendable }
func (fakeWaitOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	setVar, _ := in.Node.Args["set"].(string)

	if in.ResumePayload != nil {
		msg := in.ResumePayload["msg"]
		if trig, ok := in.Triggers["commands-"+msg]; ok {
			return ops.OpOutput{
				Branch:          trig,
				ReenterOnResume: true,
			}, nil
		}
		out := ops.OpOutput{}
		if setVar != "" {
			out.SetVars = map[string]string{setVar: msg}
		}
		return out, nil
	}
	return ops.OpOutput{SuspendKey: "test:1", ReenterOnResume: true}, nil
}

// runUntilTerminal runs Steps (passing firstPayload on the first Step only) until
// ActionSuspend, ActionDone, or ActionFail.
func runUntilTerminal(t *testing.T, es state.ExecState, tr *tree.Tree, reg *ops.Registry, firstPayload map[string]string) (state.ExecState, interpreter.Action) {
	t.Helper()
	ctx := context.Background()
	payload := firstPayload
	for i := 0; i < 10_000; i++ {
		action, next, err := interpreter.Step(ctx, nil, es, tr, reg, 0, "", nil, payload, nil)
		if err != nil {
			t.Fatalf("Step error: %v", err)
		}
		payload = nil
		es = next
		switch action.Kind {
		case interpreter.ActionDone, interpreter.ActionFail, interpreter.ActionSuspend:
			return es, action
		}
	}
	t.Fatal("runUntilTerminal: exceeded step limit")
	return es, interpreter.Action{}
}

// TestInlineTriggerCommand verifies that a trigger command received by
// fakeWaitOp runs its sub-tree inline, then re-suspends so the next message
// goes to the main waitMsg.
func TestInlineTriggerCommand(t *testing.T) {
	schema := tree.Schema{
		{"triggers": map[string]any{
			"commands": map[string]any{
				"/help": []any{map[string]any{"set": map[string]any{"helpRan": "yes"}}},
			},
		}},
		{"waitMsg": map[string]any{"set": "answer"}},
		{"set": map[string]any{"after": "yes"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}
	reg := newReg()
	reg.Register("waitMsg", fakeWaitOp{})

	es := startState(1, tr.Version)

	// Initial run → waitMsg suspends.
	es, act := runUntilTerminal(t, es, tr, reg, nil)
	if act.Kind != interpreter.ActionSuspend {
		t.Fatalf("step1: expected Suspend, got %v", act.Kind)
	}

	// Resume with trigger command → trigger runs inline, then re-suspends.
	es, act = runUntilTerminal(t, es, tr, reg, map[string]string{"msg": "/help"})
	if act.Kind != interpreter.ActionSuspend {
		t.Fatalf("step2: expected Suspend after trigger, got %v", act.Kind)
	}
	if es.Variables["helpRan"] != "yes" {
		t.Errorf("step2: trigger branch did not run: helpRan=%q", es.Variables["helpRan"])
	}

	// Resume with real message → main waitMsg receives it, flow completes.
	es, act = runUntilTerminal(t, es, tr, reg, map[string]string{"msg": "hello"})
	if act.Kind != interpreter.ActionDone {
		t.Fatalf("step3: expected Done, got %v", act.Kind)
	}
	if es.Variables["answer"] != "hello" {
		t.Errorf("step3: answer=%q, expected 'hello'", es.Variables["answer"])
	}
	if es.Variables["after"] != "yes" {
		t.Errorf("step3: after-op did not run")
	}
}

// TestInlineTriggerWithNestedWait verifies the complex case where the trigger
// sub-tree itself contains a suspendable op (fakeWaitOp). The trigger's wait
// suspends the entire flow; the next message goes to the trigger's waitMsg,
// not to the main waitMsg. When the trigger finishes the main waitMsg
// re-executes and suspends for the final message.
func TestInlineTriggerWithNestedWait(t *testing.T) {
	schema := tree.Schema{
		{"triggers": map[string]any{
			"commands": map[string]any{
				"/help": []any{
					map[string]any{"set": map[string]any{"helpStart": "yes"}},
					map[string]any{"waitMsg": map[string]any{"set": "trigAnswer"}},
					map[string]any{"set": map[string]any{"helpEnd": "yes"}},
				},
			},
		}},
		{"waitMsg": map[string]any{"set": "mainAnswer"}},
		{"set": map[string]any{"after": "yes"}},
	}
	tr, err := tree.Parse(1, schema)
	if err != nil {
		t.Fatal(err)
	}
	reg := newReg()
	reg.Register("waitMsg", fakeWaitOp{})

	es := startState(1, tr.Version)

	// 1. Initial run → main waitMsg suspends.
	es, act := runUntilTerminal(t, es, tr, reg, nil)
	if act.Kind != interpreter.ActionSuspend {
		t.Fatalf("step1: expected Suspend, got %v", act.Kind)
	}

	// 2. "/help" → trigger starts: set helpStart, then trigger's waitMsg suspends.
	es, act = runUntilTerminal(t, es, tr, reg, map[string]string{"msg": "/help"})
	if act.Kind != interpreter.ActionSuspend {
		t.Fatalf("step2: expected Suspend (trigger's waitMsg), got %v", act.Kind)
	}
	if es.Variables["helpStart"] != "yes" {
		t.Errorf("step2: helpStart not set")
	}
	if es.Variables["helpEnd"] != "" {
		t.Errorf("step2: helpEnd should not be set yet")
	}

	// 3. Reply for trigger's waitMsg → trigger completes, main waitMsg re-suspends.
	es, act = runUntilTerminal(t, es, tr, reg, map[string]string{"msg": "help reply"})
	if act.Kind != interpreter.ActionSuspend {
		t.Fatalf("step3: expected Suspend (main waitMsg), got %v", act.Kind)
	}
	if es.Variables["helpEnd"] != "yes" {
		t.Errorf("step3: helpEnd not set — trigger did not complete")
	}
	if es.Variables["trigAnswer"] != "help reply" {
		t.Errorf("step3: trigAnswer=%q, expected 'help reply'", es.Variables["trigAnswer"])
	}
	if es.Variables["mainAnswer"] != "" {
		t.Errorf("step3: mainAnswer should still be empty")
	}

	// 4. Final message → main waitMsg receives it, flow completes.
	es, act = runUntilTerminal(t, es, tr, reg, map[string]string{"msg": "final"})
	if act.Kind != interpreter.ActionDone {
		t.Fatalf("step4: expected Done, got %v", act.Kind)
	}
	if es.Variables["mainAnswer"] != "final" {
		t.Errorf("step4: mainAnswer=%q, expected 'final'", es.Variables["mainAnswer"])
	}
	if es.Variables["after"] != "yes" {
		t.Errorf("step4: after-op did not run")
	}
}
