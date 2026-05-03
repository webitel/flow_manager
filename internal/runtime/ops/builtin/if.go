package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type ifOp struct{}

// If evaluates a JS boolean expression and branches to then or else.
// Supports the same expression syntax as the legacy flow/if.go:
// ${var} → sys.getVariable("var"), &hour() → sys.hour(), etc.
func If() ops.Op { return ifOp{} }

func (ifOp) Kind() ops.OpKind { return ops.OpKindSync }

func (ifOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	expr, _ := in.Node.Args["expression"].(string)
	expr = parseExpression(expr)

	vm := buildVM(in.Variables, in.GlobalVar, in.Timezone)
	val, err := vm.RunString("_r = (" + expr + ")")
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("if: eval %q: %w", expr, err)
	}

	result := val.ToBoolean()

	node := in.Node
	if result && len(node.Children) > 0 {
		return ops.OpOutput{Branch: node.Children[0]}, nil
	}
	if !result && len(node.Children) > 1 {
		return ops.OpOutput{Branch: node.Children[1]}, nil
	}
	return ops.OpOutput{}, nil
}
