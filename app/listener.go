package app

import (
	"fmt"
	"sync"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (f *FlowManager) Listen() {
	f.log.Info("listening connections...")
	defer f.log.Info("stopped listen new connection")
	defer close(f.stopped)
	var wg sync.WaitGroup

	f.callWatcher.Start()
	f.listWatcher.Start()

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

	if f.httpServer != nil {
		wg.Add(1)
		go f.listenWebHookConnection(f.stop, &wg, f.httpServer)
	}

	wg.Wait()
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

func (f *FlowManager) listenWebHookConnection(stop chan struct{}, wg *sync.WaitGroup, srv model.Server) {
	defer wg.Done()
	f.log.Info("listen web hook connections...")
	for {
		select {
		case <-stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}

			if err := f.WebHookRouter.Handle(c); err != nil {
				c.Log().Error(err.Error())
			}
		}
	}
}
