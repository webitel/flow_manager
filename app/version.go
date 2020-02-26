package app

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
)

func (f *FlowManager) Version() string {
	return Version()
}

func Version() string {
	return fmt.Sprintf("%s [build:%s]", model.CurrentVersion, model.BuildNumber)
}
