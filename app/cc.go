package app

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/cc"
)

func (fm *FlowManager) JoinToInboundQueue(ctx context.Context, in *cc.CallJoinToQueueRequest) (cc.MemberService_CallJoinToQueueClient, error) {
	return fm.cc.Member().JoinCallToQueue(ctx, in)
}

func (fm *FlowManager) JoinChatToInboundQueue(ctx context.Context, in *cc.ChatJoinToQueueRequest) (cc.MemberService_ChatJoinToQueueClient, error) {
	return fm.cc.Member().JoinChatToQueue(ctx, in)
}

func (fm *FlowManager) CreateMember(domainId int64, queueId int, holdSec int, member *model.CallbackMember) *model.AppError {
	return fm.Store.Member().CreateMember(domainId, queueId, holdSec, member)
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

func (fm *FlowManager) AttemptResult(result *model.AttemptResult) *model.AppError {
	err := fm.cc.Member().AttemptResult(result.Id, result.Status, result.Description, result.ReadyAt, result.ExpiredAt,
		result.Variables, result.StickyDisplay, result.AgentId)

	if err != nil {
		return model.NewAppError("AttemptResult", "app.attempt.result", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
