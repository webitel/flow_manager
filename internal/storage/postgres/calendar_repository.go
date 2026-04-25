package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type CalendarRepository struct {
	db infraSql.Store
}

func NewCalendarRepository(db infraSql.Store) store.CalendarStore {
	return &CalendarRepository{db: db}
}

const checkCalendarSQL = `select x.*
from flow.calendar_check_timing(@DomainId::int8, @Id::int, @Name::varchar) as x (name varchar, excepted varchar, accept bool, expire bool)`

func (r *CalendarRepository) Check(domainId int64, id *int, name *string) (*model.Calendar, error) {
	var calendar model.Calendar
	err := r.db.Get(context.Background(), &calendar, checkCalendarSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"Id":       id,
		"Name":     name,
	})
	if err != nil {
		return nil, fmt.Errorf("domainId=%v id=%v name=%v: %w", domainId, id, name, err)
	}
	return &calendar, nil
}

const getTimezonesSQL = `select id, sys_name from flow.calendar_timezones`

func (r *CalendarRepository) GetTimezones() ([]*model.Timezone, error) {
	var list []*model.Timezone
	if err := r.db.Select(context.Background(), &list, getTimezonesSQL, pgx.NamedArgs{}); err != nil {
		return nil, err
	}
	return list, nil
}
