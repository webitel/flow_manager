package app

import (
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
