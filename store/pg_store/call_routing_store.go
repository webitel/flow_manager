package sqlstore

import (
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

func (s SqlCallRoutingStore) FromGateway(domainId, gatewayId int) (*model.Routing, *model.AppError) {
	var routing *model.Routing
	err := s.GetReplica().SelectOne(&routing, `select
				sg.id as source_id,
				sg.name as source_name,
				'' as source_data,
				d.dc as domain_id,
				d.name as domain_name,
				coalesce(d.timezone_id, 287) timezone_id,
				coalesce(ct.name, 'UTC') as timezone_name,
				sg.scheme_id,
				ars.name as scheme_name,
				ars.updated_at as schema_updated_at,
				ars.debug,
				null as variables
		from directory.sip_gateway sg
				left join directory.wbt_domain d on sg.dc = d.dc
				left join calendar_timezones ct on d.timezone_id = ct.id
				inner join acr_routing_scheme ars on ars.id = sg.scheme_id
        where sg.id = :GatewayId and sg.dc = :DomainId`, map[string]interface{}{"GatewayId": gatewayId, "DomainId": domainId})

	if err != nil {
		return nil, model.NewAppError("SqlCallRoutingStore.FromGateway", "store.sql_call_routing.from_gateway.error", nil,
			fmt.Sprintf("domainId=%v gatewayId=%v, %v", domainId, gatewayId, err.Error()), extractCodeFromErr(err))
	}
	return routing, nil
}
