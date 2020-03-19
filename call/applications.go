package call

import (
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type callHandler func(call model.Call, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) model.ApplicationHandlers {
	var apps = make(model.ApplicationHandlers)

	apps["ringReady"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.ringReady),
	}

	apps["preAnswer"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.preAnswer),
	}

	apps["answer"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.answer),
	}

	apps["hangup"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.hangup),
	}

	apps["setAll"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.setAll),
	}

	apps["setNoLocal"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.setNoLocal),
	}

	apps["bridge"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.bridge),
	}

	apps["echo"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.echo),
	}

	apps["export"] = &model.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.export),
	}

	return apps
}

func callHandlerMiddleware(h callHandler) model.ApplicationHandler {
	return func(c model.Connection, args interface{}) (model.Response, *model.AppError) {
		if c.Type() != model.ConnectionTypeCall {
			return nil, model.NewAppError("Call", "call.middleware.valid.type", nil, "bad type", http.StatusBadRequest)
		}
		return h(c.(model.Call), args)
	}
}
