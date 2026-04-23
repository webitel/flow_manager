package cc

import (
	"context"

	genpb "github.com/webitel/flow_manager/gen/cc"
)

type AgentApi interface {
	Online(domainId, agentId int64, onDemand bool) error
	Offline(domainId, agentId int64) error
	Pause(domainId, agentId int64, payload string, timeout int) error

	WaitingChannel(agentId int, channel string) (int64, error)

	AcceptTask(appId string, domainId, attemptId int64) error
	CloseTask(appId string, domainId, attemptId int64) error
	RunTrigger(ctx context.Context, domainId, userId int64, triggerId int32, vars map[string]string) (string, error)
}

type MemberApi interface {
	AttemptResult(result *genpb.AttemptResultRequest) error
	RenewalResult(domainId, attemptId int64, renewal uint32) error

	JoinCallToQueue(ctx context.Context, in *genpb.CallJoinToQueueRequest) (genpb.MemberService_CallJoinToQueueClient, error)
	JoinChatToQueue(ctx context.Context, in *genpb.ChatJoinToQueueRequest) (genpb.MemberService_ChatJoinToQueueClient, error)
	CallJoinToAgent(ctx context.Context, in *genpb.CallJoinToAgentRequest) (genpb.MemberService_CallJoinToAgentClient, error)
	CallOutbound(ctx context.Context, in *genpb.OutboundCallRequest) (*genpb.OutboundCallResponse, error)
	TaskJoinToAgent(ctx context.Context, in *genpb.TaskJoinToAgentRequest) (genpb.MemberService_TaskJoinToAgentClient, error)
	JoinIMToQueue(ctx context.Context, in *genpb.IMJoinToQueueRequest) (*genpb.IMJoinToQueueResponse, error)

	DirectAgentToMember(domainId, memberId int64, communicationId int, agentId int64) (int64, error)
	CancelAgentDistribute(ctx context.Context, in *genpb.CancelAgentDistributeRequest) (*genpb.CancelAgentDistributeResponse, error)
	ProcessingActionForm(ctx context.Context, in *genpb.ProcessingFormActionRequest) (*genpb.ProcessingFormActionResponse, error)
	ProcessingActionComponent(ctx context.Context, in *genpb.ProcessingComponentActionRequest) (*genpb.ProcessingComponentActionResponse, error)
	SaveFormFields(domainId, attemptId int64, fields map[string]string, form []byte) error
	CancelAttempt(ctx context.Context, attemptId int64, result, appId string) error
	InterceptAttempt(ctx context.Context, domainId, attemptId int64, agentId int32) error
	ResumeAttempt(ctx context.Context, attemptId, domainId int64) error
}

type CCManager interface {
	Start() error
	Stop()

	Agent() AgentApi
	Member() MemberApi

	SubscribeAttempt(attemptId int64) <-chan QueueEvent
	UnSubscribeAttempt(attemptId int64)
}
