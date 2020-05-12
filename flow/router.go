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

type Router struct {
	fm   *app.FlowManager
	apps model.ApplicationHandlers
}

func (r Response) String() string {
	return r.Status
}

func Init(fm *app.FlowManager) {
	var router = &Router{
		fm: fm,
	}

	router.apps = ApplicationsHandlers(router)

	fm.FlowRouter = router
}

func (r *Router) Handlers() model.ApplicationHandlers {
	return r.apps
}

func ApplicationsHandlers(r *Router) model.ApplicationHandlers {
	var apps = make(model.ApplicationHandlers)

	apps["log"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Log),
	}
	apps["if"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.conditionHandler),
	}
	apps["switch"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.switchHandler),
	}
	apps["execute"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.execute),
	}
	apps["set"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.set),
	}
	apps["break"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.breakHandler),
	}
	apps["httpRequest"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.httpRequest),
	}
	apps["string"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.stringApp),
	}
	apps["math"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Math),
	}
	apps["calendar"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Calendar),
	}
	apps["list"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.List),
	}

	return apps
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	return model.NewAppError("Flow", "flow.router.not_implement", nil, "not implement", http.StatusInternalServerError)
}

func (r *Router) doExecWithContext() {

}

func (r *Router) doExecute(delMe func(c model.Connection, args interface{}) (model.Response, *model.AppError)) model.ApplicationHandler {
	return func(ctx context.Context, c model.Connection, args interface{}) (model.Response, *model.AppError) {
		return delMe(c, args)
	}
}
