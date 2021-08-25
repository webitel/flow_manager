package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) GetCallPosition(callId string) (int64, *model.AppError) {
	return fm.Store.Member().CallPosition(callId)
}

func (f *FlowManager) GetMemberProperties(domainId int64, search *model.SearchMember, mapRes model.Variables) (model.Variables, *model.AppError) {
	return f.Store.Member().GetProperties(domainId, search, mapRes)
}

func (f *FlowManager) PatchMembers(domainId int64, search *model.SearchMember, patch *model.PatchMember) (int, *model.AppError) {
	return f.Store.Member().PatchMembers(domainId, search, patch)
}

func (f *FlowManager) EWTPuzzle(domainId int64, callId string, min int, queueIds []int, bucketIds []int) (float64, *model.AppError) {
	return f.Store.Member().EWTPuzzle(callId, min, queueIds, bucketIds)
}
