package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/workers/runtime_recovery"
	"github.com/webitel/flow_manager/internal/workers/session_recovery"
	"github.com/webitel/flow_manager/model"
)

func (f *FlowManager) Listen() {
	f.log.Info("listening connections...")
	defer f.log.Info("stopped listen new connection")
	defer close(f.stopped)
	var wg sync.WaitGroup

	f.callWatcher.Start()
	f.listWatcher.Start()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	go func() {
		<-f.stop
		workerCancel()
	}()

	if f.checkpointRepo != nil {
		go session_recovery.New(f.checkpointRepo, f.id, f.log).Run(workerCtx)
	}

	if f.runtimeStateRepo != nil {
		go runtime_recovery.New(f.runtimeStateRepo, f.id, f.log).Run(workerCtx)
	}

	type backgroundRunner interface {
		StartBackground(ctx context.Context)
	}
	if br, ok := f.IMRouter.(backgroundRunner); ok {
		br.StartBackground(workerCtx)
	}

	go f.listenCallEvents(f.stop)

	if f.eslServer != nil {
		wg.Add(1)
		go f.listenCallConnection(f.stop, &wg, f.eslServer)
	}

	if f.grpcServer != nil {
		wg.Add(1)
		go f.listenGrpcConnection(f.stop, &wg, f.grpcServer)
	}

	if f.mailServer != nil {
		wg.Add(1)
		go f.listenInboundEmail(f.stop, &wg, f.mailServer)
	}

	if f.channelServer != nil {
		wg.Add(1)
		go f.listenChannelConnection(f.stop, &wg, f.channelServer)
	}

	if f.imServer != nil {
		wg.Add(1)
		go f.listenImConnection(f.stop, &wg, f.imServer)
	}

	wg.Wait()
}

func (f *FlowManager) listenImConnection(stop chan struct{}, wg *sync.WaitGroup, srv model.Server) {
	defer wg.Done()
	f.log.Info("listen im connections...")
	for {
		select {
		case <-stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}
			if err := f.IMRouter.Handle(c); err != nil {
				c.Log().Error(err.Error())
			}
		}
	}
}

func (f *FlowManager) listenCallConnection(stop chan struct{}, wg *sync.WaitGroup, srv model.Server) {
	defer wg.Done()
	f.log.Info("listen call connections...")
	for {
		select {
		case <-stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}

			if err := f.CallRouter.Handle(c); err != nil {
				c.Log().Error(err.Error())
			}
		}
	}
}

func (f *FlowManager) listenGrpcConnection(stop chan struct{}, wg *sync.WaitGroup, srv model.Server) {
	defer wg.Done()
	wlog.Info(fmt.Sprintf("listen GRPC connections..."))
	for {
		select {
		case <-stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}

			switch c.Type() {
			case model.ConnectionTypeChat:
				if err := f.ChatRouter.Handle(c); err != nil {
					c.Log().Error(err.Error())
				}
			case model.ConnectionTypeForm:
				if err := f.FormRouter.Handle(c); err != nil {
					c.Log().Error(err.Error())
				}
			default:
				if err := f.GRPCRouter.Handle(c); err != nil {
					c.Log().Error(err.Error())
				}
			}

		}
	}
}

func (f *FlowManager) listenInboundEmail(stop chan struct{}, wg *sync.WaitGroup, srv model.Server) {
	defer wg.Done()
	wlog.Info(fmt.Sprintf("listen inbound email connections..."))
	for {
		select {
		case <-stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}

			if err := f.EmailRouter.Handle(c); err != nil {
				c.Log().Error(err.Error())
			}
		}
	}
}

func (f *FlowManager) listenChannelConnection(stop chan struct{}, wg *sync.WaitGroup, srv model.Server) {
	defer wg.Done()
	wlog.Info(fmt.Sprintf("listen channel connections..."))
	for {
		select {
		case <-stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}

			if err := f.ChannelRouter.Handle(c); err != nil {
				c.Log().Error(err.Error())
			}
		}
	}
}
