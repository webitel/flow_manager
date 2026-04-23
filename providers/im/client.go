package im

import (
	"crypto/tls"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const ServiceName = "im-gateway-service"

type Client struct {
	consulAddr string
	startOnce  sync.Once
	*grpcdial.Client[p.MessageClient]
	log *wlog.Logger
	tls *tls.Config
}

func NewClient(consulAddr string, log *wlog.Logger, t *tls.Config) *Client {
	return &Client{
		consulAddr: consulAddr,
		log:        log,
		tls:        t,
	}
}

func (cm *Client) Start() error {
	cm.log.Debug("starting " + ServiceName + " client")

	var err error
	cm.startOnce.Do(func() {
		var opts []grpc.DialOption
		if cm.tls != nil {
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(cm.tls)))
		} else {
			opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		cm.Client, err = grpcdial.NewClientWithOpts(cm.consulAddr, ServiceName, p.NewMessageClient, opts...)
	})
	return err
}

func (cm *Client) Stop() {
	cm.log.Debug("stopping " + ServiceName + " client")
	_ = cm.Client.Close()
}
