package grpc_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type grpcHandler func(ctx context.Context, scope *flow.Flow, call model.GRPCConnection, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["cancel"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.cancel),
	}
	apps["confirm"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.confirm),
	}

	return apps
}

func grpcHandlerMiddleware(h grpcHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args interface{}) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(model.GRPCConnection), args)
		})
	}
}
