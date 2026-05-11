package call

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/flow"
	domaincontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	ports "github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	callops "github.com/webitel/flow_manager/internal/runtime/ops/domain/call"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// callChannel is the channel discriminator stored in flow.runtime_state.
// Matches model.ConnectionTypeCall (iota = 0).
const callChannel = int16(model.ConnectionTypeCall)

type Router struct {
	fm         ports.RouterDeps
	contacts   domaincontacts.Client
	meeting    domainmeeting.Client
	apps       flow.ApplicationHandlers
	driver     *interpreter.Driver
	sessionMgr *sessionmgr.Manager
}

func Init(deps ports.RouterDeps, fr flow.Router, contacts domaincontacts.Client, meeting domainmeeting.Client) model.Router {
	router := &Router{
		fm:       deps,
		contacts: contacts,
		meeting:  meeting,
	}

	router.apps = fr.Handlers()

	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:   deps,
		Router: router,
		Apps:   router.apps,
		ExtraOps: func(reg *ops.Registry) {
			callops.Register(reg)
			callops.RegisterFM(reg, deps)
			callops.RegisterMedia(reg, deps)
			callops.RegisterComplex(reg, deps)
		},
		LoadTree: func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error) {
			s, appErr := deps.GetSchemaById(domainID, schemaID)
			if appErr != nil {
				return nil, appErr
			}
			rawSchema := make([]map[string]any, len(s.Schema))
			for i, app := range s.Schema {
				rawSchema[i] = map[string]any(app)
			}
			return tree.Parse(s.Id, rawSchema)
		},
	})
	router.driver = kit.Driver
	router.sessionMgr = sessionmgr.New(kit.Coord, deps.RuntimeStateRepo(), deps.Log())

	return router
}

func (r *Router) AddApplications(apps flow.ApplicationHandlers) flow.Handler {
	r2 := *r
	r2.apps = maps.Clone(r.apps)

	for k, v := range apps {
		r2.apps[k] = v
	}

	return &r2
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
	}
	return in
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
	transferQueueId := call.TransferQueueId()
	transferAgentId := call.TransferAgentId()
	isTransfer := call.IsTransfer()

	// TODO WTEL-4370
	ccXfer := strings.HasSuffix(call.GetVariable("variable_transfer_history"),
		fmt.Sprintf(":bl_xfer:%s/default/XML", call.Destination())) && call.GetVariable("variable_cc_app_id") != ""

	if transferSchemaId != nil && isTransfer {
		routing, err = r.fm.SearchTransferredRouting(call.DomainId(), *transferSchemaId)
	} else if transferQueueId != 0 && isTransfer {
		call.Log().Info("transfer from: " + call.From().String() + " to queue_id ")
		routing, _ = r.fm.TransferQueueRouting(call.DomainId(), transferQueueId)

	} else if transferAgentId != 0 && isTransfer {
		call.Log().Info("transfer from: " + call.From().String() + " to agent_id ")
		routing, _ = r.fm.TransferAgentRouting(call.DomainId(), transferAgentId)

	} else if isTransfer && queueId == nil && (ccXfer || !call.IsOriginateRequest()) {
		call.Log().Info("transfer from: " + call.From().String() + " to destination " + call.Destination())
		if routing, err = r.fm.SearchOutboundToDestinationRouting(call.DomainId(), call.Destination()); err == nil {
			call.outboundVars, err = getOutboundReg(routing.SourceData, call.Destination())
		}
		if call.Direction() == model.CallDirectionInbound {
			call.SetTransferFromId()
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
	call.SetDomainName(routing.DomainName)

	if err = call.SetSchemaId(routing.SchemaId); err != nil {
		call.Log().Err(err)
	}

	if meeting := call.MeetingId(); meeting != "" {
		vars, err2 := r.meeting.Get(call.Context(), meeting)
		if err2 != nil {
			call.Log().Error(err2.Error(), wlog.Err(err2))
		} else {
			call.Set(call.Context(), model.VariablesFromStringMap(vars))
		}
	}

	rawSchema := make([]map[string]any, len(routing.Schema.Schema))
	for i, app := range routing.Schema.Schema {
		rawSchema[i] = map[string]any(app)
	}
	tr, parseErr := tree.Parse(routing.SchemaId, rawSchema)
	if parseErr != nil {
		wlog.Error(fmt.Sprintf("call %s parse error: %s", call.Id(), parseErr.Error()))
		return
	}

	tags := make(map[string]string, len(tr.ByTag))
	for tag, node := range tr.ByTag {
		tags[tag] = node.ID
	}

	decorator := func(ctx context.Context) context.Context {
		return legacy.WithConnection(ctx, call)
	}

	schemaId := routing.SchemaId
	var activeRec *persistence.Record

	teardown := func() {
		// Wait for the call to fully disconnect before running the trigger.
		// If context is already done, this returns immediately.
		<-conn.Context().Done()

		if _, ok := tr.Triggers[flow.TriggerDisconnected]; ok {
			call.ClearExportVariables()
			var vars map[string]string
			if activeRec != nil {
				vars = activeRec.State.Variables
			}
			ctxDisc, cancel := context.WithDeadline(context.Background(), time.Now().Add(60*time.Second))
			defer cancel()
			ctxDisc = decorator(ctxDisc)
			if trigErr := r.driver.RunTrigger(ctxDisc, tr, flow.TriggerDisconnected, vars, call.DomainId(), call.Id()); trigErr != nil {
				call.Log().Error(fmt.Sprintf("call disconnect trigger: %v", trigErr))
			}
			r.fm.StoreCallVariables(call.Id(), call.DumpExportVariables())
		}

		r.fm.StoreLog(schemaId, conn.Id(), nil)
	}

	if _, createErr := runtimekit.RunSession(nil, runtimekit.HandleConfig{
		ChannelName: "call",
		ChannelType: callChannel,
		Conn:        call,
		Tr:          tr,
		Tags:        tags,
		SchemaID:    schemaId,
		DomainID:    call.DomainId(),
		AppID:       r.fm.AppID(),
		Repo:        r.fm.RuntimeStateRepo(),
		Driver:      r.driver,
		SessionMgr:  r.sessionMgr,
		Decorator:   decorator,
		Teardown:    teardown,
		OnRecord:    func(rec *persistence.Record) { activeRec = rec },
		Log:         r.fm.Log(),
	}); createErr != nil {
		wlog.Error(fmt.Sprintf("call %s runtime error: %s", call.Id(), createErr.Error()))
	}
}

func (r *Router) Decode(scope *flow.Flow, in interface{}, out interface{}) *model.AppError {
	return scope.Decode(in, out)
}
