package connctx

import (
	"context"

	"github.com/webitel/flow_manager/internal/domain/flow"
)

type contextKey struct{}

func WithConnection(ctx context.Context, conn flow.Connection) context.Context {
	return context.WithValue(ctx, contextKey{}, conn)
}

func ConnectionFromContext(ctx context.Context) flow.Connection {
	conn, _ := ctx.Value(contextKey{}).(flow.Connection)
	return conn
}
