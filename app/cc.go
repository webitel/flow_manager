package app

import (
	"context"
	"net/http"

	genpb "github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/gen/engine"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) JoinToInboundQueue(ctx context.Context, in *genpb.CallJoinToQueueRequest) (genpb.MemberService_CallJoinToQueueClient, error) {
	return fm.cc.Member().JoinCallToQueue(ctx, in)
}

func (fm *FlowManager) CallOutboundQueue(ctx context.Context, in *genpb.OutboundCallRequest) (*genpb.OutboundCallResponse, error) {
	return fm.cc.Member().CallOutbound(ctx, in)
}

func (fm *FlowManager) JoinChatToInboundQueue(ctx context.Context, in *genpb.ChatJoinToQueueRequest) (genpb.MemberService_ChatJoinToQueueClient, error) {
	return fm.cc.Member().JoinChatToQueue(ctx, in)
}

func (fm *FlowManager) CreateMember(domainId int64, queueId, holdSec int, member *model.CallbackMember) *model.AppError {
	return fm.Store.Member().CreateMember(domainId, queueId, holdSec, member)
}

func (fm *FlowManager) JoinToAgent(ctx context.Context, in *genpb.CallJoinToAgentRequest) (genpb.MemberService_CallJoinToAgentClient, error) {
	return fm.cc.Member().CallJoinToAgent(ctx, in)
}

func (fm *FlowManager) TaskJoinToAgent(ctx context.Context, in *genpb.TaskJoinToAgentRequest) (genpb.MemberService_TaskJoinToAgentClient, error) {
	return fm.cc.Member().TaskJoinToAgent(ctx, in)
}

func (fm *FlowManager) CancelUserDistribute(ctx context.Context, domainId int64, extension string) *model.AppError {
	agentId, err := fm.Store.User().GetAgentIdByExtension(domainId, extension)
	if err != nil {
		return err
	}

	if agentId == nil {
		return nil
	}

	_, perr := fm.cc.Member().CancelAgentDistribute(ctx, &genpb.CancelAgentDistributeRequest{
		AgentId: *agentId,
	})

	if perr != nil {
		return model.NewAppError("App", "CancelUserDistribute", nil, err.Error(), http.StatusNotFound)
	}

	return nil
}

func (fm *FlowManager) AttemptResult(result *model.AttemptResult) *model.AppError {
	req := &genpb.AttemptResultRequest{
		AttemptId:                   result.Id,
		Status:                      result.Status,
		Variables:                   result.Variables,
		Display:                     result.StickyDisplay,
		Description:                 result.Description,
		AgentId:                     result.AgentId,
		Redial:                      result.Redial,
		ExcludeCurrentCommunication: result.ExcludeCurrentCommunication,
	}

	if result.ExpiredAt != nil {
		req.ExpireAt = *result.ExpiredAt
	}

	if result.ReadyAt != nil {
		req.NextDistributeAt = *result.ReadyAt
	}

	if result.WaitBetweenRetries != nil {
		req.WaitBetweenRetries = *result.WaitBetweenRetries
	}

	req.AddCommunications = ccCommunications(result.AddCommunications)

	err := fm.cc.Member().AttemptResult(req)
	if err != nil {
		return model.NewAppError("AttemptResult", "app.attempt.result", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (fm *FlowManager) JoinIMToInboundQueue(ctx context.Context, in *genpb.IMJoinToQueueRequest) (int64, <-chan domcc.QueueEvent, error) {
	res, err := fm.cc.Member().JoinIMToQueue(ctx, in)
	if err != nil {
		return 0, nil, err
	}

	ch := fm.cc.SubscribeAttempt(res.AttemptId)
	return res.AttemptId, ch, err
}

func (fm *FlowManager) LeavingIMToInboundQueue(attId int64) {
	fm.cc.UnSubscribeAttempt(attId)
}

func ccCommunications(r []model.CallbackCommunication) []*genpb.MemberCommunicationCreateRequest {
	l := len(r)

	if l == 0 {
		return nil
	}

	var comm []*genpb.MemberCommunicationCreateRequest
	if l != 0 {
		comm = make([]*genpb.MemberCommunicationCreateRequest, 0, l)
		for _, v := range r {
			if v.Destination != "" && v.Type.Id != nil {
				c := &genpb.MemberCommunicationCreateRequest{
					Destination: v.Destination,
					Type: &engine.Lookup{
						Id: int64(*v.Type.Id),
					},
				}

				if v.Priority != nil {
					c.Priority = int32(*v.Priority)
				}

				if v.Description != nil {
					c.Description = *v.Description
				}

				if v.ResourceId != nil {
					c.Resource = &engine.Lookup{
						Id: int64(*v.ResourceId),
					}
				}

				if v.Display != nil {
					c.Display = *v.Display
				}

				comm = append(comm, c)
			}
		}
	}

	return comm
}

func (fm *FlowManager) CancelAttempt(ctx context.Context, att model.InQueueKey, result string) *model.AppError {
	err := fm.cc.Member().CancelAttempt(ctx, att.AttemptId, result, att.AppId)
	if err != nil {
		return model.NewAppError("CancelAttempt", "app.attempt.cancel", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (fm *FlowManager) ResumeAttempt(ctx context.Context, attemptId, domainId int64) error {
	return fm.cc.Member().ResumeAttempt(ctx, attemptId, domainId)
}
