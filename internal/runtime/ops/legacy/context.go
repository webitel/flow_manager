package legacy

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type contextKey struct{}

// WithConnection stores conn in ctx for retrieval by LegacyOp.Execute.
func WithConnection(ctx context.Context, conn model.Connection) context.Context {
	return context.WithValue(ctx, contextKey{}, conn)
}

// ConnectionFromContext retrieves the connection stored by WithConnection.
func ConnectionFromContext(ctx context.Context) model.Connection {
	conn, _ := ctx.Value(contextKey{}).(model.Connection)
	return conn
}
