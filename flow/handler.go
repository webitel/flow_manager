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
		AllowNoConnect: true,
		Handler:        r.conditionHandler,
	}

	apps["switch"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.switchHandler,
	}

	apps["execute"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.execute,
	}

	apps["set"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.set,
	}

	apps["break"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.breakHandler,
	}

	apps["httpRequest"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.httpRequest,
	}

	apps["string"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.stringApp,
	}

	apps["math"] = &model.Application{
		AllowNoConnect: true,
		Handler:        r.Math,
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

	wlog.Debug(fmt.Sprintf("flow \"%s\" start conn %s", i.name, i.conn.Id()))
	defer wlog.Debug(fmt.Sprintf("flow \"%s\" stopped conn %s", i.name, i.conn.Id()))

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

		if i.IsCancel() || req.IsCancel() {
			wlog.Debug(fmt.Sprintf("flow [%s] break", i.Name()))
			break
		}
	}
}
