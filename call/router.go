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

func (r *Router) ToRequired(call model.Call, in *model.CallEndpoint) (*model.CallEndpoint, *model.AppError) {
	if in == nil {
		wlog.Error(fmt.Sprintf("call %s not found to", call.Id()))
		_, err := call.HangupAppErr()
		return nil, err
	} else {
		return in, nil
	}
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	call := conn.(model.Call)
	var routing *model.Routing
	var err *model.AppError

	wlog.Info(fmt.Sprintf("call %s [%d %s] from: [%s] to: [%s] destination %s", call.Id(), call.DomainId(), call.Direction(),
		call.From().String(), call.To().String(), call.Destination()))

	from := call.From()
	if from == nil {
		wlog.Info("not allowed call: from is empty")
		_, err = call.HangupAppErr()
		return err
	}

	switch call.Direction() {
	case model.CallDirectionInbound:
		var to *model.CallEndpoint
		to, err = r.ToRequired(call, call.To())
		if err != nil {
			_, err = call.HangupAppErr()
			return err
		}

		switch from.Type {
		case model.CallEndpointTypeDestination:
			if id := to.IntId(); id == nil {
				_, err = call.HangupNoRoute()
				return err
			} else {
				wlog.Debug(fmt.Sprintf("call %s search schema from gateway \"%s\" [%d]", call.Id(), to.Name, *id))
				routing, err = r.fm.GetRoutingFromDestToGateway(call.DomainId(), *id)
			}

		}
	case model.CallDirectionOutbound:
		switch from.Type {
		case model.CallEndpointTypeUser:
			routing, err = r.fm.SearchOutboundToDestinationRouting(call.DomainId(), call.Destination())
		}

	default:
		err = model.NewAppError("Call.Handle", "call.router.valid.direction", nil, fmt.Sprintf("no handler direction %s", call.Direction()), http.StatusInternalServerError)
	}

	if err != nil {
		wlog.Error(err.Error())
		_, err = call.HangupAppErr()
		return err
	}

	if routing == nil {
		wlog.Error(fmt.Sprintf("call %s not found routing", call.Id()))
		_, err = call.HangupNoRoute()
		return err
	}

	call.SetDomainName(routing.DomainName) //fixme

	i := flow.New(flow.Config{
		Name:    routing.Schema.Name,
		Handler: r,
		Apps:    routing.Schema.Schema,
		Conn:    conn,
	})
	flow.Route(i, r)

	return nil
}
