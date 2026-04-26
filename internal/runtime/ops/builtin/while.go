package builtin

import (
	"context"
	"fmt"

	"github.com/robertkrimen/otto"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type whileOp struct{}

// While evaluates a JS boolean condition; if true it enters the do-body with
// Repeat=true so the interpreter re-runs the while node after the body.
func While() ops.Op { return whileOp{} }

func (whileOp) Kind() ops.OpKind { return ops.OpKindSync }

func (whileOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	cond, _ := in.Node.Args["condition"].(string)
	cond = expand(cond, in.Variables)

	vm := otto.New()
	val, err := vm.Run("_r = (" + cond + ")")
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("while: eval %q: %w", cond, err)
	}

	result, _ := val.ToBoolean()
	if !result || len(in.Node.Children) == 0 {
		return ops.OpOutput{}, nil
	}

	// Children[0] = do container.
	return ops.OpOutput{Branch: in.Node.Children[0], Repeat: true}, nil
}
