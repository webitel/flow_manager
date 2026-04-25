package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

func (fm *FlowManager) InitCacheTimezones() *model.AppError {
	list, storeErr := fm.Store.Calendar().GetTimezones()
	if storeErr != nil {
		return model.NewAppError("InitCacheTimezones", "store.calendar.get_timezones", nil, storeErr.Error(), http.StatusInternalServerError)
	}

	fm.timezoneList = make(map[int]*time.Location, len(list))

	for _, v := range list {
		if loc, err := time.LoadLocation(v.SysName); err != nil {
			wlog.Warn(fmt.Sprintf("bad database timezone name %s, skip cache", v.SysName))
		} else {
			fm.timezoneList[v.Id] = loc
		}
	}

	return nil
}

func (fm *FlowManager) GetLocation(id int) *time.Location {
	loc, _ := fm.timezoneList[id]
	return loc
}
