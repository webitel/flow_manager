package legacy

import (
	"context"
	"errors"
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
		// No live connection available (e.g. timer wakeup after service restart).
		// Skip gracefully so the flow can continue to non-IM ops.
		return ops.OpOutput{}, nil
	}

	// Sync interpreter variables into the connection so that legacy ops can
	// interpolate them via conn.ParseText / conn.Get. Without this, variables
	// set by builtin ops (set, etc.) would not be visible to sendText and
	// similar handlers that call scope.Decode / conn.ParseText.
	if len(in.Variables) > 0 {
		vars := make(model.Variables, len(in.Variables))
		for k, v := range in.Variables {
			vars[k] = v
		}
		conn.Set(ctx, vars) // error not actionable; IM Set is in-memory and never fails
	}

	before := snapshotVars(conn.Variables())
	scope := flow.New(l.router, flow.Config{Conn: conn})

	var args any = in.Node.RawArgs
	if l.app.ArgsParser != nil {
		args = l.app.ArgsParser(conn, args)
	}

	result := <-l.app.Handler(ctx, scope, args)

	var appErr *model.AppError

	if errors.As(result.Err, &appErr) && appErr != nil { // todo
		return ops.OpOutput{}, fmt.Errorf("legacy op %q: %w", l.name, appErr)
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
