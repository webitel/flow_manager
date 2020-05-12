package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) ListCheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError) {
	return fm.Store.List().CheckNumber(domainId, number, listId, listName)
}
