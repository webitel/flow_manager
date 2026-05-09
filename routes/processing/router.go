package processing

import (
	"context"
	"fmt"
	"maps"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	ports "github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/coordinator"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	procop "github.com/webitel/flow_manager/internal/runtime/ops/domain/processing"
	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// processingChannel is the channel discriminator stored in flow.runtime_state.
// Matches model.ConnectionTypeForm (iota = 5).
const processingChannel = int16(model.ConnectionTypeForm)

type Router struct {
	fm         ports.RouterDeps
	apps       flow.ApplicationHandlers
	driver     *interpreter.Driver
	coord      coordinator.Coordinator
	sessionMgr *sessionmgr.Manager
}

// Connection is the interface that the processing transport layer must satisfy.
// PushForm is intentionally absent: the native generateForm op uses
// ProcessingConn.SendForm instead; legacy generate_form.go has been deleted.
type Connection interface {
	model.Connection
	SchemaId() int
	SetComponent(name string, component any)
	GetComponentByName(name string) any
	Export(ctx context.Context, vars []string)
	DumpExportVariables() map[string]string
}

func Init(deps ports.RouterDeps, fr flow.Router) model.Router {
	router := &Router{fm: deps}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	// coord is late-bound: nil when ExtraOps runs, set after Bootstrap returns.
	var coord coordinator.Coordinator

	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:   deps,
		Router: router,
		Apps:   router.apps,
		ExtraOps: func(reg *ops.Registry) {
			procop.Register(reg, procop.DispatchFunc(func(ctx context.Context, key string, payload map[string]string) error {
				if coord == nil {
					return nil
				}
				return coord.Dispatch(ctx, key, payload)
			}))
			procop.RegisterComponents(reg, deps)
			procop.RegisterAttempt(reg, deps)
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
		}
		return h.Handler(ctx, scope, req.Args())
	}
	return flow.Do(func(result *model.Result) {
		result.Err = model.NewAppError("Form.Request", "form.request.not_found", nil,
			fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
	})
}

func (r *Router) handle(conn model.Connection) {
	c := conn.(Connection)

	s, appErr := r.fm.GetSchemaById(conn.DomainId(), c.SchemaId())
	if appErr != nil {
		r.fm.Log().Error(fmt.Sprintf("processing: conn %s schema error: %s", conn.Id(), appErr.Error()))
		conn.Close()
		return
	}

	rawSchema := make([]map[string]any, len(s.Schema))
	for i, app := range s.Schema {
		rawSchema[i] = map[string]any(app)
	}
	tr, parseErr := tree.Parse(s.Id, rawSchema)
	if parseErr != nil {
		r.fm.Log().Error(fmt.Sprintf("processing: conn %s parse error: %s", conn.Id(), parseErr.Error()))
		conn.Close()
		return
	}

	tags := make(map[string]string, len(tr.ByTag))
	for tag, node := range tr.ByTag {
		tags[tag] = node.ID
	}

	decorator := func(ctx context.Context) context.Context {
		ctx = legacy.WithConnection(ctx, conn)
		if pc, ok := conn.(procop.ProcessingConn); ok {
			ctx = procop.WithConn(ctx, pc)
		}
		return ctx
	}

	if _, createErr := runtimekit.RunSession(nil, runtimekit.HandleConfig{
		ChannelName: "processing",
		ChannelType: processingChannel,
		Conn:        conn,
		Tr:          tr,
		Tags:        tags,
		SchemaID:    c.SchemaId(),
		DomainID:    conn.DomainId(),
		AppID:       r.fm.AppID(),
		Repo:        r.fm.RuntimeStateRepo(),
		Driver:      r.driver,
		SessionMgr:  r.sessionMgr,
		Decorator:   decorator,
		Teardown:    func() { conn.Close() },
		Log:         r.fm.Log(),
	}); createErr != nil {
		r.fm.Log().Error(fmt.Sprintf("processing: conn %s runtime error: %s", conn.Id(), createErr.Error()))
		conn.Close()
	}
}

func (r *Router) Decode(scope *flow.Flow, in, out any) *model.AppError {
	return scope.Decode(in, out)
}
