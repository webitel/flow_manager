package email

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
	r := &Router{
		fm: fm,
	}

	r.apps = model.UnionApplicationMap(
		fm.FlowRouter.Handlers(),
		ApplicationsHandlers(r),
	)

	fm.EmailRouter = r
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
	e := &emailParser{
		timezoneName:    "",
		EmailConnection: conn.(model.EmailConnection),
	}

	s, err := r.fm.GetSchemaById(1, 27)
	if err != nil {
		return err
	}
	f := flow.New(flow.Config{
		Timezone: "",
		Name:     "email",
		Handler:  r,
		Apps:     s.Schema,
		Conn:     e,
	})

	flow.Route(f, r)
	fmt.Println("END")
	return nil
}
