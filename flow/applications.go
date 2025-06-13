package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

func ApplicationsHandlers(r *router) ApplicationHandlers {
	apps := make(ApplicationHandlers)

	apps["log"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.Log),
	}
	apps["start"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.start),
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
	apps["monoPay"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.monopayHandler),
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
	apps["getContact"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.getContact),
	}
	apps["findContact"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.findContact),
	}
	apps["linkContact"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.linkContact),
	}
	apps["updateContact"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.updateContact),
	}
	apps["addContact"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.addContact),
	}
	apps["notification"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.notification),
	}
	apps["joinAgentToTask"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.joinAgentToTask),
	}
	apps["global"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.global),
	}
	apps["topicExtraction"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.topicExtraction),
	}
	apps["getCases"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.getCases),
	}
	apps["locateCase"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.locateCase),
	}
	apps["updateCase"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.updateCase),
	}
	apps["createCase"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.createCase),
	}
	apps["linkCommunication"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.linkCommunication),
	}
	apps["getServiceCatalogs"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.getServiceCatalogs),
	}
	apps["publishComment"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.publishComment),
	}
	apps["makeCall"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.makeCall),
	}
	apps["unSet"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.unSet),
	}
	apps["createLink"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.createLink),
	}
	apps["deleteLink"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.deleteLink),
	}
	apps["locateService"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.locateService),
	}
	apps["createRelatedCase"] = &Application{
		AllowNoConnect: true,
		Handler:        r.doExecute(r.createRelatedCase),
	}
	return apps
}

func (r *router) start(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	return model.CallResponseOK, nil
}
