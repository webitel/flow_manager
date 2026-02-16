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
	consulAddr string
	startOnce  sync.Once
	*wbt.Client[p.MessageClient]
	log *wlog.Logger
	ctx context.Context
	tls *tls.Config
}

func NewClient(consulAddr string, log *wlog.Logger, t *tls.Config) *Client {
	cli := &Client{
		consulAddr: consulAddr,
		log:        log,
		tls:        t,
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

		cm.Client, err = wbt.NewClient(cm.consulAddr, ServiceName, p.NewMessageClient, opts...)
		if err != nil {
			return
		}
	})
	return err
}

func (cm *Client) Stop() {
	cm.log.Debug("stopping " + ServiceName + " client")
	_ = cm.Client.Close()
}
