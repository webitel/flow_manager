package im

import (
	"context"
	"crypto/tls"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/webitel/engine/pkg/wbt"
	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	providers "github.com/webitel/flow_manager/gen/im/service/provider/v1"
	t "github.com/webitel/flow_manager/gen/im/service/thread/v1"
)

const (
	ServiceNameGateway = "im-gateway-service"
	ServiceNameThread  = "im-thread-service"
)

type Client struct {
	consulAddr      string
	startOnce       sync.Once
	messageService  *wbt.Client[p.MessageClient]
	threadService   *wbt.Client[p.ThreadManagementClient]
	accountService  *wbt.Client[p.AccountClient]
	th              *wbt.Client[t.ThreadManagementClient]
	facebookService *wbt.Client[providers.FacebookServiceClient]
	log             *wlog.Logger
	ctx             context.Context
	tls             *tls.Config
}

func NewClient(consulAddr string, log *wlog.Logger, t *tls.Config) *Client {
	cli := &Client{
		consulAddr: consulAddr,
		log:        log,
		tls:        t,
		ctx:        context.Background(),
	}

	return cli
}

func (cm *Client) Start() error {
	cm.log.Debug("starting " + ServiceNameGateway + " client")

	var err error

	cm.startOnce.Do(func() {
		var opts []wbt.Option

		if cm.tls != nil {
			cm.tls.InsecureSkipVerify = true
			opts = append(opts, wbt.WithGrpcOptions(
				grpc.WithTransportCredentials(credentials.NewTLS(cm.tls)),
			))
		}

		cm.messageService, err = wbt.NewClient(cm.consulAddr, ServiceNameGateway, p.NewMessageClient, opts...)
		if err != nil {
			return
		}
		cm.threadService, err = wbt.NewClient(cm.consulAddr, ServiceNameGateway, p.NewThreadManagementClient, opts...)
		if err != nil {
			return
		}

		if cm.accountService, err = wbt.NewClient(cm.consulAddr, ServiceNameGateway, p.NewAccountClient, opts...); err != nil {
			cm.log.Error("creating IM account service connection", wlog.Err(err))

			return
		}

		cm.th, err = wbt.NewClient(cm.consulAddr, ServiceNameThread, t.NewThreadManagementClient, opts...)
		if err != nil {
			return
		}

		if cm.facebookService, err = wbt.NewClient(cm.consulAddr, ServiceNameGateway, providers.NewFacebookServiceClient, opts...); err != nil {
			cm.log.Error("creating IM facebook service connection", wlog.Err(err))
			return
		}
	})

	return err
}

func (cm *Client) Stop() {
	cm.log.Debug("stopping " + ServiceNameGateway + " client")
	_ = cm.messageService.Close()
	_ = cm.threadService.Close()

	if err := cm.accountService.Close(); err != nil {
		cm.log.Error("closing account service connection gracefully", wlog.Err(err))
	}

	if err := cm.facebookService.Close(); err != nil {
		cm.log.Error("closing facebook service connection gracefully", wlog.Err(err))
	}
}
