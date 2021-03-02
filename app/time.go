package app

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"time"
)

func (fm *FlowManager) InitCacheTimezones() *model.AppError {
	list, err := fm.Store.Calendar().GetTimezones()
	if err != nil {
		return err
	}

	fm.timezoneList = make(map[int]*time.Location, len(list))

	for _, v := range list {
		if loc, err := time.LoadLocation(v.SysName); err != nil {
			wlog.Error(fmt.Sprintf("bad database timezone name %s, skip cache", v.SysName))
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
