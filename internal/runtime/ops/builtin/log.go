package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type logOp struct{}

// Log logs a message with optional variable interpolation. It never mutates
// state and always returns an empty OpOutput.
func Log() ops.Op { return logOp{} }

func (logOp) Kind() ops.OpKind { return ops.OpKindSync }

func (logOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	msg, _ := in.Node.Args["log"].(string)
	if msg == "" {
		// Fallback: any string value under any key.
		for _, v := range in.Node.Args {
			if s, ok := v.(string); ok && s != "" {
				msg = s
				break
			}
		}
	}
	wlog.Info(fmt.Sprintf("[runtime] %s", expand(msg, in.Variables)))
	return ops.OpOutput{}, nil
}
