package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type CallRoutingRepository struct {
	db infraSql.Store
}

func NewCallRoutingRepository(db infraSql.Store) store.CallRoutingStore {
	return &CallRoutingRepository{db: db}
}

const fromQueueSQL = `select
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
where sg.id = @QueueId and sg.domain_id = @DomainId`

func (r *CallRoutingRepository) FromQueue(domainId int64, queueId int) (*model.Routing, error) {
	var routing model.Routing
	err := r.db.Get(context.Background(), &routing, fromQueueSQL, pgx.NamedArgs{
		"QueueId":  queueId,
		"DomainId": domainId,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFoundRoute
		}
		return nil, fmt.Errorf("domainId=%v queueId=%v: %w", domainId, queueId, err)
	}
	return &routing, nil
}

const fromGatewaySQL = `select
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
where sg.id = @GatewayId and sg.dc = @DomainId`

func (r *CallRoutingRepository) FromGateway(domainId int64, gatewayId int) (*model.Routing, error) {
	var routing model.Routing
	err := r.db.Get(context.Background(), &routing, fromGatewaySQL, pgx.NamedArgs{
		"GatewayId": gatewayId,
		"DomainId":  domainId,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFoundRoute
		}
		return nil, fmt.Errorf("domainId=%v gatewayId=%v: %w", domainId, gatewayId, err)
	}
	return &routing, nil
}

const searchToDestinationSQL = `select
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
where r.domain_id = @DomainId and (not r.disabled) and @Destination::varchar(50) ~ r.pattern
order by r.pos desc
limit 1`

func (r *CallRoutingRepository) SearchToDestination(domainId int64, destination string) (*model.Routing, error) {
	var routing model.Routing
	err := r.db.Get(context.Background(), &routing, searchToDestinationSQL, pgx.NamedArgs{
		"DomainId":    domainId,
		"Destination": destination,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFoundRoute
		}
		return nil, fmt.Errorf("domainId=%v dest=%v: %w", domainId, destination, err)
	}
	return &routing, nil
}
