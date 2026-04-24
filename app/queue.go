package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
)

func (f *FlowManager) FindQueueByName(domainId int64, name string) (int32, *model.AppError) {
	id, err := f.Store.Queue().FindQueueByName(domainId, name)
	if err != nil {
		return 0, model.NewAppError("FindQueueByName", "store.queue.find_by_name", nil, err.Error(), http.StatusInternalServerError)
	}
	return id, nil
}
