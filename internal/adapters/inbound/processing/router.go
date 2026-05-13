package processing

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/coordinator"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	procop "github.com/webitel/flow_manager/internal/runtime/ops/domain/processing"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/runtimekit"
	"github.com/webitel/flow_manager/internal/runtime/sessionmgr"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/model"
)

// processingChannel is the channel discriminator stored in flow.runtime_state.
// Matches model.ConnectionTypeForm (iota = 5).
const processingChannel = int16(model.ConnectionTypeForm)

type Router struct {
	fm         Deps
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

func Init(deps Deps) model.Router {
	router := &Router{fm: deps}

	// coord is late-bound: nil when ExtraOps runs, set after Bootstrap returns.
	var coord coordinator.Coordinator

	kit := runtimekit.Bootstrap(runtimekit.Config{
		Deps:     deps,
		ExtraOps: func(reg *ops.Registry) {
			procop.Register(reg, procop.DispatchFunc(func(ctx context.Context, key string, payload map[string]string) error {
				if coord == nil {
					return nil
				}
				return coord.Dispatch(ctx, key, payload)
			}))
			procop.RegisterComponents(reg, deps)
			procop.RegisterAttempt(reg, deps)
			procop.RegisterFormTable(reg)
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
		ctx = connctx.WithConnection(ctx, conn)
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

