package email

import (
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type emailHandler func(call model.EmailConnection, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) model.ApplicationHandlers {
	var apps = make(model.ApplicationHandlers)

	apps["reply"] = &model.Application{
		AllowNoConnect: false,
		Handler:        emailHandlerMiddleware(r.reply),
	}

	return apps
}

func emailHandlerMiddleware(h emailHandler) model.ApplicationHandler {
	return func(c model.Connection, args interface{}) (model.Response, *model.AppError) {
		if c.Type() != model.ConnectionTypeEmail {
			return nil, model.NewAppError("Email", "email.middleware.valid.type", nil, "bad type", http.StatusBadRequest)
		}
		return h(c.(model.EmailConnection), args)
	}
}
