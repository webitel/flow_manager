package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) ListCheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError) {
	return fm.Store.List().CheckNumber(domainId, number, listId, listName)
}

func (fm *FlowManager) ListAddCommunication(domainId int64, search *model.SearchEntity, comm *model.ListCommunication) *model.AppError {
	return fm.Store.List().AddDestination(domainId, search, comm)
}
