package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) GetUserProperties(domainId int64, search *model.SearchUser, mapRes model.Variables) (model.Variables, *model.AppError) {
	return f.Store.User().GetProperties(domainId, search, mapRes)
}
