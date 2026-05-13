package store_adapter

import (
	"context"
	"errors"

	"github.com/webitel/flow_manager/internal/infrastructure/cache"
)

// ErrExternalStoreDisabled is returned when the external SQL feature is
// disabled (cfg.ExternalSql == false).
var ErrExternalStoreDisabled = errors.New("external store disabled")

// SetExternalStore wires the optional ExternalStoreManager after construction.
func (a *Adapter) SetExternalStore(es *cache.ExternalStoreManager) {
	a.externalStore = es
}

// GetSqlDb returns the cached connection for the given driver/DSN pair.
func (a *Adapter) GetSqlDb(driver, dns string) (*cache.ExternalDb, error) {
	if a.externalStore == nil {
		return nil, ErrExternalStoreDisabled
	}
	return a.externalStore.Connect(driver, dns)
}

// SqlQuery executes a raw SQL query against an external database.
func (a *Adapter) SqlQuery(ctx context.Context, driver, dns, query string, params []interface{}) (map[string]interface{}, error) {
	db, err := a.GetSqlDb(driver, dns)
	if err != nil {
		return nil, err
	}
	return db.Query(ctx, query, params)
}
