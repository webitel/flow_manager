package sqlstore

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlLogStore struct {
	SqlStore
}

func NewSqlLogStore(sqlStore SqlStore) store.LogStore {
	st := &SqlLogStore{sqlStore}
	return st
}

func (s SqlLogStore) Save(schemaId int, connId string, log interface{}) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into flow.scheme_log (schema_id, log, conn_id)
values (:SchemaId, :Log, :ConnId)`, map[string]interface{}{
		"SchemaId": schemaId,
		"Log":      model.InterfaceToJson(log),
		"ConnId":   connId,
	})

	if err != nil {
		return model.NewAppError("SqlLogStore.Save", "store.sql_log.save.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}
