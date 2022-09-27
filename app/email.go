package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) GetEmailProperties(domainId int64, id *int64, messageId *string, mapRes model.Variables) (model.Variables, *model.AppError) {
	return f.Store.Email().GerProperties(domainId, id, messageId, mapRes)
}
