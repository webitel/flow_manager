package grpc_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type ExportArgs []string

func (r *Router) export(ctx context.Context, scope *flow.Flow, conn model.GRPCConnection, args interface{}) (model.Response, *model.AppError) {
	var argv ExportArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return conn.Export(ctx, argv)
}
