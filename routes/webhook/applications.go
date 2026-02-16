package webhook

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/http"
)

type hookHandler func(ctx context.Context, scope *flow.Flow, c *http.Connection, args any) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	apps := make(flow.ApplicationHandlers)

	apps["httpResponse"] = &flow.Application{
		Handler: hookHandlerMiddleware(r.httpResponse),
	}

	return apps
}

func hookHandlerMiddleware(h hookHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args any) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(*http.Connection), args)
		})
	}
}
