package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type processingHandler func(ctx context.Context, scope *flow.Flow, c Connection, args any) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	apps := make(flow.ApplicationHandlers)

	apps["generateForm"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.generateForm),
	}
	apps["formComponent"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.formComponent),
	}
	apps["formFile"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.formFile),
	}
	apps["formTable"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.formTable),
	}
	apps["attemptResult"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.attemptResult),
	}
	apps["resumeAttempt"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.resumeAttempt),
	}
	apps["formSelectCaseStatus"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.formSelectCaseStatus),
	}
	apps["export"] = &flow.Application{
		Handler: processingHandlerMiddleware(r.export),
	}

	return apps
}

func processingHandlerMiddleware(h processingHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args any) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(Connection), args)
		})
	}
}
