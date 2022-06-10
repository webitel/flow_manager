package sqlstore

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
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
     select u.id::int8 as id, coalesce(u.name, u.username)::varchar as name, coalesce(x.d, uss.dnd) dnd, u.extension as destination,
		('{' || concat_ws(',',
            E'sip_h_X-Webitel-Direction=internal',
            E'sip_h_X-Webitel-User-Id=' || u.id,
            E'sip_h_X-Webitel-Domain-Id=' || u.dc,

            E'effective_callee_id_name=''' || coalesce(u.name, u.username) || '''',
            E'effective_callee_id_number=' || coalesce(u.extension, '') || '',

            case when json_typeof(push.config->'apns') = 'array' then 'wbt_push_apn=''' || (array_to_string(array(SELECT json_array_elements_text(push.config->'apns')), ',')) || '''' end,
            case when json_typeof(push.config->'fcm') = 'array' then 'wbt_push_fcm=''' || (array_to_string(array(SELECT json_array_elements_text(push.config->'fcm')), ',')) || '''' end
           ) || '}')::text[] as variables
     from directory.wbt_user u
		left join directory.wbt_user_status uss on uss.user_id = u.id
        left join lateral ( SELECT json_object_agg(pn.typ, pn.key) AS json_object_agg
                             FROM (SELECT s.props ->> 'pn-type'::text                     AS typ,
                                          array_agg(DISTINCT s.props ->> 'pn-rpid'::text) AS key
                                   FROM directory.wbt_session s
                                   WHERE s.user_id IS NOT NULL
								     AND s.access notnull
                                     AND NULLIF(s.props ->> 'pn-rpid'::text, ''::text) IS NOT NULL
                                     AND s.user_id = u.id
									 AND now() at time zone 'UTC' < s.expires
                                   GROUP BY s.user_id, (s.props ->> 'pn-type'::text)) pn) push(config) ON true
		left join lateral (
		   select true as d
		   from call_center.cc_calls c
		   where c.user_id = u.id and c.hangup_at isnull and c.direction notnull
		   limit 1
		) x on not uss.dnd and (e.endpoint->>'idle')::bool
     where (e.endpoint->>'type')::varchar = 'user' and u.dc = :DomainId and
           ( u.extension = (e.endpoint->>'extension')::varchar or
             u.id = (e.endpoint->>'id')::bigint)

     union all

     select g.id::int8, g.name::varchar, case when g.register and g.enable then reg.state != 3 else g.enable is false end as dnd, (regexp_matches(g.proxy, '^(sip:)?([^;?]+)'))[2] destination,
           case when g.register is true then
                array['sip_h_X-Webitel-Direction=outbound',
                    E'sip_auth_username=' || g.username,
                    E'sip_auth_password=' || g.password,
                    E'sip_from_uri=' || g.account,
					'sip_h_X-Webitel-Gateway-Id=' || g.id
                ]
            else
                array[
					'sip_invite_domain=' || regexp_replace(g.host, '([a-zA-Z+.\-\d]+):?.*', '\1'),
                    'sip_h_X-Webitel-Direction=outbound',
					'sip_h_X-Webitel-Gateway-Id=' || g.id
                ]
            end vars
     from directory.sip_gateway g
        left join directory.sip_gateway_register reg on reg.id = g.id
     where  (e.endpoint->>'type')::varchar = 'gateway' and  g.dc = :DomainId and
             ( g.name = (e.endpoint->>'name')::varchar or
             g.id = (e.endpoint->>'id')::bigint or g.name = (e.endpoint->'gateway'->>'name')::varchar or g.id = (e.endpoint->'gateway'->>'id')::bigint)

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
