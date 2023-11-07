package webhook

import (
	"context"

	"github.com/webitel/flow_manager/providers/web_hook"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type hookHandler func(ctx context.Context, scope *flow.Flow, c *web_hook.Connection, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["httpResponse"] = &flow.Application{
		Handler: hookHandlerMiddleware(r.httpResponse),
	}

	return apps
}

func hookHandlerMiddleware(h hookHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args interface{}) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(*web_hook.Connection), args)
		})
	}
}
