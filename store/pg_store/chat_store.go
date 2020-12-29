package sqlstore

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlChatStore struct {
	SqlStore
}

func NewSqlChatStore(sqlStore SqlStore) store.ChatStore {
	st := &SqlChatStore{sqlStore}
	return st
}

// New chat

func (s SqlChatStore) RoutingFromProfile(domainId, profileId int64) (*model.Routing, *model.AppError) {
	var routing *model.Routing

	err := s.GetReplica().SelectOne(&routing, `select
		p.id as source_id,
		p.name as source_name,
		'' as source_data,
		p.domain_id as domain_id,
		d.name as domain_name,
		coalesce(d.timezone_id, 287) timezone_id,
		coalesce(ct.name, 'UTC') as timezone_name,
		p.schema_id as scheme_id,
		ars.name as scheme_name,
		ars.updated_at as schema_updated_at,
		ars.debug,
		null as variables
	from chat.profile p
		inner join flow.acr_routing_scheme ars on ars.id = p.schema_id
		inner join directory.wbt_domain d on d.dc = p.domain_id
		left join flow.calendar_timezones ct on d.timezone_id = ct.id
	where p.id = :ProfileId and p.domain_id = :DomainId`, map[string]interface{}{
		"ProfileId": profileId,
		"DomainId":  domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.RoutingFromProfile", "store.sql_chat.routing.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return routing, nil
}
