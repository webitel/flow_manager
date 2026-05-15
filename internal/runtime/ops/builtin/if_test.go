package builtin_test

import (
	"context"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ifInput builds an OpInput for the if op.
// children[0] = then-branch node, children[1] = else-branch node (optional).
func ifInput(expr string, vars map[string]string, children ...*tree.Node) ops.OpInput {
	return ops.OpInput{
		Node: &tree.Node{
			Args:     map[string]any{"expression": expr},
			Children: children,
		},
		Variables: vars,
	}
}

var thenNode = &tree.Node{ID: "then"}
var elseNode = &tree.Node{ID: "else"}

func TestIf_True_ReturnsThenBranch(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("1 == 1", nil, thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != thenNode {
		t.Errorf("Branch = %v, want thenNode", out.Branch)
	}
}

func TestIf_False_ReturnsElseBranch(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("1 == 2", nil, thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != elseNode {
		t.Errorf("Branch = %v, want elseNode", out.Branch)
	}
}

func TestIf_True_NoChildren_EmptyOutput(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("true", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil {
		t.Errorf("expected nil Branch when no children")
	}
}

func TestIf_False_OnlyThen_EmptyOutput(t *testing.T) {
	// false with only one child (then) → no else, returns empty.
	out, err := builtin.If().Execute(context.Background(),
		ifInput("false", nil, thenNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != nil {
		t.Error("expected nil Branch when false and no else child")
	}
}

func TestIf_VariableExpansion(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("${lang} == 'uk'",
			map[string]string{"lang": "uk"},
			thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != thenNode {
		t.Errorf("expected thenNode when ${lang}='uk' and expression is uk==uk")
	}
}

func TestIf_VariableExpansion_False(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("${lang} == 'uk'",
			map[string]string{"lang": "en"},
			thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != elseNode {
		t.Errorf("expected elseNode when ${lang}='en'")
	}
}

func TestIf_NumericComparison(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("${count} > 5",
			map[string]string{"count": "10"},
			thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != thenNode {
		t.Errorf("expected thenNode when count=10 > 5")
	}
}

func TestIf_WdayHelper(t *testing.T) {
	// &wday() returns current day; we just check it doesn't error.
	_, err := builtin.If().Execute(context.Background(),
		ifInput("&wday(1-7)", nil, thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIf_InvalidExpression_ReturnsError(t *testing.T) {
	_, err := builtin.If().Execute(context.Background(),
		ifInput("}}{{", nil))
	if err == nil {
		t.Fatal("expected error for invalid JS expression")
	}
}

func TestIf_AndOperator(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("${a} == '1' && ${b} == '2'",
			map[string]string{"a": "1", "b": "2"},
			thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != thenNode {
		t.Error("expected thenNode when both conditions true")
	}
}

func TestIf_OrOperator(t *testing.T) {
	out, err := builtin.If().Execute(context.Background(),
		ifInput("${a} == 'x' || ${b} == '2'",
			map[string]string{"a": "no", "b": "2"},
			thenNode, elseNode))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != thenNode {
		t.Error("expected thenNode when one OR condition is true")
	}
}
