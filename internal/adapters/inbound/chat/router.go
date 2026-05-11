package chat

import (
	"context"
	"fmt"
	"net/http"
	"time"

	proto "github.com/webitel/flow_manager/gen/chat"
	ports "github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	chatop "github.com/webitel/flow_manager/internal/runtime/ops/domain/chat"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/messaging"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
)

// chatChannel is the channel discriminator stored in flow.runtime_state.
// Matches model.ConnectionTypeChat (iota = 4).
const chatChannel int16 = 4

type Router struct {
	fm         ports.RouterDeps
	driver     *interpreter.Driver
	sessionMgr *sessionmgr.Manager
}

func Init(deps ports.RouterDeps) model.Router {
	router := &Router{
		fm: deps,
	}

	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:     deps,
		ExtraOps: func(reg *ops.Registry) {
			chatop.Register(reg, deps)
			chatop.RegisterSend(reg, deps)
			chatop.RegisterSTT(reg, deps)
			chatop.RegisterQueue(reg, deps)
			chatop.RegisterMisc(reg)
			chatop.RegisterRecv(reg)
		},
		LoadTree: func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error) {
			routing, appErr := deps.GetChatRouteFromSchemaId(domainID, int32(schemaID))
			if appErr != nil {
				return nil, appErr
			}
			if routing == nil {
				return nil, fmt.Errorf("chat: schema %d not found for domain %d", schemaID, domainID)
			}
			rawSchema := make([]map[string]any, len(routing.Schema.Schema))
			for i, app := range routing.Schema.Schema {
				rawSchema[i] = map[string]any(app)
			}
			return tree.Parse(routing.SchemaId, rawSchema)
		},
	})
	router.driver = kit.Driver
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
	conv := conn.(model.Conversation)
	var routing *model.Routing
	var err *model.AppError
	shId := conv.SchemaId()

	if shId > 0 {
		routing, err = r.fm.GetChatRouteFromSchemaId(conv.DomainId(), shId)
	} else if conv.UserId() > 0 {
		routing, _ = r.fm.GetChatRouteFromUserId(conv.DomainId(), conv.UserId())
	} else if conv.ProfileId() > 0 {
		routing, err = r.fm.GetChatRouteFromProfile(conv.DomainId(), conv.ProfileId())
	} else {
		// TODO ERROR
	}

	if routing == nil {
		err = model.NewAppError("Chat", "chat.routing.not_found", nil, "Not found routing schema", http.StatusBadRequest)
	}
	if err != nil {
		conv.Stop(err, proto.CloseConversationCause_flow_err)
		return
	}

	conn.Set(conn.Context(), map[string]any{
		model.FlowSchemaNameVariable: routing.Schema.Name,
	})

	rawSchema := make([]map[string]any, len(routing.Schema.Schema))
	for i, app := range routing.Schema.Schema {
		rawSchema[i] = map[string]any(app)
	}
	tr, parseErr := tree.Parse(routing.SchemaId, rawSchema)
	if parseErr != nil {
		conv.Stop(model.NewAppError("Chat", "chat.schema.parse", nil, parseErr.Error(), http.StatusInternalServerError), proto.CloseConversationCause_flow_err)
		return
	}

	tags := make(map[string]string, len(tr.ByTag))
	for tag, node := range tr.ByTag {
		tags[tag] = node.ID
	}

	ctx := conn.Context()

	rec, loadErr := r.fm.RuntimeStateRepo().LoadByConnectionID(ctx, conn.Id())
	if loadErr != nil {
		conv.Stop(model.NewAppError("Chat", "chat.runtime.load", nil, loadErr.Error(), http.StatusInternalServerError), proto.CloseConversationCause_flow_err)
		return
	}

	cp := session.Save(r.fm.CheckpointRepo(), r.fm.AppID(), conn, routing.SchemaId)

	var activeRec *persistence.Record
	decorator := func(ctx context.Context) context.Context {
		ctx = connctx.WithConnection(ctx, conv)
		ctx = messaging.WithConnID(ctx, conn.Id())
		if cw, ok := conn.(chatop.ChatWaitable); ok {
			ctx = chatop.WithChatWaitable(ctx, cw)
		}
		return ctx
	}
	teardownFn := func() {
		r.teardownNative(conn, conv, cp, tr, activeRec, decorator)
	}

	if _, createErr := runtimekit.RunSession(rec, runtimekit.HandleConfig{
		ChannelName: "chat",
		ChannelType: chatChannel,
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
		conv.Stop(model.NewAppError("Chat", "chat.runtime.create", nil, createErr.Error(), http.StatusInternalServerError), proto.CloseConversationCause_flow_err)
	}
}

func (r *Router) teardownNative(
	conn model.Connection,
	conv model.Conversation,
	cp *session.Checkpoint,
	tr *tree.Tree,
	rec *persistence.Record,
	decorate func(context.Context) context.Context,
) {
	session.Update(r.fm.CheckpointRepo(), cp, conn)

	if !conv.IsTransfer() {
		conv.Stop(nil, proto.CloseConversationCause_flow_end)
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
			r.fm.Log().Error(fmt.Sprintf("chat teardown: disconnect trigger conn=%s: %v", conn.Id(), trigErr))
		}
	}

	session.Close(r.fm.CheckpointRepo(), r.fm.Log(), cp, conn.Id())
}

