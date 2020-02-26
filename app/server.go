package app

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (f *FlowManager) RegisterServers(s ...model.Server) *model.AppError {
	for _, v := range s {
		if err := v.Start(); err != nil {
			return err
		}
		wlog.Info(fmt.Sprintf("started [%s] server", v.Name()))
		f.servers = append(f.servers, v)
	}

	return nil
}

func (f *FlowManager) StopServers() {
	for _, v := range f.servers {
		wlog.Info(fmt.Sprintf("stopped [%s] server", v.Name()))
		v.Stop()
	}
}
