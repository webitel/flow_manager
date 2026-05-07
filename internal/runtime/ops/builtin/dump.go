package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type dumpOp struct{}

// Dump returns the native dump op which prints all current variables to stdout.
func Dump() ops.Op { return dumpOp{} }

func (dumpOp) Kind() ops.OpKind { return ops.OpKindSync }

func (dumpOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	for k, v := range in.Variables {
		fmt.Printf("%s = %s\n", k, v)
	}
	return ops.OpOutput{}, nil
}
