package main

import (
	"fmt"
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/call"
	"github.com/webitel/flow_manager/email"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/grpc_route"
	"github.com/webitel/wlog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println(app.Version())
	interruptChan := make(chan os.Signal, 1)
	fm, err := app.NewFlowManager()

	if err != nil {
		panic(err.Error())
	}

	router := flow.NewRouter(fm)
	call.Init(fm, router)
	email.Init(fm, router)
	grpc_route.Init(fm, router)

	go fm.Listen()
	setDebug()

	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interruptChan
	fm.Shutdown()
}

func setDebug() {
	//debug.SetGCPercent(-1)

	go func() {
		wlog.Info("start debug server on http://localhost:8092/debug/pprof/")
		err := http.ListenAndServe(":8092", nil)
		wlog.Info(err.Error())
	}()
}
