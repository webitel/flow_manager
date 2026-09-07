package im

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type Router struct {
	fm                     *app.FlowManager
	apps                   flow.ApplicationHandlers
	userInfoHandlersFabric *IMUserInfoHandlerFactory
}

type Dialog model.IMDialog

func Init(fm *app.FlowManager, fr flow.Router) {
	router := &Router{
		fm: fm,
	}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	router.userInfoHandlersFabric = NewIMUserInfoHandlerFactory(&IMUserInfoDeviceHandler{}, &IMUserInfoGateHandler{})

	fm.IMRouter = router
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

	maps.Copy(r2.apps, apps)

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
		result.Err = model.NewAppError("Chat.Request", "chat.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
	})
}

func (r *Router) handle(conn model.Connection) {
	conv := conn.(Dialog)
	if err := r.runSchema(conn, conv, conv.SchemaId(), conn.Context(), ""); err != nil {
		conv.Stop(err)
	} else {
		conv.Stop(nil)
	}
}

func (r *Router) runSchema(conn model.Connection, conv Dialog, shId int, ctx context.Context, cid string) *model.AppError {
	var routing *model.Routing
	var err *model.AppError

	if shId > 0 {
		routing, err = r.fm.GetChatRouteFromSchemaId(conv.DomainId(), int32(shId))
	}
	if routing == nil {
		err = model.NewAppError("IM", "im.routing.not_found", nil, "Not found routing schema", http.StatusBadRequest)
	}
	if err != nil {
		return err
	}

	i := flow.New(r, flow.Config{
		SchemaId: routing.SchemaId,
		Name:     routing.Schema.Name,
		Schema:   routing.Schema.Schema,
		Handler:  r,
		Conn:     conv,
		Timezone: routing.TimezoneName,
	})

	conn.Set(ctx, map[string]any{
		model.FlowSchemaNameVariable: routing.Schema.Name,
	})

	flow.Route(ctx, i, r)

	if conv.IsTransfer() {
		newCtx := conv.NewContext()
		schemaId, c2 := conv.TransferredSchema()
		if err = r.runSchema(conn, conv, schemaId, newCtx, c2); err != nil {
			return err
		}
		i.ClearCancel()
		flow.Route(conv.NewContext(), i, r)
	}

	if cid != "" {
		conv.Complete(cid)
	}

	if d, err := i.TriggerScope(flow.TriggerDisconnected); err == nil {
		ctxDisc, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		flow.Route(ctxDisc, d, r)
		<-ctxDisc.Done()
		cancel()
	}

	return nil
}

func (r *Router) Decode(scope *flow.Flow, in, out any) *model.AppError {
	return scope.Decode(in, out)
}
