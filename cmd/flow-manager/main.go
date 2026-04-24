// TODO WTEL-7091
//go:debug rsa1024min=0

package main

import (
	"context"
	"net/http"

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
		fx.Provide(bsfx.NewSqlSupplier),
		fx.Provide(bsfx.NewStore),
		fx.Provide(bsfx.NewCheckpointRepo),
		fx.Provide(bsfx.NewCacheStores),
		// clients
		fx.Provide(bsfx.NewMQ),
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
		fx.Provide(newContactsClient),
		fx.Provide(newEngineClient),
		fx.Provide(newMeetingClient),
		fx.Provide(flow.NewRouter),
		fx.Invoke(initRouters),
		fx.Invoke(registerLifecycle),
	).Run()
}

func initRouters(fm *app.FlowManager, router flow.Router, contacts domaincontacts.Client, meetings domainmeeting.Client) {
	fm.CallRouter = call.Init(fm, router, contacts, meetings)
	fm.GRPCRouter = grpc.Init(fm, router)
	fm.ChatRouter = chat.Init(fm, router)
	fm.FormRouter = processing.Init(fm, router)
	fm.EmailRouter = email.Init(fm, router, contacts)
	fm.ChannelRouter = channel.Init(fm, router)
	fm.WebHookRouter = webhook.Init(fm, router)
	fm.IMRouter = im.Init(fm, router)
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

func startDebugServer() {
	wlog.Info("start debug server on http://localhost:8092/debug/pprof/")
	if err := http.ListenAndServe(":8092", nil); err != nil {
		wlog.Info(err.Error())
	}
}
