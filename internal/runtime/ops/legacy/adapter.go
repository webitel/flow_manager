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

	tc := newVarTrackingConn(conn)
	scope := flow.New(l.router, flow.Config{Conn: tc})

	var args interface{} = in.Node.Args
	if l.app.ArgsParser != nil {
		args = l.app.ArgsParser(tc, args)
	}

	result := <-l.app.Handler(ctx, scope, args)
	if result.Err != nil {
		return ops.OpOutput{}, result.Err
	}

	return ops.OpOutput{SetVars: tc.delta()}, nil
}
