package flow

import (
	"context"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type Response struct {
	Status string
}

var ResponseOK = Response{"SUCCESS"}

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

func ApplicationsHandlers(r *router) ApplicationHandlers {
	var apps = make(ApplicationHandlers)

	apps["log"] = &Application{
		AllowNoConnect: true,
		Handler:        r.Log,
	}
	apps["if"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.conditionHandler),
	}
	apps["switch"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.switchHandler),
	}
	apps["execute"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.execute),
	}
	apps["set"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.set),
	}
	apps["break"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.breakHandler),
	}
	apps["httpRequest"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.httpRequest),
	}
	apps["string"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.stringApp),
	}
	apps["math"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Math),
	}
	apps["calendar"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Calendar),
	}
	//apps["list"] = &Application{
	//	AllowNoConnect: true,
	//	Handler:        r.doExecute(r.List),
	//}

	return apps
}

func (r *router) Handle(conn model.Connection) *model.AppError {
	return model.NewAppError("Flow", "flow.router.not_implement", nil, "not implement", http.StatusInternalServerError)
}

func (r *router) doExecute(delMe func(c model.Connection, args interface{}) (model.Response, *model.AppError)) ApplicationHandler {
	return func(ctx context.Context, scope *Flow, args interface{}) model.ResultChannel {
		return Do(func(result *model.Result) {
			result.Res, result.Err = delMe(scope.Connection, args)
		})
	}
}
