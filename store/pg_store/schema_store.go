package sqlstore

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlSchemaStore struct {
	SqlStore
}

func NewSqlSchemaStore(sqlStore SqlStore) store.SchemaStore {
	st := &SqlSchemaStore{sqlStore}
	return st
}

func (s SqlSchemaStore) Get(domainId, id int) (*model.Schema, *model.AppError) {
	var out *model.Schema
	if err := s.GetReplica().SelectOne(&out, `select s.id, s.domain_id, d.name as domain_name, s.name, s.scheme as schema, s.type, s.updated_at
from acr_routing_scheme s
    inner join directory.wbt_domain d on d.dc = s.domain_id
where s.domain_id = :DomainId and s.id = :Id`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       id,
	}); err != nil {
		return nil, model.NewAppError("SqlSchemaStore.Get", "store.sql_schema.get.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return out, nil
}

func (s SqlSchemaStore) GetUpdatedAt(id int) (int64, *model.AppError) {
	i, err := s.GetReplica().SelectInt(`select s.updated_at
from acr_routing_scheme s
where id = :Id`, map[string]interface{}{
		"Id": id,
	})

	if err != nil {
		return 0, model.NewAppError("SqlSchemaStore.GetUpdatedAt", "store.sql_schema.get_time.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return i, nil
}
