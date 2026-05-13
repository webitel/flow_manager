package app

import bscfg "github.com/webitel/flow_manager/internal/bootstrap/config"

func (f *FlowManager) Config() *bscfg.Config {
	return f.config
}
