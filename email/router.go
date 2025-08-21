package email

import (
	"context"
	"fmt"
	"github.com/webitel/wlog"
	"maps"
	"net/http"

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

func (r *Router) GlobalVariable(domainId int64, name string) string {
	return r.fm.SchemaVariable(context.TODO(), domainId, name)
}

func (r *Router) AddApplications(apps flow.ApplicationHandlers) flow.Handler {
	r2 := *r
	r2.apps = maps.Clone(r.apps)

	for k, v := range apps {
		r2.apps[k] = v
	}

	return &r2
}

func (r *Router) Handlers() flow.ApplicationHandlers {
	return r.apps
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

func (r *Router) Handle(emailConnection model.Connection) *model.AppError {
	go r.handle(emailConnection)
	return nil
}

func (r *Router) handle(emailConnection model.Connection) {
	conn := &emailParser{
		timezoneName:    "",
		EmailConnection: emailConnection.(model.EmailConnection),
	}

	//conn := emailConnection.(model.EmailConnection)

	s, err := r.fm.GetSchemaById(conn.DomainId(), conn.SchemaId())
	if err != nil {
		wlog.Error(fmt.Sprintf("[%s] error: %s", conn.Id(), err.Error()))
		return
	}

	autoLink, _ := r.fm.GetSystemSettings(conn.Context(), conn.DomainId(), model.SysAutoLinkMailToContact)
	if autoLink.BoolValue {
		r.linkContact(conn)
	}

	f := flow.New(r, flow.Config{
		Timezone: "",
		Name:     s.Name,
		Handler:  r,
		Schema:   s.Schema,
		Conn:     conn, // e
	})

	flow.Route(conn.Context(), f, r)
}

func (r *Router) Decode(scope *flow.Flow, in interface{}, out interface{}) *model.AppError {
	return scope.Decode(in, out)
}
