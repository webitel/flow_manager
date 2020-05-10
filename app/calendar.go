package app

import "github.com/webitel/flow_manager/model"

func (fm *FlowManager) CheckCalendar(domainId int64, id *int, name *string) (*model.Calendar, *model.AppError) {
	return fm.Store.Calendar().Check(domainId, id, name)
}
