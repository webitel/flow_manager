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
	apps["sendImage"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendImage),
	}
	apps["sendFile"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendFile),
	}
	apps["sendAction"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendAction),
	}
	apps["sendTts"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.sendTTS),
	}
	apps["stt"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.STT),
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
	apps["unSet"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.UnSet),
	}
	apps["chatAi"] = &flow.Application{
		Handler: chatHandlerMiddleware(r.chatAI),
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
