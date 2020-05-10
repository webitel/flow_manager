package sqlstore

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlCalendarStore struct {
	SqlStore
}

func NewSqlCalendarStore(sqlStore SqlStore) store.CalendarStore {
	st := &SqlCalendarStore{sqlStore}
	return st
}

func (s SqlCalendarStore) Check(domainId int64, id *int, name *string) (*model.Calendar, *model.AppError) {
	var calendar *model.Calendar
	err := s.GetReplica().SelectOne(&calendar, `select x.*
from calendar_check_timing(:DomainId::int8, :Id::int, :Name::varchar ) as x  (name varchar, excepted varchar, accept bool, expire bool)`,
		map[string]interface{}{
			"DomainId": domainId,
			"Id":       id,
			"Name":     name,
		})
	if err != nil {
		return nil, model.NewAppError("SqlCalendarStore.Check", "store.sql_calendar.check.error", nil,
			fmt.Sprintf("DomainId=%v Name=%v Id=%v, %v", domainId, name, id, err.Error()), extractCodeFromErr(err))
	}

	return calendar, nil
}
