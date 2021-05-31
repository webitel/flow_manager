package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) SetCallGranteeId(domainId int64, id string, granteeId int64) *model.AppError {
	return f.Store.Call().SetGranteeId(domainId, id, granteeId)
}
