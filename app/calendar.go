package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) CheckCalendar(domainId int64, id *int, name *string) (*model.Calendar, *model.AppError) {
	c, err := fm.Store.Calendar().Check(domainId, id, name)
	if err != nil {
		return nil, model.NewAppError("CheckCalendar", "store.calendar.check", nil, err.Error(), http.StatusInternalServerError)
	}
	return c, nil
}
