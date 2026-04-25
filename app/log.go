package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
)

func (f *FlowManager) StoreLog(schemaId int, connId string, log []*model.StepLog) *model.AppError {
	if log == nil || len(log) == 0 {
		return nil
	}
	if err := f.Store.Log().Save(schemaId, connId, log); err != nil {
		return model.NewAppError("StoreLog", "store.log.save", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}
