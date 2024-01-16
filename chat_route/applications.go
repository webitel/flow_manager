package chat_route

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type callHandler func(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["sendMessage"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendMessage),
	}
	apps["sendText"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendText),
	}
	apps["sendFile"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendFile),
	}
	apps["sendTTS"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendTTS),
	}
	apps["STT"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.STT),
	}
	apps["sendImage"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendImage),
	}
	apps["recvMessage"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.recvMessage),
	}
	apps["bridge"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.bridge),
	}
	apps["joinQueue"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.joinQueue),
	}
	apps["cancelQueue"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.cancelQueue),
	}
	apps["export"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.export),
	}
	apps["menu"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.Menu),
	}
	apps["sendAction"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendAction),
	}
	apps["unSet"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.UnSet),
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
