package builtin

import (
	"context"
	"fmt"

	"github.com/robertkrimen/otto"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type ifOp struct{}

// If evaluates a JS boolean expression and branches to then or else.
func If() ops.Op { return ifOp{} }

func (ifOp) Kind() ops.OpKind { return ops.OpKindSync }

func (ifOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	expr, _ := in.Node.Args["expression"].(string)
	expr = expand(expr, in.Variables)

	vm := otto.New()
	val, err := vm.Run("_r = (" + expr + ")")
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("if: eval %q: %w", expr, err)
	}

	result, _ := val.ToBoolean()

	// Children[0] = then container, Children[1] = else container.
	node := in.Node
	if result && len(node.Children) > 0 {
		return ops.OpOutput{Branch: node.Children[0]}, nil
	}
	if !result && len(node.Children) > 1 {
		return ops.OpOutput{Branch: node.Children[1]}, nil
	}
	return ops.OpOutput{}, nil
}
