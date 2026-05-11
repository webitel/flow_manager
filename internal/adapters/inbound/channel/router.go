package channel

import (
	"context"
	"fmt"
	"maps"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	ports "github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
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
	fm         ports.RouterDeps
	apps       flow.ApplicationHandlers
	driver     *interpreter.Driver
	sessionMgr *sessionmgr.Manager
}

func Init(deps ports.RouterDeps, fr flow.Router) model.Router {
	r := &Router{fm: deps}
	r.apps = fr.Handlers()

	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:   deps,
		Router: r,
		Apps:   r.apps,
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
		}
		return h.Handler(ctx, scope, req.Args())
	}
	return flow.Do(func(result *model.Result) {
		result.Err = model.NewAppError("Channel.Request", "channel.request.not_found", nil,
			fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
	})
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
		return legacy.WithConnection(ctx, conn)
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

func (r *Router) Decode(scope *flow.Flow, in, out any) *model.AppError {
	return scope.Decode(in, out)
}
