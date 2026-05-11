package im

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	driver     *interpreter.Driver
	coord      coordinator.Coordinator
	sessionMgr *sessionmgr.Manager
}

type Dialog model.IMDialog

func Init(deps ports.RouterDeps, contacts domcontacts.Client) model.Router {
	router := &Router{
		fm: deps,
	}

	// coord is captured by the ExtraOps closure below. Bootstrap calls ExtraOps
	// synchronously before returning the kit, so coord is set after Bootstrap
	// returns. By the time a CC event fires and Dispatch is called, coord is
	// already assigned (late-binding pattern).
	var coord coordinator.Coordinator
	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:           deps,
		ContactsClient: contacts,
		ExtraOps: func(reg *ops.Registry) {
			reg.Register("recvMessage", messaging.New())
			imop.Register(reg, deps, imop.DispatchFunc(func(ctx context.Context, key string, payload map[string]string) error {
				if coord == nil {
					return nil
				}
				if err := coord.Dispatch(ctx, key, payload); err != nil {
					return err
				}
				// After dispatching a CC event the flow may have completed.
				// Check the record and stop the connection so sessionmgr
				// tears down immediately instead of waiting for the next message.
				connID := messaging.ConnIDFromContext(ctx)
				if connID == "" {
					return nil
				}
				rec, _ := deps.RuntimeStateRepo().LoadByConnectionID(ctx, connID)
				if rec == nil || (rec.Status != state.StatusRunning && rec.Status != state.StatusSuspended) {
					if conn := legacy.ConnectionFromContext(ctx); conn != nil {
						if d, ok := conn.(model.IMDialog); ok {
							d.Stop(nil)
						}
					}
				}
				return nil
			}))
			imop.RegisterSend(reg, deps)
			imop.RegisterMenu(reg)
			imop.RegisterUnSet(reg)
			imop.RegisterExport(reg)
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

	// activeRec is set via OnRecord once the persistence.Record is established
	// (either loaded from DB or freshly created). Teardown reads variables from
	// it to pass to the disconnect trigger.
	var activeRec *persistence.Record

	// Channel-specific dispatch context decoration: legacy adapters need the
	// connection in ctx, recv_message needs the connID for its SuspendKey.
	decorator := func(ctx context.Context) context.Context {
		ctx = legacy.WithConnection(ctx, conv)
		ctx = messaging.WithConnID(ctx, conn.Id())
		return ctx
	}
	teardownFn := func() {
		r.teardown(conn, conv, cp, tr, activeRec, decorator)
	}
	if _, createErr := runtimekit.RunSession(rec, runtimekit.HandleConfig{
		ChannelName: "im",
		ChannelType: imChannel,
		Conn:        conn,
		Tr:          tr,
		Tags:        tags,
		SchemaID:    routing.SchemaId,
		DomainID:    conv.DomainId(),
		AppID:       r.fm.AppID(),
		Repo:        r.fm.RuntimeStateRepo(),
		Driver:      r.driver,
		SessionMgr:  r.sessionMgr,
		Decorator:   decorator,
		Teardown:    teardownFn,
		OnRecord:    func(r *persistence.Record) { activeRec = r },
		Log:         r.fm.Log(),
	}); createErr != nil {
		conv.Stop(model.NewAppError("IM", "im.runtime.create", nil, createErr.Error(), http.StatusInternalServerError))
	}
}

// teardown finalises a completed or failed IM flow: updates the checkpoint,
// stops the connection, fires the disconnect trigger via the native driver,
// and closes the session.
func (r *Router) teardown(
	conn model.Connection,
	conv Dialog,
	cp *session.Checkpoint,
	tr *tree.Tree,
	rec *persistence.Record,
	decorate func(context.Context) context.Context,
) {
	session.Update(r.fm.CheckpointRepo(), cp, conn)

	if !conv.IsTransfer() {
		conv.Stop(nil)
	}

	if _, ok := tr.Triggers["disconnected"]; ok {
		var vars map[string]string
		if rec != nil {
			vars = rec.State.Variables
		}
		ctxDisc, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		defer cancel()
		ctxDisc = decorate(ctxDisc)
		if trigErr := r.driver.RunTrigger(ctxDisc, tr, "disconnected", vars, conv.DomainId(), conn.Id()); trigErr != nil {
			r.fm.Log().Error(fmt.Sprintf("im teardown: disconnect trigger conn=%s: %v", conn.Id(), trigErr))
		}
	}

	session.Close(r.fm.CheckpointRepo(), r.fm.Log(), cp, conn.Id())
}
