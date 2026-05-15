package bsfx

import (
	"context"

	"go.uber.org/fx"

	"github.com/webitel/wlog"

	aibridge "github.com/webitel/flow_manager/internal/adapters/outbound/aibridge"
	cases "github.com/webitel/flow_manager/internal/adapters/outbound/cases"
	outcc "github.com/webitel/flow_manager/internal/adapters/outbound/cc"
	eventAdapter "github.com/webitel/flow_manager/internal/adapters/outbound/event"
	outstorage "github.com/webitel/flow_manager/internal/adapters/outbound/storage"
	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	domstorage "github.com/webitel/flow_manager/internal/domain/storage"
	inframq "github.com/webitel/flow_manager/internal/infrastructure/mq"
	"github.com/webitel/flow_manager/internal/storage"
	callWatcherPkg "github.com/webitel/flow_manager/internal/workers/call_watcher"
	listWatcherPkg "github.com/webitel/flow_manager/internal/workers/list_watcher"
)

func NewEventBus(lc fx.Lifecycle, cfg *bscfg.Config, id AppID, log *wlog.Logger) (ports.EventBus, error) {
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

func NewStorageClient(cfg *bscfg.Config) (domstorage.Client, error) {
	return outstorage.NewStorageClient(cfg.DiscoverySettings.Url)
}

func NewCasesClient(cfg *bscfg.Config) (*cases.Api, error) {
	return cases.NewClient(cfg.DiscoverySettings.Url)
}

// NewAiBotsClient creates the AI bots gRPC client and registers Start/Stop in
// the fx lifecycle.
func NewAiBotsClient(lc fx.Lifecycle, cfg *bscfg.Config) (*aibridge.Client, error) {
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
func NewCCManager(lc fx.Lifecycle, cfg *bscfg.Config, eventQueue ports.EventBus) (domcc.CCManager, error) {
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

// NewEventBusAdapter wraps the event bus into the adapter that satisfies
// notification and call-event dep interfaces used by routers and workers.
func NewEventBusAdapter(eventQueue ports.EventBus, st storage.Store, cfg *bscfg.Config) *eventAdapter.EventBusAdapter {
	return eventAdapter.NewEventBusAdapter(eventQueue, st, cfg)
}

// NewCallWatcher creates the call-event worker. Lifecycle (Start/Stop) is
// managed by registerLifecycle in main so it can use the Dispatcher's stop channel.
func NewCallWatcher(st storage.Store, deps *eventAdapter.EventBusAdapter, log *wlog.Logger) *callWatcherPkg.Worker {
	return callWatcherPkg.New(st, deps, log)
}

// NewListWatcher creates the list-cleanup worker.
func NewListWatcher(st storage.Store, log *wlog.Logger) *listWatcherPkg.Worker {
	return listWatcherPkg.New(st, log)
}
