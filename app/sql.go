package app

import (
	"context"
	"errors"

	"github.com/webitel/flow_manager/internal/infrastructure/cache"
)

var ErrExternalStoreDisabled = errors.New("external store disabled")

func (fm *FlowManager) GetSqlDb(driver, dns string) (*cache.ExternalDb, error) {
	if fm.ExternalStore == nil {
		return nil, ErrExternalStoreDisabled
	}
	return fm.ExternalStore.Connect(driver, dns)
}

func (fm *FlowManager) SqlQuery(ctx context.Context, driver, dns, query string, params []interface{}) (map[string]interface{}, error) {
	db, err := fm.GetSqlDb(driver, dns)
	if err != nil {
		return nil, err
	}
	return db.Query(ctx, query, params)
}
