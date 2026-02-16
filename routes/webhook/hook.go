package webhook

import (
	"context"
	"fmt"
	"maps"
	"net/http"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	httpprovider "github.com/webitel/flow_manager/providers/http"
)

type Router struct {
	fm   *app.FlowManager
	apps flow.ApplicationHandlers
}

func Init(fm *app.FlowManager, fr flow.Router) {
	router := &Router{
		fm: fm,
	}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	fm.WebHookRouter = router
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

func (r *Router) AddApplications(apps flow.ApplicationHandlers) flow.Handler {
	r2 := *r
	r2.apps = maps.Clone(r.apps)

	for k, v := range apps {
		r2.apps[k] = v
	}

	return &r2
}

func (r *Router) handle(conn model.Connection) {
	gr := conn.(*httpprovider.Connection)

	s, err := r.fm.GetSchemaById(conn.DomainId(), gr.SchemaId())
	if err != nil {
		wlog.Error(fmt.Sprintf("connection %s, error: %s", conn.Id(), err.Error()))
		conn.Close()
		return
	}

	i := flow.New(r, flow.Config{
		Name:     s.Name,
		Schema:   s.Schema,
		Handler:  r,
		Conn:     conn,
		Timezone: "",
	})

	flow.Route(conn.Context(), i, r)
	conn.Close()
}

func (r *Router) Decode(scope *flow.Flow, in, out any) *model.AppError {
	return scope.Decode(in, out)
}
