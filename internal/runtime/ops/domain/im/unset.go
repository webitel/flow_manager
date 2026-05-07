package im

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// RegisterUnSet registers the unSet op.
func RegisterUnSet(reg *ops.Registry) {
	reg.Register("unSet", &unSetOp{})
}

type unSetOp struct{}

func (o *unSetOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *unSetOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("unSet: no IMDialog in context")
	}

	keys := rawStringSlice(in)
	if len(keys) == 0 {
		return ops.OpOutput{}, fmt.Errorf("unSet: required parameter missing")
	}

	if _, appErr := dialog.UnSet(ctx, keys); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("unSet: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}
