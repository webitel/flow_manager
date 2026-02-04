package app

import (
	"fmt"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (f *FlowManager) RegisterServers() *model.AppError {
	servers := []model.Server{f.grpcServer, f.eslServer, f.mailServer, f.channelServer, f.httpServer, f.imServer}

	for _, v := range servers {
		if err := startServer(v); err != nil {
			return err
		}
	}

	return nil
}

func (f *FlowManager) StopServers() {
	servers := []model.Server{f.grpcServer, f.eslServer, f.mailServer, f.channelServer, f.httpServer, f.imServer}

	for _, v := range servers {
		stopServer(v)
	}
}

func startServer(s model.Server) *model.AppError {
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
