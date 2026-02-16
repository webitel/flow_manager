package im

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type callHandler func(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	apps := make(flow.ApplicationHandlers)

	apps["sendMessage"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendMessage),
	}
	apps["sendText"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendText),
	}
	apps["recvMessage"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.recvMessage),
	}

	return apps
}

func chatHandlerMiddleware(h callHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args any) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(Dialog), args)
		})
	}
}
