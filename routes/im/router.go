package im

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/webitel/flow_manager/flow"
	ports "github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/coordinator"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/calendar"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/messaging"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
)

// imChannel is the channel discriminator stored in flow.runtime_state.
const imChannel int16 = 2

type Router struct {
	fm     ports.RouterDeps
	apps   flow.ApplicationHandlers
	driver *interpreter.Driver
	coord  coordinator.Coordinator
}

type Dialog model.IMDialog

func Init(deps ports.RouterDeps, fr flow.Router) model.Router {
	router := &Router{
		fm: deps,
	}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	delete(router.apps, "calendar")
	delete(router.apps, "softSleep")
	delete(router.apps, "recvMessage")

	reg := ops.NewRegistry()
	builtin.Register(reg)
	legacy.RegisterFromMap(reg, router, router.apps)

	reg.Register("calendar", calendar.New(func(ctx context.Context, domainID int64, id *int, name *string) (*calendar.Result, error) {
		cal, err := deps.GetStore().Calendar().Check(domainID, id, name)
		if err != nil {
			return nil, err
		}
		return &calendar.Result{
			Accept:   cal.Accept,
			Expire:   cal.Expire,
			Excepted: cal.Excepted,
		}, nil
	}))

	reg.Register("recvMessage", messaging.New())

	router.driver = interpreter.NewDriver(
		deps.RuntimeStateRepo(),
		reg,
		deps.Log(),
		func(ctx context.Context, domainID int64, name string) string {
			return deps.SchemaVariable(ctx, domainID, name)
		},
	)

	loadTree := func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error) {
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
	}
	router.coord = coordinator.New(deps.RuntimeStateRepo(), router.driver, loadTree)

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

	// Recovery: reconnected to an already-suspended flow — skip Run entirely.
	if rec != nil && rec.Status == state.StatusSuspended {
		// The message that triggered handle() is the intended response to the
		// suspended recv_message. Replay it immediately after registering the handler.
		initialMsg := conn.Variables()[model.ConversationStartMessageVariable]
		r.watchSuspended(conn, conv, rec, cp, i, initialMsg)
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
		r.watchSuspended(conn, conv, rec, cp, i, "")
		return
	}

	r.teardown(conn, conv, cp, i)
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

// watchSuspended keeps the IM connection alive while the flow is suspended,
// routing each inbound message through the coordinator to resume the flow.
// When the flow is no longer suspended (completed, failed, or timed out via
// timer wakeup) or the connection context is cancelled, teardown is called
// exactly once.
//
// initialMsg, when non-empty, is dispatched immediately after the handler is
// registered. This handles the recovery path where the triggering message
// arrived before the handler was registered.
func (r *Router) watchSuspended(conn model.Connection, conv Dialog, rec *persistence.Record, cp *session.Checkpoint, i *flow.Flow, initialMsg string) {
	var (
		once    sync.Once
		unregFn func()
	)

	done := make(chan struct{})
	teardownFn := func() {
		once.Do(func() {
			close(done)
			if unregFn != nil {
				unregFn()
			}
			r.teardown(conn, conv, cp, i)
		})
	}

	waitConn, ok := conv.(ports.WaitableConnection)
	if !ok {
		r.fm.Log().Warn(fmt.Sprintf("im watchSuspended: connection %s does not implement WaitableConnection", conn.Id()))
		teardownFn()
		return
	}

	connID := conn.Id()
	suspendKey := "msg:" + connID

	// dispatch sends an arbitrary payload to the coordinator and tears down if
	// the flow is no longer running/suspended afterwards.
	dispatch := func(payload map[string]string) {
		connCtx := legacy.WithConnection(conn.Context(), conv)
		connCtx = messaging.WithConnID(connCtx, connID)

		if err := r.coord.Dispatch(connCtx, suspendKey, payload); err != nil {
			r.fm.Log().Warn(fmt.Sprintf("im watchSuspended: dispatch error conn=%s: %v", connID, err))
		}

		latest, loadErr := r.fm.RuntimeStateRepo().LoadByConnectionID(conn.Context(), connID)
		if loadErr != nil {
			r.fm.Log().Warn(fmt.Sprintf("im watchSuspended: reload after dispatch failed conn=%s: %v", connID, loadErr))
			teardownFn()
			return
		}
		if latest == nil || (latest.Status != state.StatusSuspended && latest.Status != state.StatusRunning) {
			teardownFn()
		}
	}

	unregFn = waitConn.OnInboundMessage(func(text string) {
		dispatch(map[string]string{"msg": text})
	})

	// If the suspended op is recv_message with a wake_at deadline, fire a local
	// timer instead of relying on the DB polling worker. This gives accurate
	// short timeouts (e.g. 60s) and works across service restarts: if wake_at
	// is already in the past we dispatch immediately.
	//
	// No need to cancel the timer when a message arrives first — the atomic
	// claim in LoadByResumeKey ensures a second dispatch is always a no-op.
	if rec != nil && rec.State.Pending != nil && rec.State.Pending.OpName == "recv_message" {
		if wakeAtStr, ok := rec.State.Pending.Args["wake_at"]; ok {
			if wakeAt, parseErr := time.Parse(time.RFC3339, wakeAtStr); parseErr == nil {
				delay := time.Until(wakeAt)
				if delay <= 0 {
					go dispatch(map[string]string{"timeout": "true"})
				} else {
					time.AfterFunc(delay, func() {
						dispatch(map[string]string{"timeout": "true"})
					})
				}
			}
		}
	}

	// Replay the message that triggered the recovery path — it arrived before
	// the handler was registered so OnInboundMessage would have missed it.
	if initialMsg != "" {
		go dispatch(map[string]string{"msg": initialMsg})
	}

	// Ensure teardown when the connection context is cancelled OR when the
	// flow finishes via dispatch (done is closed inside teardownFn's once.Do).
	go func() {
		select {
		case <-conn.Context().Done():
		case <-done:
		}
		teardownFn()
	}()
}

func (r *Router) Decode(scope *flow.Flow, in, out any) *model.AppError {
	return scope.Decode(in, out)
}
