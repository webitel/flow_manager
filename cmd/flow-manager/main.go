// TODO WTEL-7091
//go:debug rsa1024min=0

package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
	outboundcontacts "github.com/webitel/flow_manager/internal/adapters/outbound/contacts"
	domaincontacts "github.com/webitel/flow_manager/internal/domain/contacts"
	"github.com/webitel/flow_manager/routes/call"
	"github.com/webitel/flow_manager/routes/channel"
	"github.com/webitel/flow_manager/routes/chat"
	"github.com/webitel/flow_manager/routes/email"
	"github.com/webitel/flow_manager/routes/grpc"
	"github.com/webitel/flow_manager/routes/im"
	"github.com/webitel/flow_manager/routes/processing"
	"github.com/webitel/flow_manager/routes/webhook"
)

func main() {
	fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		fx.Provide(app.NewFlowManager),
		fx.Provide(newContactsClient),
		fx.Provide(flow.NewRouter),
		fx.Invoke(initRouters),
		fx.Invoke(registerLifecycle),
	).Run()
}

func initRouters(fm *app.FlowManager, router flow.Router) {
	call.Init(fm, router)
	grpc.Init(fm, router)
	chat.Init(fm, router)
	processing.Init(fm, router)
	email.Init(fm, router)
	channel.Init(fm, router)
	webhook.Init(fm, router)
	im.Init(fm, router)
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

func startDebugServer() {
	wlog.Info("start debug server on http://localhost:8092/debug/pprof/")
	if err := http.ListenAndServe(":8092", nil); err != nil {
		wlog.Info(err.Error())
	}
}
