package app

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/cc"
	"net/http"
)

func (fm *FlowManager) JoinToInboundQueue(ctx context.Context, in *cc.CallJoinToQueueRequest) (cc.MemberService_CallJoinToQueueClient, error) {
	return fm.cc.Member().JoinCallToQueue(ctx, in)
}

func (fm *FlowManager) JoinChatToInboundQueue(ctx context.Context, in *cc.ChatJoinToQueueRequest) (cc.MemberService_ChatJoinToQueueClient, error) {
	return fm.cc.Member().JoinChatToQueue(ctx, in)
}

func (fm *FlowManager) AddMemberToQueueQueue(domainId int64, queueId int, number, name string, typeId, holdSec int, variables map[string]string) *model.AppError {
	return fm.Store.Call().AddMemberToQueueQueue(domainId, queueId, number, name, typeId, holdSec, variables)
}

func (fm *FlowManager) JoinToAgent(ctx context.Context, in *cc.CallJoinToAgentRequest) (cc.MemberService_CallJoinToAgentClient, error) {
	return fm.cc.Member().CallJoinToAgent(ctx, in)
}

func (fm *FlowManager) CancelUserDistribute(ctx context.Context, domainId int64, extension string) *model.AppError {
	agentId, err := fm.Store.User().GetAgentIdByExtension(domainId, extension)
	if err != nil {
		return err
	}

	if agentId == nil {
		return nil
	}

	_, perr := fm.cc.Member().CancelAgentDistribute(ctx, &cc.CancelAgentDistributeRequest{
		AgentId: *agentId,
	})

	if perr != nil {
		return model.NewAppError("App", "CancelUserDistribute", nil, err.Error(), http.StatusNotFound)
	}

	return nil
}
