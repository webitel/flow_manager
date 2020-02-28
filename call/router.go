package call

import (
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
)

type Router struct {
	fm   *app.FlowManager
	apps model.ApplicationHandlers
}

func Init(fm *app.FlowManager) {
	var router = &Router{
		fm: fm,
	}

	fm.CallRouter = &Router{
		fm: fm,
		apps: model.UnionApplicationMap(
			fm.FlowRouter.Handlers(),
			ApplicationsHandlers(router),
		),
	}
}

func (r *Router) Handlers() model.ApplicationHandlers {
	return r.apps
}

func (r *Router) Request(conn model.Connection, req model.ApplicationRequest) (model.Response, *model.AppError) {
	if h, ok := r.apps[req.Id()]; ok {
		if h.ArgsParser != nil {
			return h.Handler(conn, h.ArgsParser(conn, req.Args()))
		} else {
			return h.Handler(conn, req.Args())
		}

	}
	return nil, model.NewAppError("Call.Request", "call.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	call := conn.(model.Call)
	var routing *model.Routing
	var err *model.AppError

	wlog.Debug(fmt.Sprintf("call %s [domain_id=%d direction=%s user_id=%d] - %s", call.Id(), call.DomainId(),
		call.Direction(), call.UserId(), call.Destination()))

	switch call.Direction() {
	case model.CallDirectionInbound:
	case model.CallDirectionOutbound:
	}

	routing, err = r.fm.GetRoutingFromGateway(1, 3)
	if err != nil {
		wlog.Warn(err.Error())
		if err.StatusCode == http.StatusNotFound {
			_, err = call.HangupNoRoute()
		} else {
			_, err = call.HangupAppErr()
		}
		return err
	}

	i := flow.New(flow.Config{
		Name:    routing.Schema.Name,
		Handler: r,
		Apps:    routing.Schema.Schema,
		Conn:    conn,
	})
	flow.Route(i, r)

	return nil
}
