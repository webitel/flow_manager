package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) GetCallPosition(callId string) (int64, *model.AppError) {
	return fm.Store.Member().CallPosition(callId)
}
