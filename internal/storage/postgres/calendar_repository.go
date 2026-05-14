package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/flow_manager/internal/domain/calendar"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/internal/storage"
)

type CalendarRepository struct {
	db infraSql.Store
}

func NewCalendarRepository(db infraSql.Store) storage.CalendarStore {
	return &CalendarRepository{db: db}
}

const checkCalendarSQL = `select x.*
from flow.calendar_check_timing(@DomainId::int8, @Id::int, @Name::varchar) as x (name varchar, excepted varchar, accept bool, expire bool)`

func (r *CalendarRepository) Check(domainId int64, id *int, name *string) (*calendar.Calendar, error) {
	var cal calendar.Calendar
	err := r.db.Get(context.Background(), &cal, checkCalendarSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"Id":       id,
		"Name":     name,
	})
	if err != nil {
		return nil, fmt.Errorf("domainId=%v id=%v name=%v: %w", domainId, id, name, err)
	}
	return &cal, nil
}

const getTimezonesSQL = `select id, sys_name from flow.calendar_timezones`

func (r *CalendarRepository) GetTimezones() ([]*calendar.Timezone, error) {
	var list []*calendar.Timezone
	if err := r.db.Select(context.Background(), &list, getTimezonesSQL, pgx.NamedArgs{}); err != nil {
		return nil, err
	}
	return list, nil
}
