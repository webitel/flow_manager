package chat_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type callHandler func(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["sendText"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendText),
	}
	apps["recvMessage"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.recvMessage),
	}
	apps["bridge"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.bridge),
	}

	return apps
}

func chatHandlerMiddleware(h callHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args interface{}) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(Conversation), args)
		})
	}
}
