package chat

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	proto "github.com/webitel/flow_manager/gen/chat"
	"github.com/webitel/flow_manager/model"
)

type Router struct {
	fm   *app.FlowManager
	apps flow.ApplicationHandlers
}

type Conversation model.Conversation // TODO

func Init(fm *app.FlowManager, fr flow.Router) {
	router := &Router{
		fm: fm,
	}

	router.apps = flow.UnionApplicationMap(
		fr.Handlers(),
		ApplicationsHandlers(router),
	)

	fm.ChatRouter = router
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
	conv := conn.(Conversation)
	var routing *model.Routing
	var err *model.AppError
	shId := conv.SchemaId()

	if shId > 0 {
		routing, err = r.fm.GetChatRouteFromSchemaId(conv.DomainId(), shId)
	} else if conv.UserId() > 0 {
		routing, _ = r.fm.GetChatRouteFromUserId(conv.DomainId(), conv.UserId())
	} else if conv.ProfileId() > 0 {
		routing, err = r.fm.GetChatRouteFromProfile(conv.DomainId(), conv.ProfileId())
	} else {
		// TODO ERROR
	}

	if routing == nil {
		err = model.NewAppError("Chat", "chat.routing.not_found", nil, "Not found routing schema", http.StatusBadRequest)
	}
	if err != nil {
		conv.Stop(err, proto.CloseConversationCause_flow_err)
		return
	}

	i := flow.New(r, flow.Config{
		SchemaId: routing.SchemaId,
		Name:     routing.Schema.Name,
		Schema:   routing.Schema.Schema,
		Handler:  r,
		Conn:     conv,
		Timezone: routing.TimezoneName,
	})

	conn.Set(conn.Context(), map[string]any{
		model.FlowSchemaNameVariable: routing.Schema.Name,
	})

	flow.Route(conn.Context(), i, r)

	if !conv.IsTransfer() {
		conv.Stop(nil, proto.CloseConversationCause_flow_end)
	}

	if d, err := i.TriggerScope(flow.TriggerDisconnected); err == nil {
		// TODO config
		ctxDisc, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		flow.Route(ctxDisc, d, r)
		<-ctxDisc.Done()
		cancel()
	}
}

func (r *Router) Decode(scope *flow.Flow, in, out any) *model.AppError {
	return scope.Decode(in, out)
}
