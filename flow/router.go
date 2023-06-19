package flow

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
)

type Response struct {
	Status string
}

var ResponseOK = Response{"SUCCESS"}
var ResponseErr = Response{"FAIL"}

type router struct {
	fm   *app.FlowManager
	apps ApplicationHandlers
}

func (r Response) String() string {
	return r.Status
}

func NewRouter(fm *app.FlowManager) Router {
	var router = &router{
		fm: fm,
	}

	router.apps = ApplicationsHandlers(router)
	return router
}

func (r *router) Handlers() ApplicationHandlers {
	return r.apps
}

func ApplicationsHandlers(r *router) ApplicationHandlers {
	var apps = make(ApplicationHandlers)

	apps["log"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Log),
	}
	apps["if"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.conditionHandler),
	}
	apps["while"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.whileHandler),
	}
	apps["switch"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.switchHandler),
	}
	apps["execute"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.execute),
	}
	apps["set"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.set),
	}
	apps["break"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.breakHandler),
	}
	apps["httpRequest"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.httpRequest),
	}
	apps["string"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.stringApp),
	}
	apps["math"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Math),
	}
	apps["calendar"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Calendar),
	}
	apps["goto"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.GotoTag),
	}
	apps["list"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.List),
	}
	apps["listAdd"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.listAddCommunication),
	}
	apps["timezone"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.SetTimezone),
	}
	apps["softSleep"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.sleep),
	}
	apps["callbackQueue"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.callbackQueue),
	}
	apps["getQueueMetrics"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.getQueueMetrics),
	}
	apps["getQueueInfo"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.getQueueInfo),
	}
	apps["classifier"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.classifierHandler),
	}
	apps["js"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Js),
	}
	apps["userInfo"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.GetUser),
	}
	apps["sendEmail"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.sendEmail),
	}
	apps["generateLink"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.generateLink),
	}
	apps["ccPosition"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.QueueCallPosition),
	}
	apps["memberInfo"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.GetMember),
	}
	apps["patchMembers"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.PatchMembers),
	}
	apps["schema"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.schema),
	}
	apps["lastBridged"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.lastBridged),
	}
	apps["getQueueAgents"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.getQueueAgents),
	}
	apps["ewt"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.EWTCall),
	}
	apps["sql"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.SqlHandler),
	}
	apps["broadcastChatMessage"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.broadcastChatMessage),
	}
	apps["getEmail"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.getEmail),
	}
	apps["printFile"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.printFile),
	}
	apps["mq"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.mq),
	}
	apps["dump"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.dumpVarsHandler),
	}
	apps["cache"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.cache),
	}
	apps["chatHistory"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.chatHistory),
	}

	return apps
}

func (r *router) Handle(conn model.Connection) *model.AppError {
	return model.NewAppError("Flow", "flow.router.not_implement", nil, "not implement", http.StatusInternalServerError)
}

type flowHandler func(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError)

func (r *router) doExecute(delMe flowHandler) ApplicationHandler {
	return func(ctx context.Context, scope *Flow, args interface{}) model.ResultChannel {
		return Do(func(result *model.Result) {
			result.Res, result.Err = delMe(ctx, scope, scope.Connection, args)
		})
	}
}
