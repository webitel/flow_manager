package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) StoreLog(schemaId int, connId string, log []*model.StepLog) *model.AppError {
	if log == nil || len(log) == 0 {
		return nil
	}
	return f.Store.Log().Save(schemaId, connId, log)
}
