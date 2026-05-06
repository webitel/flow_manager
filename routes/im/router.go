package im

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/flow"
	domcontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	ports "github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/coordinator"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	imop "github.com/webitel/flow_manager/internal/runtime/ops/domain/im"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/messaging"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
)

// imChannel is the channel discriminator stored in flow.runtime_state.
const imChannel int16 = 2

type Router struct {
	fm         ports.RouterDeps
	apps       flow.ApplicationHandlers
	driver     *interpreter.Driver
	coord      coordinator.Coordinator
	sessionMgr *sessionmgr.Manager
}

type Dialog model.IMDialog

func Init(deps ports.RouterDeps, fr flow.Router, contacts domcontacts.Client) model.Router {
	router := &Router{
		fm: deps,
	}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	delete(router.apps, "calendar")
	delete(router.apps, "softSleep")

	// coord is captured by the ExtraOps closure below. Bootstrap calls ExtraOps
	// synchronously before returning the kit, so coord is set after Bootstrap
	// returns. By the time a CC event fires and Dispatch is called, coord is
	// already assigned (late-binding pattern).
	var coord coordinator.Coordinator
	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:           deps,
		Router:         router,
		Apps:           router.apps,
		ContactsClient: contacts,
		ExtraOps: func(reg *ops.Registry) {
			reg.Register("recvMessage", messaging.New())
			imop.Register(reg, deps, imop.DispatchFunc(func(ctx context.Context, key string, payload map[string]string) error {
				if coord == nil {
					return nil
				}
				return coord.Dispatch(ctx, key, payload)
			}))
		},
		LoadTree: func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error) {
			routing, appErr := deps.GetChatRouteFromSchemaId(domainID, int32(schemaID))
			if appErr != nil {
				return nil, appErr
			}
			if routing == nil {
				return nil, fmt.Errorf("im: schema %d not found for domain %d", schemaID, domainID)
			}
			rawSchema := make([]map[string]any, len(routing.Schema.Schema))
			for i, app := range routing.Schema.Schema {
				rawSchema[i] = map[string]any(app)
			}
			return tree.Parse(routing.SchemaId, rawSchema)
		},
	})
	router.driver = kit.Driver
	router.coord = kit.Coord
	coord = kit.Coord
	router.sessionMgr = sessionmgr.New(kit.Coord, deps.RuntimeStateRepo(), deps.Log())

	return router
}

func (r *Router) GlobalVariable(domainId int64, name string) string {
	return r.fm.SchemaVariable(context.TODO(), domainId, name)
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	go r.handle(conn)
	return nil
}

func (r *Router) AddApplications(apps flow.ApplicationHandlers) flow.Handler {
	r2 := *r
	r2.apps = maps.Clone(r.apps)

	for k, v := range apps {
		r2.apps[k] = v
	}

	return &r2
}

func (r *Router) Request(ctx context.Context, scope *flow.Flow, req model.ApplicationRequest) <-chan model.Result {
	if h, ok := r.apps[req.Id()]; ok {
		if h.ArgsParser != nil {
			return h.Handler(ctx, scope, h.ArgsParser(scope.Connection, req.Args()))
		} else {
			return h.Handler(ctx, scope, req.Args())
		}
	} else {
		return flow.Do(func(result *model.Result) {
			result.Err = model.NewAppError("Chat.Request", "chat.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
		})
	}
}

func (r *Router) handle(conn model.Connection) {
	conv := conn.(Dialog)
	var routing *model.Routing
	var err *model.AppError
	shId := conv.SchemaId()

	if shId > 0 {
		routing, err = r.fm.GetChatRouteFromSchemaId(conv.DomainId(), int32(shId))
	} else {
		// TODO ERROR
	}

	if routing == nil {
		err = model.NewAppError("IM", "im.routing.not_found", nil, "Not found routing schema", http.StatusBadRequest)
	}
	if err != nil {
		conv.Stop(err)
		return
	}

	conn.Set(conn.Context(), map[string]any{
		model.FlowSchemaNameVariable: routing.Schema.Name,
	})

	// Convert model.Applications → tree.Schema.
	rawSchema := make([]map[string]any, len(routing.Schema.Schema))
	for i, app := range routing.Schema.Schema {
		rawSchema[i] = map[string]any(app)
	}
	tr, parseErr := tree.Parse(routing.SchemaId, rawSchema)
	if parseErr != nil {
		conv.Stop(model.NewAppError("IM", "im.schema.parse", nil, parseErr.Error(), http.StatusInternalServerError))
		return
	}

	// Build tag index for ExecState.
	tags := make(map[string]string, len(tr.ByTag))
	for tag, node := range tr.ByTag {
		tags[tag] = node.ID
	}

	ctx := conn.Context()

	// Check for an existing active record (process restart recovery).
	rec, loadErr := r.fm.RuntimeStateRepo().LoadByConnectionID(ctx, conn.Id())
	if loadErr != nil {
		conv.Stop(model.NewAppError("IM", "im.runtime.load", nil, loadErr.Error(), http.StatusInternalServerError))
		return
	}

	cp := session.Save(r.fm.CheckpointRepo(), r.fm.AppID(), conn, routing.SchemaId)

	// Legacy flow.New is still needed for the disconnect trigger in teardown.
	i := flow.New(r, flow.Config{
		SchemaId: routing.SchemaId,
		Name:     routing.Schema.Name,
		Schema:   routing.Schema.Schema,
		Handler:  r,
		Conn:     conv,
		Timezone: routing.TimezoneName,
	})

	// Legacy path: resumable runtime is disabled via config flag.
	if !r.fm.Config().Runtime.UseResumable.IMEnabled() {
		flow.Route(conn.Context(), i, r)
		r.teardown(conn, conv, cp, i)
		return
	}

	// Channel-specific dispatch context decoration: legacy adapters need the
	// connection in ctx, recv_message needs the connID for its SuspendKey.
	decorator := func(ctx context.Context) context.Context {
		ctx = legacy.WithConnection(ctx, conv)
		ctx = messaging.WithConnID(ctx, conn.Id())
		return ctx
	}
	teardownFn := func() {
		r.teardown(conn, conv, cp, i)
	}
	sessConn, ok := conv.(sessionmgr.Connection)
	if !ok {
		r.fm.Log().Warn(fmt.Sprintf("im handle: connection %s does not satisfy sessionmgr.Connection", conn.Id()))
		teardownFn()
		return
	}

	// Recovery: reconnected to an already-suspended flow — skip Run entirely.
	if rec != nil && rec.Status == state.StatusSuspended {
		// The message that triggered handle() is the intended response to the
		// suspended recv_message. Replay it immediately after registering the handler.
		initialMsg := conn.Variables()[model.ConversationStartMessageVariable]
		r.sessionMgr.Watch(sessConn, rec, initialMsg, decorator, teardownFn)
		return
	}

	if rec == nil {
		// Fresh start — seed state from connection variables.
		es := state.NewExecState(routing.SchemaId, tr.Version, tags)
		for k, v := range conn.Variables() {
			es.Variables[k] = v
		}
		rec = &persistence.Record{
			ConnectionID: conn.Id(),
			DomainID:     conv.DomainId(),
			Channel:      imChannel,
			SchemaID:     routing.SchemaId,
			AppID:        r.fm.AppID(),
			State:        es,
			Status:       state.StatusRunning,
		}
		if createErr := r.fm.RuntimeStateRepo().Create(ctx, rec); createErr != nil {
			conv.Stop(model.NewAppError("IM", "im.runtime.create", nil, createErr.Error(), http.StatusInternalServerError))
			return
		}
	}

	runCtx := legacy.WithConnection(ctx, conv)
	runCtx = messaging.WithConnID(runCtx, conn.Id())
	if runErr := r.driver.Run(runCtx, rec, tr, nil); runErr != nil {
		r.fm.Log().Error(fmt.Sprintf("IM driver.Run conn=%s: %v", conn.Id(), runErr))
	}

	// If the flow suspended waiting for an event, keep the connection alive
	// and watch for the resume trigger instead of tearing down.
	if rec.Status == state.StatusSuspended {
		r.sessionMgr.Watch(sessConn, rec, "", decorator, teardownFn)
		return
	}

	teardownFn()
}

// teardown finalises a completed or failed IM flow: updates the checkpoint,
// stops the connection, fires the disconnect trigger, and closes the session.
func (r *Router) teardown(conn model.Connection, conv Dialog, cp *session.Checkpoint, i *flow.Flow) {
	session.Update(r.fm.CheckpointRepo(), cp, conn)

	if !conv.IsTransfer() {
		conv.Stop(nil)
	}

	if d, err := i.TriggerScope(flow.TriggerDisconnected); err == nil {
		// TODO config
		ctxDisc, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		flow.Route(ctxDisc, d, r)
		<-ctxDisc.Done()
		cancel()
	}

	session.Close(r.fm.CheckpointRepo(), r.fm.Log(), cp, conn.Id())
}

func (r *Router) Decode(scope *flow.Flow, in, out any) *model.AppError {
	return scope.Decode(in, out)
}
