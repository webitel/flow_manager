package legacy

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// LegacyOp adapts a flow.Application (legacy handler) to the ops.Op interface.
// The caller must store a model.Connection in ctx via WithConnection before calling Execute.
//
// Variable tracking: snapshot conn.Variables() before/after the handler so type
// assertions inside the handler (e.g. scope.Connection.(model.Call)) are preserved —
// the original connection is passed as-is, not wrapped.
type LegacyOp struct {
	name   string
	app    *flow.Application
	router model.Router
}

func (l *LegacyOp) Kind() ops.OpKind { return ops.OpKindSuspendable }

func (l *LegacyOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn := ConnectionFromContext(ctx)
	if conn == nil {
		return ops.OpOutput{}, fmt.Errorf("legacy op %q: no connection in context", l.name)
	}

	before := snapshotVars(conn.Variables())
	scope := flow.New(l.router, flow.Config{Conn: conn})

	var args interface{} = in.Node.RawArgs
	if l.app.ArgsParser != nil {
		args = l.app.ArgsParser(conn, args)
	}

	result := <-l.app.Handler(ctx, scope, args)
	if result.Err != nil {
		return ops.OpOutput{}, result.Err
	}

	return ops.OpOutput{SetVars: diffVars(before, conn.Variables())}, nil
}

func snapshotVars(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func diffVars(before, after map[string]string) map[string]string {
	var out map[string]string
	for k, v := range after {
		if before[k] != v {
			if out == nil {
				out = make(map[string]string)
			}
			out[k] = v
		}
	}
	return out
}
