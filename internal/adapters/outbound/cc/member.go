package cc

import (
	"context"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

type memberApi struct {
	*grpcdial.Client[cc.MemberServiceClient]
}

func NewMemberApi(c *grpcdial.Client[cc.MemberServiceClient]) MemberApi {
	return &memberApi{
		Client: c,
	}
}

func (api *memberApi) JoinCallToQueue(ctx context.Context, in *cc.CallJoinToQueueRequest) (cc.MemberService_CallJoinToQueueClient, error) {
	return api.API.CallJoinToQueue(ctx, in)
}

func (api *memberApi) JoinChatToQueue(ctx context.Context, in *cc.ChatJoinToQueueRequest) (cc.MemberService_ChatJoinToQueueClient, error) {
	return api.API.ChatJoinToQueue(ctx, in)
}

func (api *memberApi) JoinIMToQueue(ctx context.Context, in *cc.IMJoinToQueueRequest) (*cc.IMJoinToQueueResponse, error) {
	return api.API.IMJoinToQueue(ctx, in)
}

func (api *memberApi) DirectAgentToMember(domainId, memberId int64, communicationId int, agentId int64) (int64, error) {
	res, err := api.API.DirectAgentToMember(context.Background(), &cc.DirectAgentToMemberRequest{
		MemberId:        memberId,
		AgentId:         agentId,
		CommunicationId: int32(communicationId),
		DomainId:        domainId,
	})
	if err != nil {
		return 0, err
	}

	return res.AttemptId, nil
}

func (api *memberApi) AttemptResult(result *cc.AttemptResultRequest) error {
	_, err := api.API.AttemptResult(context.Background(), result)
	if err != nil {
		return err
	}

	return nil
}

func (api *memberApi) RenewalResult(domainId, attemptId int64, renewal uint32) error {
	_, err := api.API.AttemptRenewalResult(context.Background(), &cc.AttemptRenewalResultRequest{
		DomainId:  domainId,
		AttemptId: attemptId,
		Renewal:   renewal,
	})

	return err
}

func (api *memberApi) CallJoinToAgent(ctx context.Context, in *cc.CallJoinToAgentRequest) (cc.MemberService_CallJoinToAgentClient, error) {
	return api.API.CallJoinToAgent(ctx, in)
}

func (api *memberApi) CallOutbound(ctx context.Context, in *cc.OutboundCallRequest) (*cc.OutboundCallResponse, error) {
	return api.API.OutboundCall(ctx, in)
}

func (api *memberApi) TaskJoinToAgent(ctx context.Context, in *cc.TaskJoinToAgentRequest) (cc.MemberService_TaskJoinToAgentClient, error) {
	return api.API.TaskJoinToAgent(ctx, in)
}

func (api *memberApi) CancelAgentDistribute(ctx context.Context, in *cc.CancelAgentDistributeRequest) (*cc.CancelAgentDistributeResponse, error) {
	return api.API.CancelAgentDistribute(ctx, in)
}

func (api *memberApi) ProcessingActionForm(ctx context.Context, in *cc.ProcessingFormActionRequest) (*cc.ProcessingFormActionResponse, error) {
	ctx2 := api.StaticHost(ctx, in.AppId)
	return api.API.ProcessingFormAction(ctx2, in)
}

func (api *memberApi) ProcessingActionComponent(ctx context.Context, in *cc.ProcessingComponentActionRequest) (*cc.ProcessingComponentActionResponse, error) {
	ctx2 := api.StaticHost(ctx, in.AppId)
	return api.API.ProcessingComponentAction(ctx2, in)
}

func (api *memberApi) CancelAttempt(ctx context.Context, attemptId int64, result, appId string) error {
	ctx2 := api.StaticHost(ctx, appId)

	_, err := api.API.CancelAttempt(ctx2, &cc.CancelAttemptRequest{
		AttemptId: attemptId,
		Result:    result,
	})

	return err
}

func (api *memberApi) InterceptAttempt(ctx context.Context, domainId, attemptId int64, agentId int32) error {
	_, err := api.API.InterceptAttempt(ctx, &cc.InterceptAttemptRequest{
		DomainId:  domainId,
		AttemptId: attemptId,
		AgentId:   agentId,
	})

	return err
}

func (api *memberApi) ResumeAttempt(ctx context.Context, attemptId, domainId int64) error {
	_, err := api.API.ResumeAttempt(ctx, &cc.ResumeAttemptRequest{
		DomainId:  domainId,
		AttemptId: attemptId,
	})

	return err
}

func (api *memberApi) SaveFormFields(domainId, attemptId int64, fields map[string]string, form []byte) error {
	_, err := api.API.ProcessingFormSave(context.Background(), &cc.ProcessingFormSaveRequest{
		DomainId:  domainId,
		AttemptId: attemptId,
		Fields:    fields,
		Form:      form,
	})

	return err
}
