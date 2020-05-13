package email

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type emailHandler func(call model.EmailConnection, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["reply"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        emailHandlerMiddleware(r.reply),
	}

	return apps
}

func emailHandlerMiddleware(h emailHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args interface{}) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			if scope.Connection.Type() != model.ConnectionTypeEmail {
				result.Err = model.NewAppError("Email", "email.middleware.valid.type", nil, "bad type", http.StatusBadRequest)
			} else {
				result.Res, result.Err = h(scope.Connection.(model.EmailConnection), args)
			}
		})
	}
}
