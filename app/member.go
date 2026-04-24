package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) GetCallPosition(callId string) (int64, *model.AppError) {
	res, err := fm.Store.Member().CallPosition(callId)
	return res, toAppError("App.GetCallPosition", err)
}

func (f *FlowManager) GetMemberProperties(domainId int64, search *model.SearchMember, mapRes model.Variables) (model.Variables, *model.AppError) {
	res, err := f.Store.Member().GetProperties(domainId, search, mapRes)
	return res, toAppError("App.GetMemberProperties", err)
}

func (f *FlowManager) PatchMembers(domainId int64, search *model.SearchMember, patch *model.PatchMember) (int, *model.AppError) {
	res, err := f.Store.Member().PatchMembers(domainId, search, patch)
	return res, toAppError("App.PatchMembers", err)
}

func (f *FlowManager) EWTPuzzle(domainId int64, callId string, min int, queueIds []int, bucketIds []int) (float64, *model.AppError) {
	res, err := f.Store.Member().EWTPuzzle(domainId, callId, min, queueIds, bucketIds)
	return res, toAppError("App.EWTPuzzle", err)
}
