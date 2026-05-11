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
	bsfx "github.com/webitel/flow_manager/internal/bootstrap/fx"
	domaincontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	domainengine "github.com/webitel/flow_manager/internal/domain/engine"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"

	_ "net/http/pprof"
)

//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.engine.yaml
//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.cases.yaml
//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.cc.yaml
//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.chat.yaml
//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.general.yaml
//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.storage.yaml
//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.wbt.yaml
//go:generate go run github.com/bufbuild/buf/cmd/buf@latest generate --template buf/buf.gen.yaml
//go:generate go mod tidy

func main() {
	fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		fx.Provide(bsfx.NewConfig),
		fx.Provide(bsfx.NewAppID),
		fx.Provide(bsfx.NewLogger),
		fx.Provide(bsfx.NewPgxPool),
		fx.Provide(bsfx.NewSqlStore),
		fx.Provide(bsfx.NewStore),
		fx.Provide(bsfx.NewCheckpointRepo),
		fx.Provide(bsfx.NewCacheStores),
		fx.Provide(bsfx.NewEventBus),
		fx.Provide(bsfx.NewStorageClient),
		fx.Provide(bsfx.NewCasesClient),
		fx.Provide(bsfx.NewAiBotsClient),
		fx.Provide(bsfx.NewCCManager),
		fx.Provide(bsfx.NewCallbackResolver),
		fx.Provide(bsfx.NewTLSConfig),
		fx.Provide(bsfx.NewChatManager),
		fx.Provide(bsfx.NewServers),
		fx.Provide(app.NewFlowManager),
		fx.Provide(newRootContactsClient),
		fx.Provide(newRootEngineClient),
		fx.Provide(newRootMeetingClient),
		fx.Provide(flow.NewRouter),
		fx.Invoke(initRootRouters),
		fx.Invoke(bsfx.RegisterStartupHooks),
		fx.Invoke(registerRootLifecycle),
	).Run()
}

func initRootRouters(fm *app.FlowManager, router flow.Router, contacts domaincontacts.Client, meetings domainmeeting.Client) {
	call.Init(fm, router, contacts, meetings)
	grpc.Init(fm, router)
	chat.Init(fm, router)
	processing.Init(fm, router)
	email.Init(fm, router, contacts)
	channel.Init(fm, router)
	im.Init(fm, router, contacts)
}

func registerRootLifecycle(lc fx.Lifecycle, fm *app.FlowManager) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go fm.Listen()
			go setDebug()
			return nil
		},
		OnStop: func(_ context.Context) error {
			fm.Shutdown()
			return nil
		},
	})
}

func newRootContactsClient(lc fx.Lifecycle, fm *app.FlowManager) (domaincontacts.Client, error) {
	c := outboundcontacts.New(fm.Config().DiscoverySettings.Url)
	lc.Append(fx.Hook{OnStart: func(_ context.Context) error { return c.Start() }})
	return c, nil
}

func newRootEngineClient(lc fx.Lifecycle, fm *app.FlowManager) (domainengine.Client, error) {
	c := outboundengine.New(fm.Config().DiscoverySettings.Url)
	lc.Append(fx.Hook{OnStart: func(_ context.Context) error { return c.Start() }})
	return c, nil
}

func newRootMeetingClient(lc fx.Lifecycle, fm *app.FlowManager) (domainmeeting.Client, error) {
	c := outboundmeeting.New(fm.Config().DiscoverySettings.Url)
	lc.Append(fx.Hook{OnStart: func(_ context.Context) error { return c.Start() }})
	return c, nil
}

func setDebug() {
	wlog.Info("start debug server on http://localhost:8092/debug/pprof/")
	if err := http.ListenAndServe(":8092", nil); err != nil {
		wlog.Info(err.Error())
	}
}
