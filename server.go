// TODO WTEL-7091
//go:debug rsa1024min=0

package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/flow"
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
	interruptChan := make(chan os.Signal, 1)
	fm, err := app.NewFlowManager()
	if err != nil {
		panic(err.Error())
	}

	router := flow.NewRouter(fm)
	call.Init(fm, router)
	grpc.Init(fm, router)
	chat.Init(fm, router)
	processing.Init(fm, router)
	email.Init(fm, router)
	channel.Init(fm, router)
	webhook.Init(fm, router)
	im.Init(fm, router)

	go fm.Listen()
	setDebug()

	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interruptChan
	fm.Shutdown()
}

func setDebug() {
	// debug.SetGCPercent(-1)

	go func() {
		wlog.Info("start debug server on http://localhost:8092/debug/pprof/")
		err := http.ListenAndServe(":8092", nil)
		wlog.Info(err.Error())
	}()
}
