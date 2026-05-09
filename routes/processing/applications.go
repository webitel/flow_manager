package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type processingHandler func(ctx context.Context, scope *flow.Flow, c Connection, args any) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	return flow.ApplicationHandlers{
		// formTable stays legacy: its output callbacks use flow.Route for sub-flows.
		"formTable": {Handler: processingHandlerMiddleware(r.formTable)},
	}
}

func processingHandlerMiddleware(h processingHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args any) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			var appErr *model.AppError
			result.Res, appErr = h(ctx, scope, scope.Connection.(Connection), args)
			if appErr != nil {
				result.Err = appErr
			}
		})
	}
}
