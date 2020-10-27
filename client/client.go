package client

import (
	"fmt"
	"github.com/webitel/engine/discovery"
	"github.com/webitel/flow_manager/model"
	// "github.com/webitel/flow_manager/providers/grpc/workflow"
	"github.com/webitel/protos/workflow"
	"github.com/webitel/wlog"
	"sync"
)

const (
	WatcherInterval = 5 * 1000
)

type DoDistributeRequest struct {
}

type FlowManager interface {
	Start() error
	Stop()

	Queue() QueueApi
}

type QueueApi interface {
	DoDistributeAttempt(in *workflow.DistributeAttemptRequest) (*workflow.DistributeAttemptResponse, error)
}

type flowManager struct {
	serviceDiscovery discovery.ServiceDiscovery
	poolConnections  discovery.Pool

	watcher   *discovery.Watcher
	startOnce sync.Once
	stop      chan struct{}
	stopped   chan struct{}

	queue QueueApi
}

func NewFlowManager(serviceDiscovery discovery.ServiceDiscovery) FlowManager {
	fm := &flowManager{
		stop:             make(chan struct{}),
		stopped:          make(chan struct{}),
		poolConnections:  discovery.NewPoolConnections(),
		serviceDiscovery: serviceDiscovery,
	}

	fm.queue = NewQueueApi(fm)

	return fm
}

func (am *flowManager) Start() error {
	wlog.Debug("starting flow manager service")

	if services, err := am.serviceDiscovery.GetByName(model.AppServiceName); err != nil {
		return err
	} else {
		for _, v := range services {
			am.registerConnection(v)
		}
	}

	am.startOnce.Do(func() {
		am.watcher = discovery.MakeWatcher("flow manager", WatcherInterval, am.wakeUp)
		go am.watcher.Start()
		go func() {
			defer func() {
				wlog.Debug("stopped flow manager manager")
				close(am.stopped)
			}()

			for {
				select {
				case <-am.stop:
					wlog.Debug("flow manager received stop signal")
					return
				}
			}
		}()
	})
	return nil
}

func (am *flowManager) Stop() {
	if am.watcher != nil {
		am.watcher.Stop()
	}

	if am.poolConnections != nil {
		am.poolConnections.CloseAllConnections()
	}

	close(am.stop)
	<-am.stopped
}

func (am *flowManager) registerConnection(v *discovery.ServiceConnection) {
	addr := fmt.Sprintf("%s:%d", v.Host, v.Port)
	client, err := NewFlowConnection(v.Id, addr)
	if err != nil {
		wlog.Error(fmt.Sprintf("connection %s [%s] error: %s", v.Id, addr, err.Error()))
		return
	}
	am.poolConnections.Append(client)
	wlog.Debug(fmt.Sprintf("register connection %s [%s]", client.Name(), addr))
}

func (am *flowManager) wakeUp() {
	list, err := am.serviceDiscovery.GetByName(model.AppServiceName)
	if err != nil {
		wlog.Error(err.Error())
		return
	}

	for _, v := range list {
		if _, err := am.poolConnections.GetById(v.Id); err == discovery.ErrNotFoundConnection {
			am.registerConnection(v)
		}
	}
	am.poolConnections.RecheckConnections()
}

func (cc *flowManager) getRandomClient() (*fConnection, error) {
	cli, err := cc.poolConnections.Get(discovery.StrategyRoundRobin)
	if err != nil {
		return nil, err
	}

	return cli.(*fConnection), nil
}

func (am *flowManager) Queue() QueueApi {
	return am.queue
}
