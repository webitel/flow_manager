package app

import (
	"fmt"

	"github.com/webitel/wlog"

	fmgrpc "github.com/webitel/flow_manager/internal/adapters/inbound/grpc"
	"github.com/webitel/flow_manager/model"
)

// Servers groups all protocol-level servers so they can be injected into
// NewFlowManager as a single value, avoiding the fx same-type ambiguity.
type Servers struct {
	Grpc    *fmgrpc.Server
	Esl     model.Server
	Mail    model.Server
	Channel model.Server
	Im      model.Server
	Http    model.Server // nil when WebHook.Addr is not configured
}

func (f *FlowManager) RegisterServers() error {
	servers := []model.Server{f.grpcServer, f.eslServer, f.mailServer, f.channelServer, f.imServer}

	for _, v := range servers {
		if err := startServer(v); err != nil {
			return err
		}
	}

	return nil
}

func (f *FlowManager) StopServers() {
	servers := []model.Server{f.grpcServer, f.eslServer, f.mailServer, f.channelServer, f.imServer}

	for _, v := range servers {
		stopServer(v)
	}
}

func startServer(s model.Server) error {
	if s == nil {
		return nil
	}
	if err := s.Start(); err != nil {
		return err
	}

	wlog.Info(fmt.Sprintf("started [%s] server", s.Name()))
	return nil
}

func stopServer(s model.Server) {
	if s == nil {
		return
	}
	s.Stop()

	wlog.Info(fmt.Sprintf("stopped [%s] server", s.Name()))
	return
}
