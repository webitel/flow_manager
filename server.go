package main

import (
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/call"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/grpc_route"
	"github.com/webitel/wlog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	interruptChan := make(chan os.Signal, 1)

	fm, err := app.NewFlowManager()

	if err != nil {
		panic(err.Error())
	}
	wlog.Info(fmt.Sprintf("server build version: %s", fm.Version()))

	flow.Init(fm)
	call.Init(fm)
	grpc_route.Init(fm)

	go fm.Listen()

	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interruptChan
	fm.Shutdown()
}
