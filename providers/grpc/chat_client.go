package grpc

import (
	"context"
	"fmt"
	client "github.com/webitel/engine/chat_manager/chat"
	"github.com/webitel/engine/discovery"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"sync"
	"time"
)

var (
	ChatClientService = "webitel.chat.server"
	WatcherInterval   = 5 * 1000
)

type chatManager struct {
	serviceDiscovery discovery.ServiceDiscovery
	poolConnections  discovery.Pool

	watcher   *discovery.Watcher
	startOnce sync.Once
	stop      chan struct{}
	stopped   chan struct{}
}

type ChatClientConnection struct {
	id     string
	host   string
	client *grpc.ClientConn
	api    client.ChatServiceClient
}

func NewChatManager(serviceDiscovery discovery.ServiceDiscovery) *chatManager {
	return &chatManager{
		stop:             make(chan struct{}),
		stopped:          make(chan struct{}),
		poolConnections:  discovery.NewPoolConnections(),
		serviceDiscovery: serviceDiscovery,
	}
}

type serviceInterceptor struct {
	name string
}

func (cm *ChatClientConnection) UnaryClientInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Create a new context with the token and make the first request
	serviceCtx := metadata.AppendToOutgoingContext(ctx, model.HeaderFromServiceName, model.AppServiceName)
	return invoker(serviceCtx, method, req, reply, cc, opts...)
}

func NewChatClientConnection(id, url string) (*ChatClientConnection, error) {
	var err error
	connection := &ChatClientConnection{
		id:   id,
		host: url,
	}

	connection.client, err = grpc.Dial(
		url,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(time.Second*5),
		grpc.WithUnaryInterceptor(connection.UnaryClientInterceptor),
	)

	if err != nil {
		return nil, err
	}

	connection.api = client.NewChatServiceClient(connection.client)

	return connection, nil
}

func (cm *chatManager) Start() error {
	wlog.Debug("starting chat client")

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

func (cm *chatManager) Stop() {
	if cm.watcher != nil {
		cm.watcher.Stop()
	}

	if cm.poolConnections != nil {
		cm.poolConnections.CloseAllConnections()
	}

	close(cm.stop)
	<-cm.stopped
}

func (cm *chatManager) registerConnection(v *discovery.ServiceConnection) {
	addr := fmt.Sprintf("%s:%d", v.Host, v.Port)
	client, err := NewChatClientConnection(v.Id, addr)
	if err != nil {
		wlog.Error(fmt.Sprintf("connection %s [%s] error: %s", v.Id, addr, err.Error()))
		return
	}
	cm.poolConnections.Append(client)
	wlog.Debug(fmt.Sprintf("register connection %s [%s]", client.Name(), addr))
}

func (cm *chatManager) getClient(name string) (*ChatClientConnection, error) {
	conn, err := cm.poolConnections.GetById(name)
	if err != nil {
		return nil, err
	}
	return conn.(*ChatClientConnection), nil
}

func (cm *chatManager) wakeUp() {
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
	cm.poolConnections.RecheckConnections()
}

func (cc *ChatClientConnection) Name() string {
	return cc.id
}

func (cc *ChatClientConnection) Ready() bool {
	switch cc.client.GetState() {
	case connectivity.Idle, connectivity.Ready:
		return true
	}
	return false
}

func (cc *ChatClientConnection) Close() error {
	return cc.client.Close()
}
