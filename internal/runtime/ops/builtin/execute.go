package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type executeOp struct{}

// Execute returns the native execute op which calls a named function defined
// in the same schema. Sync mode (default) enters the function's sub-tree
// inline; async mode runs it in a goroutine via RunBranch.
func Execute() ops.Op { return executeOp{} }

func (executeOp) Kind() ops.OpKind { return ops.OpKindSync }

func (executeOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	name, _ := in.Node.Args["name"].(string)
	name = ops.ExpandStr(name, in.Variables, in.GlobalVar)
	if name == "" {
		return ops.OpOutput{}, fmt.Errorf("execute: name is required")
	}

	fn, ok := in.Functions[name]
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("execute: function %q not found", name)
	}

	async, _ := in.Node.Args["async"].(bool)
	if async && in.RunBranch != nil {
		varSnap := make(map[string]string, len(in.Variables))
		for k, v := range in.Variables {
			varSnap[k] = v
		}
		in.RunBranch(ctx, fn, varSnap)
		return ops.OpOutput{}, nil
	}

	return ops.OpOutput{Branch: fn}, nil
}
