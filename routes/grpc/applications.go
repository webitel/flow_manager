package grpc

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type grpcHandler func(ctx context.Context, scope *flow.Flow, call model.GRPCConnection, args any) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	apps := make(flow.ApplicationHandlers)

	apps["cancel"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.cancel),
	}
	apps["confirm"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.confirm),
	}
	apps["abandoned"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.abandoned),
	}
	apps["success"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.success),
	}
	apps["retry"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.retry),
	}
	apps["export"] = &flow.Application{
		Handler: grpcHandlerMiddleware(r.export),
	}

	return apps
}

func grpcHandlerMiddleware(h grpcHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args any) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(model.GRPCConnection), args)
		})
	}
}
