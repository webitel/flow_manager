package app

import (
	"fmt"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (f *FlowManager) RegisterServers() *model.AppError {
	err := startServer(f.grpcServer)
	if err != nil {
		return err
	}
	err = startServer(f.eslServer)
	if err != nil {
		return err
	}
	err = startServer(f.mailServer)
	if err != nil {
		return err
	}
	err = startServer(f.channelServer)
	if err != nil {
		return err
	}

	return nil
}

func (f *FlowManager) StopServers() {
	stopServer(f.grpcServer)
	stopServer(f.eslServer)
	stopServer(f.mailServer)
	stopServer(f.channelServer)
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
