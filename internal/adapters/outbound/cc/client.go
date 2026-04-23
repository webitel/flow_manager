package cc

import (
	"sync"

	"github.com/webitel/wlog"

	genpb "github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/infra/grpcdial"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/model"
)

const ServiceName = "call_center"

type ccManager struct {
	startOnce  sync.Once
	consulAddr string

	agentClient  *grpcdial.Client[genpb.AgentServiceClient]
	memberClient *grpcdial.Client[genpb.MemberServiceClient]

	agent    domcc.AgentApi
	member   domcc.MemberApi
	events   <-chan model.CCQueueEvent
	attempts map[int64]chan domcc.QueueEvent
	closed   chan struct{}
	sync.RWMutex
}

func NewCCManager(consulAddr string, events <-chan model.CCQueueEvent) domcc.CCManager {
	cli := &ccManager{
		consulAddr: consulAddr,
		events:     events,
		closed:     make(chan struct{}),
		attempts:   make(map[int64]chan domcc.QueueEvent),
	}

	return cli
}

func (cm *ccManager) Agent() domcc.AgentApi {
	return cm.agent
}

func (cm *ccManager) Member() domcc.MemberApi {
	return cm.member
}

func (cm *ccManager) Start() error {
	wlog.Debug("starting cc service")
	var err error

	cm.startOnce.Do(func() {
		cm.agentClient, err = grpcdial.NewClient(cm.consulAddr, ServiceName, genpb.NewAgentServiceClient)
		if err != nil {
			return
		}

		cm.memberClient, err = grpcdial.NewClient(cm.consulAddr, ServiceName, genpb.NewMemberServiceClient)
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
			cm.RLock()
			a, ok := cm.attempts[event.AttemptId]
			cm.RUnlock()
			if !ok {
				continue
			}
			a <- domcc.QueueEvent{
				AttemptId: event.AttemptId,
				Event:     event.Event,
				Result:    event.Result,
			}
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

func (cm *ccManager) SubscribeAttempt(attemptId int64) <-chan domcc.QueueEvent {
	cm.Lock()
	e, ok := cm.attempts[attemptId]
	cm.Unlock()
	if ok {
		return e
	}

	e = make(chan domcc.QueueEvent)
	cm.Lock()
	cm.attempts[attemptId] = e
	cm.Unlock()

	return e
}
