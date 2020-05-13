package email

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
	r := &Router{
		fm: fm,
	}

	r.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(r),
	)

	fm.EmailRouter = r
}

func (r *Router) Handlers() flow.ApplicationHandlers {
	return r.apps
}

func (r *Router) Request(ctx context.Context, scope *flow.Flow, req model.ApplicationRequest) model.ResultChannel {
	return flow.Do(func(result *model.Result) {
		//if h, ok := r.apps[req.Id()]; ok {
		//	if h.ArgsParser != nil {
		//		h.Handler(scope, scope.Connection, h.ArgsParser(scope.Connection, req.Args()))
		//	} else {
		//		return h.Handler(scope, scope.Connection, req.Args())
		//	}
		//}
		//return nil, model.NewAppError("GRPC.Request", "grpc.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
	})
}

func (r *Router) Handle(conn model.Connection) *model.AppError {
	//e := &emailParser{
	//	timezoneName:    "",
	//	EmailConnection: conn.(model.EmailConnection),
	//}
	//
	//s, err := r.fm.GetSchemaById(1, 27)
	//if err != nil {
	//	return err
	//}
	//f := flow.New(flow.Config{
	//	Timezone: "",
	//	Name:     "email",
	//	Handler:  r,
	//	Schema:   s.Schema,
	//	Conn:     e,
	//})
	//
	//flow.Route(context.TODO(), f, r)
	//fmt.Println("END")
	return nil
}
