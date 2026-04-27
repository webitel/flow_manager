package im

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/flow"
	ports "github.com/webitel/flow_manager/internal/domain/shared/ports"
	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/calendar"
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

	router.driver = interpreter.NewDriver(
		deps.RuntimeStateRepo(),
		reg,
		deps.Log(),
		func(ctx context.Context, domainID int64, name string) string {
			return deps.SchemaVariable(ctx, domainID, name)
		},
	)

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

	cp := session.Save(r.fm.CheckpointRepo(), r.fm.AppID(), conn, routing.SchemaId)

	// Legacy flow.New is still needed for the disconnect trigger below.
	i := flow.New(r, flow.Config{
		SchemaId: routing.SchemaId,
		Name:     routing.Schema.Name,
		Schema:   routing.Schema.Schema,
		Handler:  r,
		Conn:     conv,
		Timezone: routing.TimezoneName,
	})

	runCtx := legacy.WithConnection(ctx, conv)
	if runErr := r.driver.Run(runCtx, rec, tr); runErr != nil {
		r.fm.Log().Error(fmt.Sprintf("IM driver.Run conn=%s: %v", conn.Id(), runErr))
	}

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
