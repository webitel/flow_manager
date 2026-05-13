package channel

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// channelType is the channel discriminator stored in flow.runtime_state.
// Matches model.ConnectionTypeChannel (iota = 6).
const channelType = int16(model.ConnectionTypeChannel)

// schemaConn is the minimal interface the channel router needs from the
// underlying connection beyond model.Connection.
type schemaConn interface {
	SchemaId() int
}

type Router struct {
	fm         Deps
	driver     *interpreter.Driver
	sessionMgr *sessionmgr.Manager
}

func Init(deps Deps) model.Router {
	r := &Router{fm: deps}

	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:     deps,
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
	r.driver = kit.Driver
	r.sessionMgr = sessionmgr.New(kit.Coord, deps.RuntimeStateRepo(), deps.Log())

	return r
}

func (r *Router) GlobalVariable(domainId int64, name string) string {
	return r.fm.SchemaVariable(context.TODO(), domainId, name)
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	go r.handle(conn)
	return nil
}

func (r *Router) handle(conn model.Connection) {
	sc, ok := conn.(schemaConn)
	if !ok {
		r.fm.Log().Error(fmt.Sprintf("channel: conn %s does not expose SchemaId", conn.Id()))
		conn.Close()
		return
	}

	s, err := r.fm.GetSchemaById(conn.DomainId(), sc.SchemaId())
	if err != nil {
		r.fm.Log().Error(fmt.Sprintf("channel: conn %s schema error: %s", conn.Id(), err.Error()))
		conn.Close()
		return
	}

	rawSchema := make([]map[string]any, len(s.Schema))
	for i, app := range s.Schema {
		rawSchema[i] = map[string]any(app)
	}
	tr, parseErr := tree.Parse(s.Id, rawSchema)
	if parseErr != nil {
		r.fm.Log().Error(fmt.Sprintf("channel: conn %s parse error: %s", conn.Id(), parseErr.Error()))
		conn.Close()
		return
	}

	tags := make(map[string]string, len(tr.ByTag))
	for tag, node := range tr.ByTag {
		tags[tag] = node.ID
	}

	decorator := func(ctx context.Context) context.Context {
		return connctx.WithConnection(ctx, conn)
	}

	if _, createErr := runtimekit.RunSession(nil, runtimekit.HandleConfig{
		ChannelName: "channel",
		ChannelType: channelType,
		Conn:        conn,
		Tr:          tr,
		Tags:        tags,
		SchemaID:    sc.SchemaId(),
		DomainID:    conn.DomainId(),
		AppID:       r.fm.AppID(),
		Repo:        r.fm.RuntimeStateRepo(),
		Driver:      r.driver,
		SessionMgr:  r.sessionMgr,
		Decorator:   decorator,
		Teardown:    func() { conn.Close() },
		Log:         r.fm.Log(),
	}); createErr != nil {
		r.fm.Log().Error(fmt.Sprintf("channel: conn %s runtime error: %s", conn.Id(), createErr.Error()))
		conn.Close()
	}
}

