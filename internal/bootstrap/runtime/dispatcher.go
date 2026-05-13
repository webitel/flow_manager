// Package runtime contains the main dispatcher that connects inbound transport
// servers to domain routers.
package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/workers/runtime_recovery"
	"github.com/webitel/flow_manager/internal/workers/session_recovery"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/session"
)

// Dispatcher connects inbound transport servers to domain routers and manages
// background worker goroutines.
type Dispatcher struct {
	log *wlog.Logger
	id  string

	eslServer     flow.Server
	mailServer    flow.Server
	channelServer flow.Server
	imServer      flow.Server
	grpcServer    grpcServer

	CallRouter    flow.Router
	GRPCRouter    flow.Router
	ChatRouter    flow.Router
	FormRouter    flow.Router
	EmailRouter   flow.Router
	ChannelRouter flow.Router
	IMRouter      flow.Router

	checkpointRepo   session.Repository
	runtimeStateRepo persistence.Repository

	stop    chan struct{}
	stopped chan struct{}
}

// grpcServer is the minimal interface we need from the gRPC server in the
// dispatcher (just Consume).
type grpcServer interface {
	Consume() <-chan flow.Connection
}

// DispatcherConfig carries all dependencies for the dispatcher.
type DispatcherConfig struct {
	Log *wlog.Logger
	ID  string

	GrpcServer    grpcServer
	EslServer     flow.Server
	MailServer    flow.Server
	ChannelServer flow.Server
	ImServer      flow.Server

	CheckpointRepo   session.Repository
	RuntimeStateRepo persistence.Repository

	Stop    chan struct{}
	Stopped chan struct{}
}

// New creates a Dispatcher.
func New(cfg DispatcherConfig) *Dispatcher {
	return &Dispatcher{
		log:              cfg.Log,
		id:               cfg.ID,
		grpcServer:       cfg.GrpcServer,
		eslServer:        cfg.EslServer,
		mailServer:       cfg.MailServer,
		channelServer:    cfg.ChannelServer,
		imServer:         cfg.ImServer,
		checkpointRepo:   cfg.CheckpointRepo,
		runtimeStateRepo: cfg.RuntimeStateRepo,
		stop:             cfg.Stop,
		stopped:          cfg.Stopped,
	}
}

// CheckpointRepo returns the session-checkpoint repository.
func (f *Dispatcher) CheckpointRepo() session.Repository { return f.checkpointRepo }

// RuntimeStateRepo returns the resumable-flow state repository.
func (f *Dispatcher) RuntimeStateRepo() persistence.Repository { return f.runtimeStateRepo }

// Listen blocks until all transport goroutines finish. It should be called in
// its own goroutine.
func (f *Dispatcher) Listen() {
	f.log.Info("listening connections...")
	defer f.log.Info("stopped listen new connection")
	defer close(f.stopped)
	var wg sync.WaitGroup

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

	if f.eslServer != nil {
		wg.Add(1)
		go f.listenCallConnection(&wg, f.eslServer)
	}

	if f.grpcServer != nil {
		wg.Add(1)
		go f.listenGrpcConnection(&wg, f.grpcServer)
	}

	if f.mailServer != nil {
		wg.Add(1)
		go f.listenInboundEmail(&wg, f.mailServer)
	}

	if f.channelServer != nil {
		wg.Add(1)
		go f.listenChannelConnection(&wg, f.channelServer)
	}

	if f.imServer != nil {
		wg.Add(1)
		go f.listenImConnection(&wg, f.imServer)
	}

	wg.Wait()
}

func (f *Dispatcher) listenImConnection(wg *sync.WaitGroup, srv flow.Server) {
	defer wg.Done()
	f.log.Info("listen im connections...")
	for {
		select {
		case <-f.stop:
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

func (f *Dispatcher) listenCallConnection(wg *sync.WaitGroup, srv flow.Server) {
	defer wg.Done()
	f.log.Info("listen call connections...")
	for {
		select {
		case <-f.stop:
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

func (f *Dispatcher) listenGrpcConnection(wg *sync.WaitGroup, srv grpcServer) {
	defer wg.Done()
	wlog.Info(fmt.Sprintf("listen GRPC connections..."))
	for {
		select {
		case <-f.stop:
			return
		case c, ok := <-srv.Consume():
			if !ok {
				return
			}

			switch c.Type() {
			case flow.ConnectionTypeChat:
				if err := f.ChatRouter.Handle(c); err != nil {
					c.Log().Error(err.Error())
				}
			case flow.ConnectionTypeForm:
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

func (f *Dispatcher) listenInboundEmail(wg *sync.WaitGroup, srv flow.Server) {
	defer wg.Done()
	wlog.Info(fmt.Sprintf("listen inbound email connections..."))
	for {
		select {
		case <-f.stop:
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

func (f *Dispatcher) listenChannelConnection(wg *sync.WaitGroup, srv flow.Server) {
	defer wg.Done()
	wlog.Info(fmt.Sprintf("listen channel connections..."))
	for {
		select {
		case <-f.stop:
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
