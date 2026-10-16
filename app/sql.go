package app

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store/cachelayer"
)

var ErrExternalStoreDisabled = model.NewAppError("App", "app.settings.sql.external.disabled", nil, "External store disabled", http.StatusForbidden)

func (fm *FlowManager) GetSqlDb(ctx context.Context, driver, dns string, writable bool) (*cachelayer.ExternalDb, *model.AppError) {
	if fm.ExternalStore == nil {
		return nil, ErrExternalStoreDisabled
	}

	connectCfg := cachelayer.ExternalSQLStoreConnectConfig{
		Driver:   driver,
		DSN:      dns,
		Writable: writable,
	}

	return fm.ExternalStore.Connect(ctx, &connectCfg)
}
