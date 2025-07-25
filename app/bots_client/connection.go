package bots_client

import (
	"context"
	"github.com/webitel/engine/pkg/wbt"
	"github.com/webitel/flow_manager/gen/ai_bots"
	"github.com/webitel/wlog"
	"sync"
)

type Client struct {
	consulAddr string
	startOnce  sync.Once
	converse   *wbt.Client[ai_bots.ConverseServiceClient]
	bot        *wbt.Client[ai_bots.BotsServiceClient]
	embed      *wbt.Client[ai_bots.EmbeddingServiceClient]
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
		cm.bot, err = wbt.NewClient(cm.consulAddr, "ai-bots", ai_bots.NewBotsServiceClient)
		if err != nil {
			return
		}
		cm.embed, err = wbt.NewClient(cm.consulAddr, "ai-bots", ai_bots.NewEmbeddingServiceClient)
		if err != nil {
			return
		}
		cm.converse, err = wbt.NewClient(cm.consulAddr, "ai-bots", ai_bots.NewConverseServiceClient)
		if err != nil {
			return
		}
	})
	return err
}

func (cm *Client) Bot() ai_bots.BotsServiceClient {
	return cm.bot.Api
}

func (cm *Client) Embed() ai_bots.EmbeddingServiceClient {
	return cm.embed.Api
}

func (cm *Client) Converse() ai_bots.ConverseServiceClient {
	return cm.converse.Api
}

func (cm *Client) WithConnection(ctx context.Context, connection string) context.Context {
	return cm.converse.StaticHost(ctx, connection)
}

func (cm *Client) Stop() {

}
