package im

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// RegisterExport registers the export op.
func RegisterExport(reg *ops.Registry) {
	reg.Register("export", &exportOp{})
}

type exportOp struct{}

func (o *exportOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *exportOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	dialog, ok := dialogFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("export: no IMDialog in context")
	}

	vars := rawStringSlice(in)
	if _, appErr := dialog.Export(ctx, vars); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("export: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}
