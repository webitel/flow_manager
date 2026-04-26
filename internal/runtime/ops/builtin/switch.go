package builtin

import (
	"context"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type switchOp struct{}

// Switch reads a variable and branches to the matching case container.
func Switch() ops.Op { return switchOp{} }

func (switchOp) Kind() ops.OpKind { return ops.OpKindSync }

func (switchOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	varExpr, _ := in.Node.Args["variable"].(string)
	value := expand(varExpr, in.Variables)

	index, ok := in.Node.Args["_cases_index"].(map[string]int)
	if !ok {
		return ops.OpOutput{}, nil
	}

	// Try exact match first, then fall back to "_" default.
	childIdx, matched := index[value]
	if !matched {
		childIdx, matched = index["_"]
	}
	if !matched || childIdx >= len(in.Node.Children) {
		return ops.OpOutput{}, nil
	}

	return ops.OpOutput{Branch: in.Node.Children[childIdx]}, nil
}
