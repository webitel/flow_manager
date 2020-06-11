package call

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
	"time"
)

type Router struct {
	fm               *app.FlowManager
	apps             flow.ApplicationHandlers
	disconnectedApps flow.ApplicationHandlers
}

func Init(fm *app.FlowManager, fr flow.Router) {
	var router = &Router{
		fm: fm,
	}

	router.disconnectedApps = fr.Handlers()

	router.apps = flow.UnionApplicationMap(
		router.disconnectedApps,
		ApplicationsHandlers(router),
	)

	fm.CallRouter = router
}

func (r *Router) Handlers() flow.ApplicationHandlers {
	return r.apps
}

func (r *Router) ToRequired(call model.Call, in *model.CallEndpoint) *model.CallEndpoint {
	if in == nil {
		wlog.Error(fmt.Sprintf("call %s not found to", call.Id()))
		if _, err := call.HangupAppErr(call.Context()); err != nil {
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

	queueId := call.IVRQueueId()
	transferSchemaId := call.TransferSchemaId()

	if transferSchemaId != nil && call.IsTransfer() {
		routing, err = r.fm.SearchTransferredRouting(call.DomainId(), *transferSchemaId)
	} else if call.IsTransfer() && queueId == nil {
		wlog.Info(fmt.Sprintf("call %s [%d %s] is transfer from: [%s] to destination %s", call.Id(), call.DomainId(), call.Direction(),
			call.From().String(), call.Destination()))
		if routing, err = r.fm.SearchOutboundToDestinationRouting(call.DomainId(), call.Destination()); err == nil {
			call.outboundVars, err = getOutboundReg(routing.SourceData, call.Destination())
		}
	} else if queueId != nil {
		wlog.Info(fmt.Sprintf("call %s [%d %s] is ivr from: [%s] to destination %s", call.Id(), call.DomainId(), call.Direction(),
			call.From().String(), call.Destination()))

		routing, err = r.fm.SearchOutboundFromQueueRouting(call.DomainId(), *queueId)

	} else {
		wlog.Info(fmt.Sprintf("call %s [%d %s] from: [%s] to: [%s] destination %s", call.Id(), call.DomainId(), call.Direction(),
			call.From().String(), call.To().String(), call.Destination()))

		from := call.From()
		if from == nil {
			wlog.Error("not allowed call: from is empty")
			if _, err = call.HangupAppErr(call.Context()); err != nil {
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
					if _, err = call.HangupNoRoute(call.Context()); err != nil {
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
	}

	if err != nil {
		wlog.Error(err.Error())
		if _, err = call.HangupAppErr(call.Context()); err != nil {
			wlog.Error(err.Error())
		}

		return
	}

	if routing == nil {
		wlog.Error(fmt.Sprintf("call %s not found routing", call.Id()))
		if _, err = call.HangupNoRoute(call.Context()); err != nil {
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

	flow.Route(conn.Context(), i, r)
	<-conn.Context().Done()

	if d, err := i.TriggerScope(flow.TriggerDisconnected); err == nil {
		//TODO config
		ctxDisc, _ := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		flow.Route(ctxDisc, d, r)
		<-ctxDisc.Done()
	}

}
