package flow

import (
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
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

	apps["if"] = &model.Application{
		AllowNoConnect: false,
		Handler:        r.conditionHandler,
	}

	apps["execute"] = &model.Application{
		AllowNoConnect: false,
		Handler:        r.execute,
	}

	return apps
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	return model.NewAppError("Flow", "flow.router.not_implement", nil, "not implement", http.StatusInternalServerError)
}

func (r *Router) Request(conn model.Connection, req model.ApplicationRequest) (model.Response, *model.AppError) {
	return nil, nil
}

func Route(i *Flow, handler app.Handler) {
	var req *ApplicationRequest
	var err *model.AppError
	var res model.Response

	for {
		req = i.NextRequest()
		if req == nil {
			break
		}

		if res, err = handler.Request(i.conn, req); err != nil {
			wlog.Error(fmt.Sprintf("%v [%v] - %s", req.Id(), req.Args(), err.Error()))
		} else {
			wlog.Debug(fmt.Sprintf("%v [%v] - %s", req.Id(), req.Args(), res.String()))
		}
	}
}
