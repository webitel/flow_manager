package bsfx

import (
	"context"

	"go.uber.org/fx"

	"github.com/webitel/wlog"

	inframq "github.com/webitel/flow_manager/infra/mq"
	aibridge "github.com/webitel/flow_manager/internal/adapters/outbound/aibridge"
	cases "github.com/webitel/flow_manager/internal/adapters/outbound/cases"
	outcc "github.com/webitel/flow_manager/internal/adapters/outbound/cc"
	outstorage "github.com/webitel/flow_manager/internal/adapters/outbound/storage"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	"github.com/webitel/flow_manager/model"
)

func NewEventBus(lc fx.Lifecycle, cfg *model.Config, id AppID, log *wlog.Logger) (ports.EventBus, error) {
	cli, err := inframq.NewRabbitEventBus(log, cfg.MQSettings.Url, string(id))
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return cli.Start()
		},
		OnStop: func(_ context.Context) error {
			cli.Close()
			return nil
		},
	})

	return cli, nil
}

func NewStorageClient(cfg *model.Config) (domstorage.Client, error) {
	return outstorage.NewStorageClient(cfg.DiscoverySettings.Url)
}

func NewCasesClient(cfg *model.Config) (*cases.Api, error) {
	return cases.NewClient(cfg.DiscoverySettings.Url)
}

// NewAiBotsClient creates the AI bots gRPC client and registers Start/Stop in
// the fx lifecycle.
func NewAiBotsClient(lc fx.Lifecycle, cfg *model.Config) (*aibridge.Client, error) {
	cli := aibridge.New(cfg.DiscoverySettings.Url)
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
func NewCCManager(lc fx.Lifecycle, cfg *model.Config, eventQueue ports.EventBus) (domcc.CCManager, error) {
	mgr := outcc.NewCCManager(cfg.DiscoverySettings.Url, eventQueue.ConsumeCCEvents())
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
