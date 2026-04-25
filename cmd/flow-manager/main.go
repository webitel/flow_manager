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

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	outboundcontacts "github.com/webitel/flow_manager/internal/adapters/outbound/contacts"
	outboundengine "github.com/webitel/flow_manager/internal/adapters/outbound/engine"
	outboundmeeting "github.com/webitel/flow_manager/internal/adapters/outbound/meeting"
	bsfx "github.com/webitel/flow_manager/internal/bootstrap/fx"
	domaincontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	domainengine "github.com/webitel/flow_manager/internal/domain/engine"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/domain/shared/ports"
	postgresStorage "github.com/webitel/flow_manager/internal/storage/postgres"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/routes/call"
	"github.com/webitel/flow_manager/routes/channel"
	"github.com/webitel/flow_manager/routes/chat"
	"github.com/webitel/flow_manager/routes/email"
	"github.com/webitel/flow_manager/routes/grpc"
	"github.com/webitel/flow_manager/routes/im"
	"github.com/webitel/flow_manager/routes/processing"
	"github.com/webitel/flow_manager/routes/webhook"

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
		fx.Provide(bsfx.NewCacheStores),
		fx.Invoke(runMigrations),
		// clients
		fx.Provide(bsfx.NewEventBus),
		fx.Provide(bsfx.NewStorageClient),
		fx.Provide(bsfx.NewCasesClient),
		fx.Provide(bsfx.NewAiBotsClient),
		fx.Provide(bsfx.NewCCManager),
		// servers
		fx.Provide(bsfx.NewCallbackResolver),
		fx.Provide(bsfx.NewTLSConfig),
		fx.Provide(bsfx.NewChatManager),
		fx.Provide(bsfx.NewServers),
		// app
		fx.Provide(app.NewFlowManager),
		fx.Provide(func(fm *app.FlowManager) ports.RouterDeps { return fm }),
		fx.Provide(newContactsClient),
		fx.Provide(newEngineClient),
		fx.Provide(newMeetingClient),
		fx.Provide(flow.NewRouter),
		fx.Provide(newAppRouters),
		fx.Invoke(wireRouters),
		fx.Invoke(bsfx.RegisterStartupHooks),
		fx.Invoke(registerLifecycle),
	).Run()
}

type appRouters struct {
	Call    model.Router
	GRPC    model.Router
	Chat    model.Router
	Form    model.Router
	Email   model.Router
	Channel model.Router
	WebHook model.Router
	IM      model.Router
}

func newAppRouters(
	deps ports.RouterDeps,
	router flow.Router,
	contacts domaincontacts.Client,
	meetings domainmeeting.Client,
) *appRouters {
	return &appRouters{
		Call:    call.Init(deps, router, contacts, meetings),
		GRPC:    grpc.Init(deps, router),
		Chat:    chat.Init(deps, router),
		Form:    processing.Init(deps, router),
		Email:   email.Init(deps, router, contacts),
		Channel: channel.Init(deps, router),
		WebHook: webhook.Init(deps, router),
		IM:      im.Init(deps, router),
	}
}

func wireRouters(fm *app.FlowManager, routers *appRouters) {
	fm.CallRouter = routers.Call
	fm.GRPCRouter = routers.GRPC
	fm.ChatRouter = routers.Chat
	fm.FormRouter = routers.Form
	fm.EmailRouter = routers.Email
	fm.ChannelRouter = routers.Channel
	fm.WebHookRouter = routers.WebHook
	fm.IMRouter = routers.IM
}

func registerLifecycle(lc fx.Lifecycle, fm *app.FlowManager) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go fm.Listen()
			go startDebugServer()
			return nil
		},
		OnStop: func(_ context.Context) error {
			fm.Shutdown()
			return nil
		},
	})
}

func newContactsClient(lc fx.Lifecycle, fm *app.FlowManager) (domaincontacts.Client, error) {
	c := outboundcontacts.New(fm.Config().DiscoverySettings.Url)
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return c.Start()
		},
	})
	return c, nil
}

func newEngineClient(lc fx.Lifecycle, fm *app.FlowManager) (domainengine.Client, error) {
	c := outboundengine.New(fm.Config().DiscoverySettings.Url)
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return c.Start()
		},
	})
	return c, nil
}

func newMeetingClient(lc fx.Lifecycle, fm *app.FlowManager) (domainmeeting.Client, error) {
	c := outboundmeeting.New(fm.Config().DiscoverySettings.Url)
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
