package app

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"sync"
)

func (f *FlowManager) Listen() {
	wlog.Info("listening connections...")
	defer wlog.Info("stopped listen new connection")
	defer close(f.stopped)
	var wg sync.WaitGroup

	f.callWatcher.Start()

	go f.listenCallEvents(f.stop)

	for _, v := range f.servers {
		wg.Add(1)
		switch v.Type() {
		case model.ConnectionTypeCall:
			go f.listenCallConnection(f.stop, &wg, v)
		case model.ConnectionTypeGrpc:
			go f.listenGrpcConnection(f.stop, &wg, v)
		case model.ConnectionTypeEmail:
			go f.listenInboundEmail(f.stop, &wg, v)
		default:
			wg.Done()
		}
	}

	wg.Wait()
}

func (f *FlowManager) listenCallConnection(stop chan struct{}, wg *sync.WaitGroup, srv model.Server) {
	defer wg.Done()
	wlog.Info(fmt.Sprintf("listen call connections..."))
	for {
		select {
		case <-stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}

			if err := f.CallRouter.Handle(c); err != nil {
				wlog.Error(err.Error())
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
					wlog.Error(err.Error())
				}
			default:
				if err := f.GRPCRouter.Handle(c); err != nil {
					wlog.Error(err.Error())
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
				wlog.Error(err.Error())
			}
		}
	}
}
