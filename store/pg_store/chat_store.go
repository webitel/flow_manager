package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"
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
			p.dc as domain_id,
			d.name as domain_name,
			coalesce(d.timezone_id, 287) timezone_id,
			coalesce(ct.sys_name, 'UTC') as timezone_name,
			p.flow_id as scheme_id,
			ars.name as scheme_name,
			ars.updated_at as schema_updated_at,
			ars.debug,
			null as variables
		  from chat.bot p
			inner join flow.acr_routing_scheme ars on ars.id = p.flow_id
			inner join directory.wbt_domain d on d.dc = p.dc
			left join flow.calendar_timezones ct on d.timezone_id = ct.id
		  where p.id = :ProfileId and p.dc = :DomainId`, map[string]interface{}{
		"ProfileId": profileId,
		"DomainId":  domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.RoutingFromProfile", "store.sql_chat.routing.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return routing, nil
}

func (s SqlChatStore) GetMessagesByConversation(ctx context.Context, domainId int64, conversationId string, limit int64) (*[]model.ChatMessage, *model.AppError) {
	var messages []model.ChatMessage
	_, err := s.GetReplica().WithContext(ctx).Select(
		&messages,
		`select
		case when m.text isnull or m.text = '' and m.file_name notnull then '[' || m.file_name || ']' else m.text end as msg,
		m.created_at,
		m.type,
		case when ch.name isnull then 'Bot' else ch.name end,
		case when ch.internal isnull then true else ch.internal end
	FROM chat.message m
		LEFT JOIN chat.channel ch ON m.channel_id = ch.id
	WHERE m.conversation_id = :ConversationId::uuid
	and exists(select 1 from chat.conversation c where c.id = m.conversation_id and c.domain_id = :DomainId)
	ORDER BY created_at ASC
	LIMIT :Limit;`,
		map[string]interface{}{
			"ConversationId": conversationId,
			"DomainId":       domainId,
			"Limit":          limit,
		},
	)

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.RoutingFromProfile", "store.sql_chat.routing.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return &messages, nil
}

func (s SqlChatStore) RoutingFromSchemaId(domainId int64, schemaId int32) (*model.Routing, *model.AppError) {
	var routing *model.Routing

	err := s.GetReplica().SelectOne(&routing, `select
    ars.id as source_id,
    ars.name as source_name,
    '' as source_data,
    ars.domain_id as domain_id,
    d.name as domain_name,
    coalesce(d.timezone_id, 287) timezone_id,
    coalesce(ct.sys_name, 'UTC') as timezone_name,
    ars.id as scheme_id,
    ars.name as scheme_name,
    ars.updated_at as schema_updated_at,
    ars.debug,
    null as variables
from flow.acr_routing_scheme ars
    inner join directory.wbt_domain d on d.dc = ars.domain_id
    left join flow.calendar_timezones ct on d.timezone_id = ct.id
where ars.id = :SchemaId and ars.domain_id = :DomainId`, map[string]interface{}{
		"SchemaId": schemaId,
		"DomainId": domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.RoutingFromSchemaId", "store.sql_chat.routing_schema.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	return routing, nil
}

func (s SqlChatStore) LastBridged(domainId int64, number, hours string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	f := make([]string, 0)

	for k, v := range mapRes {
		var val = ""
		switch v {
		case "extension":
			val = "extension::varchar as " + pq.QuoteIdentifier(k)
		case "id":
			val = "id::varchar as " + pq.QuoteIdentifier(k)
		case "queue_id":
			val = "queue_id::varchar as " + pq.QuoteIdentifier(k)
		case "agent_id":
			val = "agent_id::varchar as " + pq.QuoteIdentifier(k)
		case "description":
			val = "description::varchar as " + pq.QuoteIdentifier(k)
		case "created_at":
			val = "created_at::varchar as " + pq.QuoteIdentifier(k)
		case "gateway_id":
			val = "gateway_id::varchar as " + pq.QuoteIdentifier(k)
		case "destination":
			val = "destination::varchar as " + pq.QuoteIdentifier(k)
		default:

			if !strings.HasPrefix(fmt.Sprintf("%s", v), "variables.") {
				continue
			}

			val = fmt.Sprintf("coalesce(regexp_replace((h.variables->%s)::text, '\n|\t', ' ', 'g'), '') as %s", pq.QuoteLiteral(fmt.Sprintf("%s", v)[10:]), pq.QuoteIdentifier(k))
		}

		f = append(f, val)
	}

	var t *properties

	err := s.GetReplica().SelectOne(&t, `select row_to_json(t) variables
from (select `+strings.Join(f, ", ")+`
      from (select c.id::text as id,
               u.extension,
               to_char(c.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') created_at,
               a.id::text as agent_id,
               ''::text as gateway_id, --bot id,
               ah.queue_id::text as queue_id,
               coalesce(ah.description, '') as description,
               coalesce(c.props, '{}') || coalesce(ch.props, '{}') as variables,
			   c.props->>'chat' as destination
        from chat.conversation c
            left join lateral (
                select ch.user_id, ch.props
                from chat.channel ch
                where ch.conversation_id = c.id
                    and ch.internal
                order by ch.created_at desc
                limit 1
            ) ch on true
            left join lateral (
                select *
                from call_center.cc_member_attempt_history ah
                where ah.domain_id = c.domain_id
                    and ah.member_call_id = c.id::varchar
            ) ah on true
            left join directory.wbt_user u on u.id = ch.user_id
            left join call_center.cc_agent a on a.user_id = ch.user_id
        where (c.props->>'user')::text = :Number and
              ch.user_id notnull
            and (c.domain_id = :DomainId and c.created_at > now() - (:Hours::varchar || ' hours')::interval)
                      and (:QueueIds::int[] isnull or (ah.queue_id = any (:QueueIds) or ah.queue_id isnull))
        order by c.created_at desc
        limit 1) h
      order by h.created_at desc
      limit 1) t`, map[string]interface{}{
		"DomainId": domainId,
		"Hours":    hours,
		"Number":   number,
		"QueueIds": pq.Array(queueIds),
	})

	if err != nil {
		return nil, model.NewAppError("SqlChatStore.LastBridgedExtension", "store.sql_chat.get_last_bridged.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return t.Variables, nil
}

func (s SqlChatStore) ProfileType(domainId int64, profileId int) (string, *model.AppError) {
	v, err := s.GetReplica().SelectNullStr(`select provider
from chat.bot
where dc = :DomainId and id = :Id`, map[string]any{
		"DomainId": domainId,
		"Id":       profileId,
	})

	if err != nil {
		return "", model.NewAppError("SqlChatStore.ProfileType", "store.sql_chat.profile_type.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return v.String, nil
}
