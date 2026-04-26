package builtin

import (
	"context"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type breakOp struct{}

func Break() ops.Op { return breakOp{} }

func (breakOp) Kind() ops.OpKind { return ops.OpKindSync }

func (breakOp) Execute(_ context.Context, _ ops.OpInput) (ops.OpOutput, error) {
	return ops.OpOutput{Break: true}, nil
}
