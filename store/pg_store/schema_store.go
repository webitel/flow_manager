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

func (s SqlSchemaStore) Get(domainId int64, id int) (*model.Schema, *model.AppError) {
	var out *model.Schema
	if err := s.GetReplica().SelectOne(&out, `select s.id, s.domain_id, d.name as domain_name, s.name, s.scheme as schema, s.type, s.updated_at
from flow.acr_routing_scheme s
    inner join directory.wbt_domain d on d.dc = s.domain_id
where s.domain_id = :DomainId and s.id = :Id`, map[string]interface{}{
		"DomainId": domainId,
		"Id":       id,
	}); err != nil {
		return nil, model.NewAppError("SqlSchemaStore.Get", "store.sql_schema.get.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return out, nil
}

func (s SqlSchemaStore) GetUpdatedAt(domainId int64, id int) (int64, *model.AppError) {
	var i int64
	err := s.GetReplica().SelectOne(&i, `select s.updated_at
from flow.acr_routing_scheme s
where id = :Id and domain_id = :DomainId::int8`, map[string]interface{}{
		"Id":       id,
		"DomainId": domainId,
	})

	if err != nil {
		return 0, model.NewAppError("SqlSchemaStore.GetUpdatedAt", "store.sql_schema.get_time.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return i, nil
}

func (s SqlSchemaStore) GetTransferredRouting(domainId int64, schemaId int) (*model.Routing, *model.AppError) {
	var res *model.Routing
	err := s.GetReplica().SelectOne(&res, `select
        sg.id as source_id,
        sg.name as source_name,
        'transfer' as source_data,
        d.dc as domain_id,
        d.name as domain_name,
        coalesce(d.timezone_id, 287) timezone_id,
        coalesce(ct.sys_name, 'UTC') as timezone_name,
        sg.id scheme_id,
        sg.name as scheme_name,
        sg.updated_at as schema_updated_at,
        sg.debug,
        null as variables
from flow.acr_routing_scheme sg
        left join directory.wbt_domain d on sg.domain_id = d.dc
        left join flow.calendar_timezones ct on d.timezone_id = ct.id
where sg.id = :SchemaId and sg.domain_id = :DomainId`, map[string]interface{}{
		"SchemaId": schemaId,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlSchemaStore.GetTransferredRouting", "store.sql_schema.get_transferred.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return res, nil
}
