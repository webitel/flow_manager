package grpc_route

import (
	"context"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type Router struct {
	fm   *app.FlowManager
	apps flow.ApplicationHandlers
}

func Init(fm *app.FlowManager, fr flow.Router) {
	fm.GRPCRouter = &Router{
		fm: fm,
		apps: flow.UnionApplicationMap(
			fr.Handlers(),
		),
	}
}

func (r *Router) Request(ctx context.Context, scope *flow.Flow, req model.ApplicationRequest) model.ResultChannel {
	if h, ok := r.apps[req.Id()]; ok {
		if h.ArgsParser != nil {
			h.Handler(ctx, scope, h.ArgsParser(scope.Connection, req.Args()))
		} else {
			return h.Handler(ctx, scope, req.Args())
		}
	} else {

	}

	return nil
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	//gr := conn.(model.GRPCConnection)
	//
	//s, err := r.fm.GetSchema(gr.DomainId(), gr.SchemaId(), gr.SchemaUpdatedAt())
	//if err != nil {
	//	return err
	//}
	//
	//f := flow.New(flow.Config{
	//	Timezone: "",
	//	Name:     "grpc-con",
	//	Handler:  r,
	//	Apps:     s.Schema,
	//	Conn:     conn,
	//})
	//
	//go func() {
	//	flow.Route(f, r)
	//	conn.Close()
	//}()

	return nil
}
