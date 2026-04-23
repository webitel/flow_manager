package aibridge

import (
	"context"
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/gen/ai_bots"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const (
	serviceName = "ai-bots"
)

type Client struct {
	consulAddr string
	startOnce  sync.Once
	converse   *grpcdial.Client[ai_bots.ConverseServiceClient]
	bot        *grpcdial.Client[ai_bots.BotsServiceClient]
	embed      *grpcdial.Client[ai_bots.EmbeddingServiceClient]
}

func New(consulAddr string) *Client {
	cli := &Client{
		consulAddr: consulAddr,
	}

	return cli
}

func (cm *Client) Start() error {
	wlog.Debug("starting ai_bots client")
	var err error
	cm.startOnce.Do(func() {
		cm.bot, err = grpcdial.NewClient(cm.consulAddr, serviceName, ai_bots.NewBotsServiceClient)
		if err != nil {
			return
		}
		cm.embed, err = grpcdial.NewClient(cm.consulAddr, serviceName, ai_bots.NewEmbeddingServiceClient)
		if err != nil {
			return
		}
		cm.converse, err = grpcdial.NewClient(cm.consulAddr, serviceName, ai_bots.NewConverseServiceClient)
		if err != nil {
			return
		}
	})
	return err
}

func (cm *Client) Bot() ai_bots.BotsServiceClient {
	return cm.bot.API
}

func (cm *Client) Embed() ai_bots.EmbeddingServiceClient {
	return cm.embed.API
}

func (cm *Client) Converse() ai_bots.ConverseServiceClient {
	return cm.converse.API
}

func (cm *Client) WithConnection(ctx context.Context, connection string) context.Context {
	return cm.converse.StaticHost(ctx, connection)
}

func (cm *Client) Stop() {
}
