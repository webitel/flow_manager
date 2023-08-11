package flow

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
)

type Response struct {
	Status string
}

var ResponseOK = Response{"SUCCESS"}
var ResponseErr = Response{"FAIL"}

type router struct {
	fm   *app.FlowManager
	apps ApplicationHandlers
}

func (r Response) String() string {
	return r.Status
}

func NewRouter(fm *app.FlowManager) Router {
	var router = &router{
		fm: fm,
	}

	router.apps = ApplicationsHandlers(router)
	return router
}

func (r *router) Handlers() ApplicationHandlers {
	return r.apps
}

func (r *router) Handle(conn model.Connection) *model.AppError {
	return model.NewAppError("Flow", "flow.router.not_implement", nil, "not implement", http.StatusInternalServerError)
}

type flowHandler func(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError)

func (r *router) doExecute(delMe flowHandler) ApplicationHandler {
	return func(ctx context.Context, scope *Flow, args interface{}) model.ResultChannel {
		return Do(func(result *model.Result) {
			result.Res, result.Err = delMe(ctx, scope, scope.Connection, args)
		})
	}
}
