package grpc_route

import (
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type Router struct {
	fm   *app.FlowManager
	apps model.ApplicationHandlers
}

func Init(fm *app.FlowManager) {
	fm.GRPCRouter = &Router{
		fm: fm,
		apps: model.UnionApplicationMap(
			fm.FlowRouter.Handlers(),
		),
	}
}

func (r *Router) Handlers() model.ApplicationHandlers {
	return r.apps
}

func (r *Router) Request(conn model.Connection, req model.ApplicationRequest) (model.Response, *model.AppError) {
	if h, ok := r.apps[req.Id()]; ok {
		if h.ArgsParser != nil {
			return h.Handler(conn, h.ArgsParser(conn, req.Args()))
		} else {
			return h.Handler(conn, req.Args())
		}

	}
	return nil, model.NewAppError("GRPC.Request", "grpc.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	gr := conn.(model.GRPCConnection)

	s, err := r.fm.GetSchema(gr.DomainId(), gr.SchemaId(), gr.SchemaUpdatedAt())
	if err != nil {
		return err
	}

	f := flow.New(flow.Config{
		Timezone: "",
		Name:     "grpc-con",
		Handler:  r,
		Apps:     s.Schema,
		Conn:     conn,
	})

	go func() {
		flow.Route(f, r)
		conn.Close()
	}()

	return nil
}
