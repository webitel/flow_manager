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
)

const ServiceName = "im-gateway-service"

type Client struct {
	consulAddr     string
	startOnce      sync.Once
	messageService *wbt.Client[p.MessageClient]
	threadService  *wbt.Client[p.ThreadManagementClient]
	accountService *wbt.Client[p.AccountClient]

	log *wlog.Logger
	ctx context.Context
	tls *tls.Config
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
	cm.log.Debug("starting " + ServiceName + " client")

	var err error
	cm.startOnce.Do(func() {
		var opts []wbt.Option
		if cm.tls != nil {
			opts = append(opts, wbt.WithGrpcOptions(
				grpc.WithTransportCredentials(credentials.NewTLS(cm.tls)),
			))
		}

		if cm.messageService, err = wbt.NewClient(cm.consulAddr, ServiceName, p.NewMessageClient, opts...); err != nil {
			cm.log.Error("creating IM message service connection", wlog.Err(err))
			return
		}

		if cm.threadService, err = wbt.NewClient(cm.consulAddr, ServiceName, p.NewThreadManagementClient, opts...); err != nil {
			cm.log.Error("creating IM thread service connection", wlog.Err(err))
			return
		}

		if cm.accountService, err = wbt.NewClient(cm.consulAddr, ServiceName, p.NewAccountClient, opts...); err != nil {
			cm.log.Error("creating IM account service connection", wlog.Err(err))
			return
		}
	})

	return err
}

func (cm *Client) Stop() {
	cm.log.Debug("stopping " + ServiceName + " client")
	_ = cm.messageService.Close()
	_ = cm.threadService.Close()

	if err := cm.accountService.Close(); err != nil {
		cm.log.Error("closing account service connection gracefully", wlog.Err(err))
	}
}
