package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type gotoOp struct{}

func Goto() ops.Op { return gotoOp{} }

func (gotoOp) Kind() ops.OpKind { return ops.OpKindSync }

func (gotoOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	tag, _ := in.Node.Args["goto"].(string)
	if tag == "" {
		return ops.OpOutput{}, fmt.Errorf("goto: missing tag")
	}
	return ops.OpOutput{Goto: tag}, nil
}
