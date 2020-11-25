package sqlstore

import (
	"encoding/json"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"net/http"
)

type SqlEndpointStore struct {
	SqlStore
}

func NewSqlEndpointStore(sqlStore SqlStore) store.EndpointStore {
	st := &SqlEndpointStore{sqlStore}
	return st
}

func (s SqlEndpointStore) Get(domainId int64, callerName, callerNumber string, endpoints model.Applications) ([]*model.Endpoint, *model.AppError) {
	request, err := json.Marshal(endpoints)
	var res []*model.Endpoint

	if err != nil {
		return nil, model.NewAppError("SqlEndpointStore.Get", "store.sql_endpoint.parse.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), http.StatusBadRequest)
	}

	_, err = s.GetReplica().Select(&res, `with endpoints as (
    select t.*
    from jsonb_array_elements(:Request::jsonb) with ordinality as t (endpoint, idx)
)
select e.idx, res.id, res.name, coalesce(e.endpoint->>'type', '') as type_name, res.dnd, res.destination, coalesce(res.variables, '{}')::text[] as variables 
from endpoints e
 left join lateral (
     select u.id::int8 as id, coalesce(u.name, u.username)::varchar as name, u.dnd, u.extension as destination, array[
            'sip_h_X-Webitel-Direction=internal',
            'sip_h_X-Webitel-User-Id=' || u.id,
            'sip_h_X-Webitel-Domain-Id=' || u.dc,

            E'effective_callee_id_name=''' || coalesce(u.name, u.username) || '''',
            E'effective_callee_id_number=' || coalesce(u.extension, '') || ''

			--E'origination_caller_id_name="' || :CallerName || '"',
            --E'origination_caller_id_number="' || :CallerNumber || '"'
        ]::text[] variables
     from directory.wbt_user u
     where (e.endpoint->>'type')::varchar = 'user' and u.dc = :DomainId and
           ( u.extension = (e.endpoint->>'extension')::varchar or
             u.id = (e.endpoint->>'id')::bigint)

     union all

     select g.id::int8, g.name::varchar, case when g.register and g.enable then reg.state != 3 else g.enable is false end as dnd, g.proxy destination,
           case when g.register is true then
                array['sip_h_X-Webitel-Direction=outbound',
                    E'sip_auth_username=' || g.username,
                    E'sip_auth_password=' || g.password,
                    E'sip_from_uri=' || g.account,
					'sip_h_X-Webitel-Gateway-Id=' || g.id
                ]
            else
                array[
					'sip_from_host=' || g.host,
                    'sip_h_X-Webitel-Direction=outbound',
					'sip_h_X-Webitel-Gateway-Id=' || g.id
                ]
            end vars
     from directory.sip_gateway g
        left join directory.sip_gateway_register reg on reg.id = g.id
     where  (e.endpoint->>'type')::varchar = 'gateway' and  g.dc = :DomainId and
             ( g.name = (e.endpoint->>'name')::varchar or
             g.id = (e.endpoint->>'id')::bigint)

     limit 1
 ) res on true
order by e.idx`, map[string]interface{}{
		"DomainId": domainId,
		"Request":  request,
		//"CallerName":   callerName,
		//"CallerNumber": callerNumber,
	})
	if err != nil {
		return nil, model.NewAppError("SqlEndpointStore.Get", "store.sql_endpoint.get.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), extractCodeFromErr(err))
	}

	return res, nil
}
