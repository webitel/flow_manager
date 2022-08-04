package grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/webitel/flow_manager/model"
	client "github.com/webitel/protos/engine/chat"

	"github.com/webitel/engine/discovery"
	"github.com/webitel/wlog"
)

type ChatManager struct {
	serviceDiscovery discovery.ServiceDiscovery
	poolConnections  discovery.Pool

	watcher   *discovery.Watcher
	startOnce sync.Once
	stop      chan struct{}
	stopped   chan struct{}
}

func NewChatManager() *ChatManager {
	return &ChatManager{
		stop:            make(chan struct{}),
		stopped:         make(chan struct{}),
		poolConnections: discovery.NewPoolConnections(),
	}
}

func (cm *ChatManager) Start(sd discovery.ServiceDiscovery) error {
	wlog.Debug("starting chat client")
	cm.serviceDiscovery = sd

	if services, err := cm.serviceDiscovery.GetByName(ChatClientService); err != nil {
		return err
	} else {
		for _, v := range services {
			cm.registerConnection(v)
		}
	}

	cm.startOnce.Do(func() {
		cm.watcher = discovery.MakeWatcher("chat client", WatcherInterval, cm.wakeUp)
		go cm.watcher.Start()
		go func() {
			defer func() {
				wlog.Debug("stopper chat client")
				close(cm.stopped)
			}()

			for {
				select {
				case <-cm.stop:
					wlog.Debug("chat client received stop signal")
					return
				}
			}
		}()
	})
	return nil
}

func (cm *ChatManager) Stop() {
	if cm.watcher != nil {
		cm.watcher.Stop()
	}

	if cm.poolConnections != nil {
		cm.poolConnections.CloseAllConnections()
	}

	close(cm.stop)
	<-cm.stopped
}

func (cm *ChatManager) registerConnection(v *discovery.ServiceConnection) {
	addr := fmt.Sprintf("%s:%d", v.Host, v.Port)
	c, err := NewChatClientConnection(v.Id, addr)
	if err != nil {
		wlog.Error(fmt.Sprintf("connection %s [%s] error: %s", v.Id, addr, err.Error()))
		return
	}
	cm.poolConnections.Append(c)
	wlog.Debug(fmt.Sprintf("register connection %s [%s]", c.Name(), addr))
}

func (cm *ChatManager) getClient(name string) (*ChatClientConnection, error) {
	conn, err := cm.poolConnections.GetById(name)
	if err != nil {
		return nil, err
	}
	return conn.(*ChatClientConnection), nil
}

func (cm *ChatManager) getRandCli() (*ChatClientConnection, error) {
	conn, err := cm.poolConnections.Get(discovery.StrategyRoundRobin)
	if err != nil {
		return nil, err
	}
	return conn.(*ChatClientConnection), nil
}

func (cm *ChatManager) wakeUp() {
	list, err := cm.serviceDiscovery.GetByName(ChatClientService)
	if err != nil {
		wlog.Error(err.Error())
		return
	}

	for _, v := range list {
		if _, err := cm.poolConnections.GetById(v.Id); err == discovery.ErrNotFoundConnection {
			cm.registerConnection(v)
		}
	}
	cm.poolConnections.RecheckConnections(list.Ids())
}

func (cc *ChatManager) BroadcastMessage(ctx context.Context, domainId int64, req model.BroadcastChat) error {
	c, e := cc.getRandCli()
	if e != nil {
		return e
	}

	msg := &client.Message{
		Type:      req.Type,
		Text:      req.Text,
		File:      getFile(req.File),
		Variables: req.Variables,
	}

	if req.Menu != nil {
		msg.Buttons = getChatButtons(req.Menu.Buttons)
		msg.Inline = getChatButtons(req.Menu.Inline)
	}

	res, e := c.messages.BroadcastMessage(ctx, &client.BroadcastMessageRequest{
		Message: msg,
		From:    req.Profile.Id,
		Peer:    req.Peer,
	})

	if e != nil {
		return e
	}

	if len(res.Failure) > 0 {
		return errors.New(res.Failure[0].String())
	}

	return nil
}
