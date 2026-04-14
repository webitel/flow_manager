package cc

import (
	"context"
	"sync"

	"github.com/webitel/engine/pkg/wbt"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
)

const ServiceName = "call_center"

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
	AttemptResult(result *cc.AttemptResultRequest) error
	RenewalResult(domainId, attemptId int64, renewal uint32) error

	JoinCallToQueue(ctx context.Context, in *cc.CallJoinToQueueRequest) (cc.MemberService_CallJoinToQueueClient, error)
	JoinChatToQueue(ctx context.Context, in *cc.ChatJoinToQueueRequest) (cc.MemberService_ChatJoinToQueueClient, error)
	CallJoinToAgent(ctx context.Context, in *cc.CallJoinToAgentRequest) (cc.MemberService_CallJoinToAgentClient, error)
	CallOutbound(ctx context.Context, in *cc.OutboundCallRequest) (*cc.OutboundCallResponse, error)
	TaskJoinToAgent(ctx context.Context, in *cc.TaskJoinToAgentRequest) (cc.MemberService_TaskJoinToAgentClient, error)
	JoinIMToQueue(ctx context.Context, in *cc.IMJoinToQueueRequest) (*cc.IMJoinToQueueResponse, error)

	DirectAgentToMember(domainId, memberId int64, communicationId int, agentId int64) (int64, error)
	CancelAgentDistribute(ctx context.Context, in *cc.CancelAgentDistributeRequest) (*cc.CancelAgentDistributeResponse, error)
	ProcessingActionForm(ctx context.Context, in *cc.ProcessingFormActionRequest) (*cc.ProcessingFormActionResponse, error)
	ProcessingActionComponent(ctx context.Context, in *cc.ProcessingComponentActionRequest) (*cc.ProcessingComponentActionResponse, error)
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

	SubscribeAttempt(attemptId int64) <-chan model.CCQueueEvent
	UnSubscribeAttempt(attemptId int64)
}

type ccManager struct {
	startOnce  sync.Once
	consulAddr string

	agentClient  *wbt.Client[cc.AgentServiceClient]
	memberClient *wbt.Client[cc.MemberServiceClient]

	agent    AgentApi
	member   MemberApi
	events   <-chan model.CCQueueEvent
	attempts map[int64]chan model.CCQueueEvent
	closed   chan struct{}
	sync.RWMutex
}

type AttemptEvent struct {
	cc *ccManager
	model.CCQueueEvent
}

func NewCCManager(consulAddr string, events <-chan model.CCQueueEvent) CCManager {
	cli := &ccManager{
		consulAddr: consulAddr,
		events:     events,
		closed:     make(chan struct{}),
		attempts:   make(map[int64]chan model.CCQueueEvent),
	}

	return cli
}

func (cm *ccManager) Agent() AgentApi {
	return cm.agent
}

func (cm *ccManager) Member() MemberApi {
	return cm.member
}

func (cm *ccManager) Start() error {
	wlog.Debug("starting cc service")
	var err error

	cm.startOnce.Do(func() {
		cm.agentClient, err = wbt.NewClient(cm.consulAddr, ServiceName, cc.NewAgentServiceClient)
		if err != nil {
			return
		}

		cm.memberClient, err = wbt.NewClient(cm.consulAddr, ServiceName, cc.NewMemberServiceClient)
		if err != nil {
			return
		}

		cm.agent = NewAgentApi(cm.agentClient)
		cm.member = NewMemberApi(cm.memberClient)
		go cm.listenEvents()
	})
	return err
}

func (cm *ccManager) Stop() {
	close(cm.closed)
}

func (cm *ccManager) listenEvents() {
	for {
		select {
		case event := <-cm.events:
			a, ok := cm.attempts[event.AttemptId]
			if !ok {
				continue
			}
			a <- event
		case <-cm.closed:
			return
		}
	}
}

func (cm *ccManager) UnSubscribeAttempt(attemptId int64) {
	cm.Lock()
	ch, ok := cm.attempts[attemptId]
	if ok {
		close(ch)
		delete(cm.attempts, attemptId)
	}
	cm.Unlock()
}

func (cm *ccManager) SubscribeAttempt(attemptId int64) <-chan model.CCQueueEvent {
	cm.Lock()
	e, ok := cm.attempts[attemptId]
	cm.Unlock()
	if ok {
		// todo warn
		return e
	}

	e = make(chan model.CCQueueEvent)
	cm.Lock()
	cm.attempts[attemptId] = e
	cm.Unlock()

	return e
}
