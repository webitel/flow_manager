package grpc_route

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type Router struct {
	fm   *app.FlowManager
	apps flow.ApplicationHandlers
}

func Init(fm *app.FlowManager, fr flow.Router) {
	var router = &Router{
		fm: fm,
	}

	router.apps = flow.UnionApplicationMap(
		ApplicationsHandlers(router),
		fr.Handlers(),
	)

	fm.GRPCRouter = router
}

func (r *Router) GlobalVariable(domainId int64, name string) string {
	return r.fm.SchemaVariable(context.TODO(), domainId, name)
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
			result.Err = model.NewAppError("GRPC.Request", "grpc.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
		})
	}
}

func (r *Router) Handle(conn model.Connection) *model.AppError {

	go r.handle(conn)
	return nil
}

func (r *Router) handle(conn model.Connection) {
	gr := conn.(model.GRPCConnection)

	s, err := r.fm.GetSchemaById(conn.DomainId(), gr.SchemaId())
	if err != nil {
		wlog.Error(fmt.Sprintf("connection %s, error: %s", conn.Id(), err.Error()))
		conn.Close()
		return
	}

	i := flow.New(r, flow.Config{
		SchemaId: s.Id,
		Name:     s.Name,
		Schema:   s.Schema,
		Handler:  r,
		Conn:     conn,
		Timezone: "",
	})

	flow.Route(conn.Context(), i, r)
	conn.Close()
	r.disconnected(gr)
}

func (r *Router) disconnected(gr model.GRPCConnection) {
	scope := gr.Scope()
	if scope.Id != "" {
		switch scope.Channel {
		case "call":
			r.fm.StoreCallVariables(scope.Id, gr.DumpExportVariables())

		default:

		}
	}
}

func (r *Router) Decode(scope *flow.Flow, in interface{}, out interface{}) *model.AppError {
	return scope.Decode(in, out)
}
