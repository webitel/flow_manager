package sqlstore

import (
	"database/sql"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlCallRoutingStore struct {
	SqlStore
}

func NewSqlCallRoutingStore(sqlStore SqlStore) store.CallRoutingStore {
	st := &SqlCallRoutingStore{sqlStore}
	return st
}

func (s SqlCallRoutingStore) FromQueue(domainId int64, queueId int) (*model.Routing, *model.AppError) {
	var routing *model.Routing
	err := s.GetReplica().SelectOne(&routing, `select
        sg.id as source_id,
        sg.name as source_name,
        '' as source_data,
        d.dc as domain_id,
        d.name as domain_name,
        coalesce(d.timezone_id, 287) timezone_id,
        coalesce(ct.sys_name, 'UTC') as timezone_name,
        sg.schema_id scheme_id,
        ars.name as scheme_name,
        ars.updated_at as schema_updated_at,
        ars.debug,
        null as variables
from call_center.cc_queue sg
        left join directory.wbt_domain d on sg.domain_id = d.dc
        left join flow.calendar c on c.id = sg.calendar_id
        left join flow.calendar_timezones ct on d.timezone_id = ct.id
        inner join flow.acr_routing_scheme ars on ars.id = sg.schema_id
where sg.id = :QueueId and sg.domain_id = :DomainId`, map[string]interface{}{"QueueId": queueId, "DomainId": domainId})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFoundRoute
		}
		return nil, model.NewAppError("SqlCallRoutingStore.FromQueue", "store.sql_call_routing.from_queue.error", nil,
			fmt.Sprintf("domainId=%v queueId=%v, %v", domainId, queueId, err.Error()), extractCodeFromErr(err))
	}
	return routing, nil
}

func (s SqlCallRoutingStore) FromGateway(domainId int64, gatewayId int) (*model.Routing, *model.AppError) {
	var routing *model.Routing
	err := s.GetReplica().SelectOne(&routing, `select
				sg.id as source_id,
				sg.name as source_name,
				'' as source_data,
				d.dc as domain_id,
				d.name as domain_name,
				coalesce(d.timezone_id, 287) timezone_id,
				coalesce(ct.sys_name, 'UTC') as timezone_name,
				sg.scheme_id,
				ars.name as scheme_name,
				ars.updated_at as schema_updated_at,
				ars.debug,
				null as variables
		from directory.sip_gateway sg
				left join directory.wbt_domain d on sg.dc = d.dc
				left join flow.calendar_timezones ct on d.timezone_id = ct.id
				inner join flow.acr_routing_scheme ars on ars.id = sg.scheme_id
        where sg.id = :GatewayId and sg.dc = :DomainId`, map[string]interface{}{"GatewayId": gatewayId, "DomainId": domainId})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFoundRoute
		}
		return nil, model.NewAppError("SqlCallRoutingStore.FromGateway", "store.sql_call_routing.from_gateway.error", nil,
			fmt.Sprintf("domainId=%v gatewayId=%v, %v", domainId, gatewayId, err.Error()), extractCodeFromErr(err))
	}
	return routing, nil
}

func (s SqlCallRoutingStore) SearchToDestination(domainId int64, destination string) (*model.Routing, *model.AppError) {
	var routing *model.Routing
	err := s.GetReplica().SelectOne(&routing, `select
    r.id as source_id,
    r.name as source_name,
	r.pattern as source_data,
    d.dc as domain_id,
    d.name as domain_name,
    coalesce(d.timezone_id, 287) timezone_id,
    coalesce(ct.sys_name, 'UTC') as timezone_name,
	r.scheme_id,
	ars.name as scheme_name,
	ars.updated_at as schema_updated_at,
    ars.debug,
    null as variables
from flow.acr_routing_outbound_call r
    left join directory.wbt_domain d on d.dc = r.domain_id
    left join flow.calendar_timezones ct on d.timezone_id = ct.id
    inner join flow.acr_routing_scheme ars on ars.id = r.scheme_id
where r.domain_id = :DomainId and (not r.disabled) and :Destination::varchar(50) ~ r.pattern
order by r.pos desc
limit 1`, map[string]interface{}{"DomainId": domainId, "Destination": destination})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFoundRoute
		}

		return nil, model.NewAppError("SqlCallRoutingStore.SearchToDestination", "store.sql_call_routing.search_dest.error", nil,
			fmt.Sprintf("domainId=%v dest=%v, %v", domainId, destination, err.Error()), extractCodeFromErr(err))
	}
	return routing, nil
}
