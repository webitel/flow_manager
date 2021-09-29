package app

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store/cachelayer"
	"net/http"
)

var ErrExternalStoreDisabled = model.NewAppError("App", "app.settings.sql.external.disabled", nil, "External store disabled", http.StatusForbidden)

func (fm *FlowManager) GetSqlDb(driver, dns string) (*cachelayer.ExternalDb, *model.AppError) {
	if fm.ExternalStore == nil {
		return nil, ErrExternalStoreDisabled
	}

	return fm.ExternalStore.Connect(driver, dns)
}
