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
		call.Log().Error("not found TO")
		if _, err := call.HangupAppErr(call.Context()); err != nil {
			call.Log().Err(err)
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
	call.Log().Debug("not found route schema")
	if _, err := call.HangupNoRoute(call.Context()); err != nil {
		call.Log().Err(err)
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
		call.Log().Info("transfer from: " + call.From().String() + " to destination " + call.Destination())
		if routing, err = r.fm.SearchOutboundToDestinationRouting(call.DomainId(), call.Destination()); err == nil {
			call.outboundVars, err = getOutboundReg(routing.SourceData, call.Destination())
		}

		r.fm.SetBlindTransferNumber(call.DomainId(), call.Id(), call.Destination())
	} else if queueId != nil {
		call.Log().Info("ivr from: " + call.From().String() + " to destination " + call.Destination())

		routing, err = r.fm.SearchOutboundFromQueueRouting(call.DomainId(), *queueId)

	} else {
		from := call.From()
		if from == nil {
			call.Log().Error("not allowed call: from is empty")
			if _, err = call.HangupAppErr(call.Context()); err != nil {
				call.Log().Err(err)
			}

			return
		}
		call.Log().Info("call from: " + call.From().String() + " to: " + call.To().String() + ", destination: " + call.Destination())

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
						call.Log().Err(err)
					}

					return
				} else {
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

	autoLink, _ := r.fm.GetSystemSettings(conn.Context(), conn.DomainId(), model.SysAutoLinkCallToContact)
	if autoLink.BoolValue {
		r.linkContact(call)
	}

	if err != nil {
		if err == model.ErrNotFoundRoute {
			r.notFoundRoute(call)
		} else {
			call.Log().Err(err)
			if _, err = call.HangupAppErr(call.Context()); err != nil {
				call.Log().Err(err)
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
		call.Log().Err(err)
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
