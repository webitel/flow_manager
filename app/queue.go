package app

import "github.com/webitel/flow_manager/model"

func (f *FlowManager) FindQueueByName(domainId int64, name string) (int32, *model.AppError) {
	return f.Store.Queue().FindQueueByName(domainId, name)
}
