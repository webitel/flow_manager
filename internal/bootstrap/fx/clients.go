package bsfx

import (
	"context"

	"go.uber.org/fx"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/app/bots_client"
	"github.com/webitel/flow_manager/app/cc"
	cases "github.com/webitel/flow_manager/internal/adapters/outbound/cases"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/mq"
	"github.com/webitel/flow_manager/mq/rabbit"
	fmgrpc "github.com/webitel/flow_manager/providers/grpc"
)

func NewMQ(cfg *model.Config, id AppID) mq.MQ {
	return mq.NewMQ(rabbit.NewRabbitMQ(cfg.MQSettings, string(id)))
}

func NewStorageClient(cfg *model.Config) (*app.StorageClient, error) {
	return app.NewStorageClient(cfg.DiscoverySettings.Url)
}

func NewCasesClient(cfg *model.Config) (*cases.Api, error) {
	return cases.NewClient(cfg.DiscoverySettings.Url)
}

// NewAiBotsClient creates the AI bots gRPC client and registers Start/Stop in
// the fx lifecycle.
func NewAiBotsClient(lc fx.Lifecycle, cfg *model.Config) (*bots_client.Client, error) {
	cli := bots_client.New(cfg.DiscoverySettings.Url)
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return cli.Start()
		},
		OnStop: func(_ context.Context) error {
			cli.Stop()
			return nil
		},
	})
	return cli, nil
}

// NewCCManager creates the CC queue manager and registers Start/Stop in the fx
// lifecycle.
func NewCCManager(lc fx.Lifecycle, cfg *model.Config, eventQueue mq.MQ) (cc.CCManager, error) {
	mgr := cc.NewCCManager(cfg.DiscoverySettings.Url, eventQueue.ConsumeCCEvents())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return mgr.Start()
		},
		OnStop: func(_ context.Context) error {
			mgr.Stop()
			return nil
		},
	})
	return mgr, nil
}

// NewChatManager constructs the chat gRPC manager. Start requires
// cluster discovery and is called by FlowManager until the cluster is
// extracted as an fx provider (planned for Phase 2).
func NewChatManager() *fmgrpc.ChatManager {
	return fmgrpc.NewChatManager()
}
