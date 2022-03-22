package chat_route

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"net/http"
	"time"
)

type Router struct {
	fm   *app.FlowManager
	apps flow.ApplicationHandlers
}

type Conversation interface {
	model.Connection
	ProfileId() int64
	Stop(*model.AppError)
	SendTextMessage(ctx context.Context, text string) (model.Response, *model.AppError)
	SendMenu(ctx context.Context, menu *model.ChatMenuArgs) (model.Response, *model.AppError)
	SendImageMessage(ctx context.Context, url string) (model.Response, *model.AppError)
	ReceiveMessage(ctx context.Context, name string, timeout int) ([]string, *model.AppError)
	Bridge(ctx context.Context, userId int64, timeout int) *model.AppError
	Export(ctx context.Context, vars []string) (model.Response, *model.AppError)
	DumpExportVariables() map[string]string
	NodeName() string
	SchemaId() int32
	BreakCause() string
	SendFile(ctx context.Context, text string, f *model.File) (model.Response, *model.AppError)
}

func Init(fm *app.FlowManager, fr flow.Router) {
	var router = &Router{
		fm: fm,
	}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	fm.ChatRouter = router
}

func (r *Router) Handle(conn model.Connection) *model.AppError {

	go r.handle(conn)
	return nil
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
	conv := conn.(Conversation)
	var routing *model.Routing
	var err *model.AppError
	shId := conv.SchemaId()

	if shId > 0 {
		routing, err = r.fm.GetChatRouteFromSchemaId(conv.DomainId(), shId)
	} else if conv.ProfileId() > 0 {
		routing, err = r.fm.GetChatRouteFromProfile(conv.DomainId(), conv.ProfileId())
	} else {
		//TODO ERROR
	}
	if err != nil {
		conv.Stop(err)
		return
	}

	i := flow.New(flow.Config{
		Name:     routing.Schema.Name,
		Schema:   routing.Schema.Schema,
		Handler:  r,
		Conn:     conv,
		Timezone: routing.TimezoneName,
	})

	flow.Route(conn.Context(), i, r)

	// todo fixme
	if conv.BreakCause() != "transfer" {
		conv.Stop(nil)
	}

	if d, err := i.TriggerScope(flow.TriggerDisconnected); err == nil {
		//TODO config
		ctxDisc, _ := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		flow.Route(ctxDisc, d, r)
		<-ctxDisc.Done()
	}
}

func (r *Router) Decode(scope *flow.Flow, in interface{}, out interface{}) *model.AppError {
	return scope.Decode(in, out)
}
