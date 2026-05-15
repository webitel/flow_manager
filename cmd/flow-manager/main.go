// TODO WTEL-7091
//go:debug rsa1024min=0

package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/webitel/wlog"

	bsruntime "github.com/webitel/flow_manager/internal/bootstrap/runtime"
	"github.com/webitel/flow_manager/internal/adapters/inbound/call"
	"github.com/webitel/flow_manager/internal/adapters/inbound/channel"
	"github.com/webitel/flow_manager/internal/adapters/inbound/chat"
	"github.com/webitel/flow_manager/internal/adapters/inbound/email"
	"github.com/webitel/flow_manager/internal/adapters/inbound/grpc"
	"github.com/webitel/flow_manager/internal/adapters/inbound/im"
	"github.com/webitel/flow_manager/internal/adapters/inbound/processing"
	outboundcontacts "github.com/webitel/flow_manager/internal/adapters/outbound/contacts"
	outboundengine "github.com/webitel/flow_manager/internal/adapters/outbound/engine"
	outboundmeeting "github.com/webitel/flow_manager/internal/adapters/outbound/meeting"
	bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"
	bsfx "github.com/webitel/flow_manager/internal/bootstrap/fx"
	bootstrapServers "github.com/webitel/flow_manager/internal/bootstrap/servers"
	domaincontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	domainengine "github.com/webitel/flow_manager/internal/domain/engine"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/domain/flow"
	postgresStorage "github.com/webitel/flow_manager/internal/storage/postgres"
	callWatcherPkg "github.com/webitel/flow_manager/internal/workers/call_watcher"
	listWatcherPkg "github.com/webitel/flow_manager/internal/workers/list_watcher"

	_ "net/http/pprof"
)

func main() {
	fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxErrLogger{} }),
		// infrastructure
		fx.Provide(bsfx.NewConfig),
		fx.Provide(bsfx.NewAppID),
		fx.Provide(bsfx.NewLogger),
		fx.Provide(bsfx.NewStore),
		fx.Provide(bsfx.NewPgxPool),
		fx.Provide(bsfx.NewSqlStore),
		fx.Provide(bsfx.NewCheckpointRepo),
		fx.Provide(bsfx.NewRuntimeStateRepo),
		fx.Provide(bsfx.NewCacheStores),
		fx.Invoke(runMigrations),
		// clients
		fx.Provide(bsfx.NewEventBus),
		fx.Provide(bsfx.NewEventBusAdapter),
		fx.Provide(bsfx.NewStorageClient),
		fx.Provide(bsfx.NewCasesClient),
		fx.Provide(bsfx.NewAiBotsClient),
		fx.Provide(bsfx.NewCCManager),
		// servers
		fx.Provide(bsfx.NewCallbackResolver),
		fx.Provide(bsfx.NewTLSConfig),
		fx.Provide(bsfx.NewChatManager),
		fx.Provide(bsfx.NewServers),
		// workers
		fx.Provide(bsfx.NewCallWatcher),
		fx.Provide(bsfx.NewListWatcher),
		// app
		fx.Provide(newContactsClient),
		fx.Provide(newEngineClient),
		fx.Provide(newMeetingClient),
		fx.Provide(bsruntime.NewRouterDeps),
		fx.Provide(newAppRouters),
		fx.Provide(newDispatcher),
		fx.Invoke(bsfx.RegisterInfraHooks),
		fx.Invoke(registerLifecycle),
	).Run()
}

type appRouters struct {
	Call    flow.Router
	GRPC    flow.Router
	Chat    flow.Router
	Form    flow.Router
	Email   flow.Router
	Channel flow.Router
	IM      flow.Router
}

func newAppRouters(
	deps *bsruntime.RouterDeps,
	contacts domaincontacts.Client,
	meetings domainmeeting.Client,
) *appRouters {
	return &appRouters{
		Call:    call.Init(deps, contacts, meetings),
		GRPC:    grpc.Init(deps),
		Chat:    chat.Init(deps),
		Form:    processing.Init(deps, deps.Cases()),
		Email:   email.Init(deps, contacts),
		Channel: channel.Init(deps),
		IM:      im.Init(deps, contacts),
	}
}

func newDispatcher(
	deps *bsruntime.RouterDeps,
	routers *appRouters,
	srvs bootstrapServers.Servers,
) *bsruntime.Dispatcher {
	return bsruntime.New(bsruntime.DispatcherConfig{
		Log:            deps.Log(),
		ID:             deps.AppID(),
		GrpcServer:     srvs.Grpc,
		CallGrpcServer: srvs.CallGrpc,
		EslServer:      srvs.Esl,
		MailServer:     srvs.Mail,
		ChannelServer:  srvs.Channel,
		ImServer:       srvs.Im,
		Routers: bsruntime.RouterSet{
			Call:    routers.Call,
			GRPC:    routers.GRPC,
			Chat:    routers.Chat,
			Form:    routers.Form,
			Email:   routers.Email,
			Channel: routers.Channel,
			IM:      routers.IM,
		},
		CheckpointRepo:   deps.CheckpointRepo(),
		RuntimeStateRepo: deps.RuntimeStateRepo(),
	})
}

func registerLifecycle(
	lc fx.Lifecycle,
	d *bsruntime.Dispatcher,
	callWatcher *callWatcherPkg.Worker,
	listWatcher *listWatcherPkg.Worker,
) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			callWatcher.Start(d.Stop())
			listWatcher.Start()
			go d.Listen()
			go startDebugServer()
			return nil
		},
		OnStop: func(_ context.Context) error {
			callWatcher.Stop()
			listWatcher.Stop()
			d.Shutdown()
			return nil
		},
	})
}

func newContactsClient(lc fx.Lifecycle, cfg *bscfg.Config) (domaincontacts.Client, error) {
	c := outboundcontacts.New(cfg.DiscoverySettings.Url)
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return c.Start()
		},
	})
	return c, nil
}

func newEngineClient(lc fx.Lifecycle, cfg *bscfg.Config) (domainengine.Client, error) {
	c := outboundengine.New(cfg.DiscoverySettings.Url)
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return c.Start()
		},
	})
	return c, nil
}

func newMeetingClient(lc fx.Lifecycle, cfg *bscfg.Config) (domainmeeting.Client, error) {
	c := outboundmeeting.New(cfg.DiscoverySettings.Url)
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return c.Start()
		},
	})
	return c, nil
}

// fxErrLogger is an fxevent.Logger that prints only error events to stderr.
type fxErrLogger struct{}

func (fxErrLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.Started:
		if e.Err != nil {
			wlog.Error("fx start: " + e.Err.Error())
		}
	case *fxevent.Stopped:
		if e.Err != nil {
			wlog.Error("fx stop: " + e.Err.Error())
		}
	case *fxevent.RolledBack:
		if e.Err != nil {
			wlog.Error("fx rollback: " + e.Err.Error())
		}
	case *fxevent.Provided:
		if e.Err != nil {
			wlog.Error("fx provide " + e.ConstructorName + ": " + e.Err.Error())
		}
	case *fxevent.Decorated:
		if e.Err != nil {
			wlog.Error("fx decorate " + e.DecoratorName + ": " + e.Err.Error())
		}
	case *fxevent.Run:
		if e.Err != nil {
			wlog.Error("fx run " + e.Name + ": " + e.Err.Error())
		}
	}
}

func runMigrations(pool *pgxpool.Pool) error {
	return postgresStorage.RunMigrations(context.Background(), pool)
}

func startDebugServer() {
	wlog.Info("start debug server on http://localhost:8092/debug/pprof/")
	if err := http.ListenAndServe(":8092", nil); err != nil {
		wlog.Info(err.Error())
	}
}
