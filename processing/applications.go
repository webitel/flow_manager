package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type processingHandler func(ctx context.Context, scope *flow.Flow, c Connection, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["form"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.form),
	}
	apps["component"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.component),
	}

	return apps
}

func processingHandlerMiddleware(h processingHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args interface{}) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(Connection), args)
		})
	}
}
