package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) GetCallPosition(callId string) (int64, *model.AppError) {
	return fm.Store.Member().CallPosition(callId)
}

func (f *FlowManager) GetMemberProperties(domainId int64, search *model.SearchMember, mapRes model.Variables) (model.Variables, *model.AppError) {
	return f.Store.Member().GetProperties(domainId, search, mapRes)
}
