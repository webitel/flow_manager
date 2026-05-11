package connctx

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type contextKey struct{}

func WithConnection(ctx context.Context, conn model.Connection) context.Context {
	return context.WithValue(ctx, contextKey{}, conn)
}

func ConnectionFromContext(ctx context.Context) model.Connection {
	conn, _ := ctx.Value(contextKey{}).(model.Connection)
	return conn
}
