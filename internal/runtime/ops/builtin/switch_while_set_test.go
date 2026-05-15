package builtin_test

import (
	"context"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── switch ────────────────────────────────────────────────────────────────────

var caseA = &tree.Node{ID: "case-a"}
var caseB = &tree.Node{ID: "case-b"}
var caseDefault = &tree.Node{ID: "case-default"}

func switchInput(varExpr string, casesIndex map[string]int, vars map[string]string, children ...*tree.Node) ops.OpInput {
	return ops.OpInput{
		Node: &tree.Node{
			Args: map[string]any{
				"variable":      varExpr,
				"_cases_index": casesIndex,
			},
			Children: children,
		},
		Variables: vars,
	}
}

func TestSwitch_ExactMatch(t *testing.T) {
	out, err := builtin.Switch().Execute(context.Background(),
		switchInput("${choice}",
			map[string]int{"1": 0, "2": 1, "_": 2},
			map[string]string{"choice": "1"},
			caseA, caseB, caseDefault))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != caseA {
		t.Errorf("Branch = %v, want caseA", out.Branch)
	}
}

func TestSwitch_SecondCase(t *testing.T) {
	out, err := builtin.Switch().Execute(context.Background(),
		switchInput("${choice}",
			map[string]int{"1": 0, "2": 1},
			map[string]string{"choice": "2"},
			caseA, caseB))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != caseB {
		t.Errorf("Branch = %v, want caseB", out.Branch)
	}
}

func TestSwitch_DefaultFallback(t *testing.T) {
	out, err := builtin.Switch().Execute(context.Background(),
		switchInput("${choice}",
			map[string]int{"1": 0, "_": 1},
			map[string]string{"choice": "99"},
			caseA, caseDefault))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != caseDefault {
		t.Errorf("Branch = %v, want caseDefault", out.Branch)
	}
}

func TestSwitch_NoMatch_NoDefault_EmptyOutput(t *testing.T) {
	out, err := builtin.Switch().Execute(context.Background(),
		switchInput("${choice}",
			map[string]int{"1": 0},
			map[string]string{"choice": "99"},
			caseA))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil {
		t.Error("expected nil Branch when no match and no default")
	}
}

func TestSwitch_NoCasesIndex_EmptyOutput(t *testing.T) {
	// No _cases_index in Args → empty output, no error.
	out, err := builtin.Switch().Execute(context.Background(),
		ops.OpInput{
			Node:      &tree.Node{Args: map[string]any{"variable": "${x}"}},
			Variables: map[string]string{"x": "1"},
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil {
		t.Error("expected nil Branch when no _cases_index")
	}
}

func TestSwitch_LiteralVariable(t *testing.T) {
	// variable without ${} — literal value lookup.
	out, err := builtin.Switch().Execute(context.Background(),
		switchInput("uk",
			map[string]int{"uk": 0, "en": 1},
			nil,
			caseA, caseB))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != caseA {
		t.Errorf("Branch = %v, want caseA", out.Branch)
	}
}

// ── while ─────────────────────────────────────────────────────────────────────

var bodyNode = &tree.Node{ID: "while-body"}

func whileInput(cond string, vars map[string]string, children ...*tree.Node) ops.OpInput {
	return ops.OpInput{
		Node: &tree.Node{
			Args:     map[string]any{"condition": cond},
			Children: children,
		},
		Variables: vars,
	}
}

func TestWhile_True_ReturnsBranchAndRepeat(t *testing.T) {
	out, err := builtin.While().Execute(context.Background(),
		whileInput("true", nil, bodyNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != bodyNode {
		t.Errorf("Branch = %v, want bodyNode", out.Branch)
	}
	if !out.Repeat {
		t.Error("expected Repeat=true when condition is true")
	}
}

func TestWhile_False_EmptyOutput(t *testing.T) {
	out, err := builtin.While().Execute(context.Background(),
		whileInput("false", nil, bodyNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil || out.Repeat {
		t.Error("expected empty output when condition is false")
	}
}

func TestWhile_True_NoChildren_EmptyOutput(t *testing.T) {
	out, err := builtin.While().Execute(context.Background(),
		whileInput("true", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil || out.Repeat {
		t.Error("expected empty output when true but no body children")
	}
}

func TestWhile_VariableCondition(t *testing.T) {
	out, err := builtin.While().Execute(context.Background(),
		whileInput("${tries} < 3",
			map[string]string{"tries": "1"},
			bodyNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != bodyNode {
		t.Error("expected bodyNode when tries=1 < 3")
	}
}

func TestWhile_CounterBoundary(t *testing.T) {
	out, err := builtin.While().Execute(context.Background(),
		whileInput("${tries} < 3",
			map[string]string{"tries": "3"},
			bodyNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil {
		t.Error("expected nil Branch when tries=3 is NOT < 3")
	}
}

func TestWhile_InvalidCondition_ReturnsError(t *testing.T) {
	_, err := builtin.While().Execute(context.Background(),
		whileInput("}}{{", nil, bodyNode))
	if err == nil {
		t.Fatal("expected error for invalid condition")
	}
}

// ── set ───────────────────────────────────────────────────────────────────────

func setInput(args map[string]any, vars map[string]string) ops.OpInput {
	return ops.OpInput{
		Node:      &tree.Node{Args: args},
		Variables: vars,
	}
}

func TestSet_FlatArgs(t *testing.T) {
	out, err := builtin.Set().Execute(context.Background(),
		setInput(map[string]any{"lang": "uk", "max_tries": "3"}, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["lang"] != "uk" || out.SetVars["max_tries"] != "3" {
		t.Errorf("SetVars = %v", out.SetVars)
	}
}

func TestSet_VariableExpansionInValue(t *testing.T) {
	out, err := builtin.Set().Execute(context.Background(),
		setInput(
			map[string]any{"greeting": "Hello ${name}!"},
			map[string]string{"name": "Alice"},
		))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["greeting"] != "Hello Alice!" {
		t.Errorf("greeting = %q, want Hello Alice!", out.SetVars["greeting"])
	}
}

func TestSet_EmptyArgs_NilSetVars(t *testing.T) {
	out, err := builtin.Set().Execute(context.Background(),
		setInput(map[string]any{}, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars != nil {
		t.Errorf("expected nil SetVars for empty args, got %v", out.SetVars)
	}
}

func TestSet_MultipleValues(t *testing.T) {
	out, err := builtin.Set().Execute(context.Background(),
		setInput(map[string]any{"a": "1", "b": "2", "c": "3"}, nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.SetVars) != 3 {
		t.Errorf("expected 3 vars, got %d: %v", len(out.SetVars), out.SetVars)
	}
}
