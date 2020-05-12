package call

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
)

type Router struct {
	fm   *app.FlowManager
	apps flow.ApplicationHandlers
}

func Init(fm *app.FlowManager, fr flow.Router) {
	var router = &Router{
		fm: fm,
	}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	fm.CallRouter = router
}

func (r *Router) Handlers() flow.ApplicationHandlers {
	return r.apps
}

func (r *Router) Request(scope *flow.Flow, req model.ApplicationRequest) (model.Response, *model.AppError) {
	if h, ok := r.apps[req.Id()]; ok {
		if h.ArgsParser != nil {
			return h.Handler(scope, scope.Connection, h.ArgsParser(scope.Connection, req.Args()))
		} else {
			return h.Handler(scope, scope.Connection, req.Args())
		}

	}
	return nil, model.NewAppError("Call.Request", "call.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
}

func (r *Router) ToRequired(call model.Call, in *model.CallEndpoint) *model.CallEndpoint {
	if in == nil {
		wlog.Error(fmt.Sprintf("call %s not found to", call.Id()))
		if _, err := call.HangupAppErr(); err != nil {
			wlog.Error(err.Error())
		}
		return nil
	} else {
		return in
	}
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	go r.handle(conn)

	return nil
}

func (r *Router) handle(conn model.Connection) {
	call := &callParser{
		Call: conn.(model.Call),
	}

	var routing *model.Routing
	var err *model.AppError

	wlog.Info(fmt.Sprintf("call %s [%d %s] from: [%s] to: [%s] destination %s", call.Id(), call.DomainId(), call.Direction(),
		call.From().String(), call.To().String(), call.Destination()))

	from := call.From()
	if from == nil {
		wlog.Error("not allowed call: from is empty")
		if _, err = call.HangupAppErr(); err != nil {
			wlog.Error(err.Error())
		}

		return
	}

	switch call.Direction() {
	case model.CallDirectionInbound:
		var to *model.CallEndpoint
		to = r.ToRequired(call, call.To())
		if to == nil {

			return
		}

		switch from.Type {
		case model.CallEndpointTypeDestination:
			if id := to.IntId(); id == nil {
				if _, err = call.HangupNoRoute(); err != nil {
					wlog.Error(err.Error())
				}

				return
			} else {
				wlog.Debug(fmt.Sprintf("call %s search schema from gateway \"%s\" [%d]", call.Id(), to.Name, *id))
				routing, err = r.fm.GetRoutingFromDestToGateway(call.DomainId(), *id)
			}

		}
	case model.CallDirectionOutbound:
		switch from.Type {
		case model.CallEndpointTypeUser:
			if routing, err = r.fm.SearchOutboundToDestinationRouting(call.DomainId(), call.Destination()); err == nil {
				call.outboundVars, err = getOutboundReg(routing.SourceData, call.Destination())
			}
		}

	default:
		err = model.NewAppError("Call.Handle", "call.router.valid.direction", nil, fmt.Sprintf("no handler direction %s", call.Direction()), http.StatusInternalServerError)
	}

	if err != nil {
		wlog.Error(err.Error())
		if _, err = call.HangupAppErr(); err != nil {
			wlog.Error(err.Error())
		}

		return
	}

	if routing == nil {
		wlog.Error(fmt.Sprintf("call %s not found routing", call.Id()))
		if _, err = call.HangupNoRoute(); err != nil {
			wlog.Error(err.Error())
		}

		return
	}

	call.timezoneName = routing.TimezoneName
	call.SetDomainName(routing.DomainName) //fixme
	i := flow.New(flow.Config{
		Name:     routing.Schema.Name,
		Schema:   routing.Schema.Schema,
		Handler:  r,
		Conn:     call,
		Timezone: routing.TimezoneName,
	})

	ctx, _ := context.WithCancel(context.TODO()) // CALL CONTEXT
	flow.Route(ctx, i, r)
}
