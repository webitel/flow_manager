package sqlstore

import (
	"context"
	"encoding/json"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlSysSettingsStore struct {
	SqlStore
}

func NewSqlSysSettingsStore(sqlStore SqlStore) store.SystemcSettings {
	st := &SqlSysSettingsStore{sqlStore}
	return st
}

func (s SqlSysSettingsStore) Get(ctx context.Context, domainId int64, name string) (json.RawMessage, *model.AppError) {
	var res json.RawMessage
	err := s.GetReplica().WithContext(ctx).SelectOne(&res, `select value
from call_center.system_settings
where domain_id = :DomainId and name = :Name`, map[string]interface{}{
		"DomainId": domainId,
		"Name":     name,
	})

	if err != nil {
		return nil, model.NewAppError("SqlSysSettingsStore.Get", "store.sql_sys_settings.get.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}
