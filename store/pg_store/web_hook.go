package sqlstore

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlWebHookStore struct {
	SqlStore
}

func NewSqlWebHookStore(sqlStore SqlStore) store.WebHookStore {
	st := &SqlWebHookStore{sqlStore}
	return st
}

func (s SqlWebHookStore) Get(id string) (model.WebHook, *model.AppError) {
	var hook model.WebHook
	err := s.GetReplica().SelectOne(&hook, `select key, name, schema_id, origin, domain_id, "authorization"
from flow.web_hook
where key = :Id;`, map[string]interface{}{
		"Id": id,
	})

	if err != nil {
		return hook, model.NewAppError("SqlWebHookStore.Get", "store.sql_hook.get.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return hook, nil
}
