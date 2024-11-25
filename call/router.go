package call

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
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

func (r *Router) GlobalVariable(domainId int64, name string) string {
	return r.fm.SchemaVariable(context.TODO(), domainId, name)
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

func (e *Router) notFoundRoute(call model.Call) {
	wlog.Warn(fmt.Sprintf("call %s not found route schema", call.Id()))
	if _, err := call.HangupNoRoute(call.Context()); err != nil {
		wlog.Error(err.Error())
	}
}

func (r *Router) handle(conn model.Connection) {
	call := &callParser{
		Call: conn.(model.Call),
	}

	var routing *model.Routing
	var err *model.AppError

	queueId := call.IVRQueueId()
	transferSchemaId := call.TransferSchemaId()
	isTransfer := call.IsTransfer()

	// TODO WTEL-4370
	ccXfer := strings.HasSuffix(call.GetVariable("variable_transfer_history"),
		fmt.Sprintf(":bl_xfer:%s/default/XML", call.Destination())) && call.GetVariable("variable_cc_app_id") != ""

	if transferSchemaId != nil && isTransfer {
		routing, err = r.fm.SearchTransferredRouting(call.DomainId(), *transferSchemaId)
	} else if isTransfer && queueId == nil && (ccXfer || !call.IsOriginateRequest()) {
		wlog.Info(fmt.Sprintf("call %s [%d %s] is transfer from: [%s] to destination %s", call.Id(), call.DomainId(), call.Direction(),
			call.From().String(), call.Destination()))
		if routing, err = r.fm.SearchOutboundToDestinationRouting(call.DomainId(), call.Destination()); err == nil {
			call.outboundVars, err = getOutboundReg(routing.SourceData, call.Destination())
		}

		r.fm.SetBlindTransferNumber(call.DomainId(), call.Id(), call.Destination())
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
		if err == model.ErrNotFoundRoute {
			r.notFoundRoute(call)
		} else {
			wlog.Error(err.Error())
			if _, err = call.HangupAppErr(call.Context()); err != nil {
				wlog.Error(err.Error())
			}
		}

		return
	}

	if routing == nil {
		r.notFoundRoute(call)

		return
	}

	call.timezoneName = routing.TimezoneName
	call.SetDomainName(routing.DomainName) //fixme
	i := flow.New(r, flow.Config{
		SchemaId: routing.SchemaId,
		Name:     routing.Schema.Name,
		Schema:   routing.Schema.Schema,
		Handler:  r,
		Conn:     call,
		Timezone: routing.TimezoneName,
	})
	if err = call.SetSchemaId(i.SchemaId()); err != nil {
		wlog.Error(err.Error())
	}

	flow.Route(conn.Context(), i, r)
	<-conn.Context().Done()

	if d, err := i.TriggerScope(flow.TriggerDisconnected); err == nil {
		call.ClearExportVariables()

		//TODO config
		ctxDisc, cn := context.WithDeadline(context.Background(), time.Now().Add(60*time.Second))
		flow.Route(ctxDisc, d, r)
		cn()
		r.fm.StoreCallVariables(call.Id(), call.DumpExportVariables())
	}

	r.fm.StoreLog(i.SchemaId(), conn.Id(), i.Logs())

}
