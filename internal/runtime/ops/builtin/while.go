package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type whileOp struct{}

// While evaluates a JS boolean condition; if true it enters the do-body with
// Repeat=true so the interpreter re-runs the while node after the body.
// Supports the same expression syntax as the legacy flow/if.go.
func While() ops.Op { return whileOp{} }

func (whileOp) Kind() ops.OpKind { return ops.OpKindSync }

func (whileOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	cond, _ := in.Node.Args["condition"].(string)
	cond = parseExpression(cond)

	vm := buildVM(in.Variables)
	val, err := vm.Run("_r = (" + cond + ")")
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("while: eval %q: %w", cond, err)
	}

	result, _ := val.ToBoolean()
	if !result || len(in.Node.Children) == 0 {
		return ops.OpOutput{}, nil
	}

	return ops.OpOutput{Branch: in.Node.Children[0], Repeat: true}, nil
}
