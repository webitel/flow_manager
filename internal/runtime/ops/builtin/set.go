package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type setOp struct{}

// Set assigns variables from the node's args, interpolating ${var} in values.
// The args map may hold either a flat key→value object or a single key→value
// under the "set" op key (depending on the schema author's style).
func Set() ops.Op { return setOp{} }

func (setOp) Kind() ops.OpKind { return ops.OpKindSync }

func (setOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	out := ops.OpOutput{SetVars: make(map[string]string)}

	for k, v := range in.Node.Args {
		if k == "set" {
			// Nested object under the op key.
			if m, ok := v.(map[string]any); ok {
				for mk, mv := range m {
					out.SetVars[mk] = expand(fmt.Sprintf("%v", mv), in.Variables)
				}
			}
			continue
		}
		out.SetVars[k] = expand(fmt.Sprintf("%v", v), in.Variables)
	}

	if len(out.SetVars) == 0 {
		out.SetVars = nil
	}
	return out, nil
}
