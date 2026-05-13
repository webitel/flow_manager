package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/routing"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	pgsql "github.com/webitel/flow_manager/internal/infrastructure/sql/pgsql"
	"github.com/webitel/flow_manager/store"
)

type ChatRepository struct {
	db infraSql.Store
}

func NewChatRepository(db infraSql.Store) store.ChatStore {
	return &ChatRepository{db: db}
}

const routingFromProfileSQL = `select
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
where p.id = @ProfileId and p.dc = @DomainId`

func (r *ChatRepository) RoutingFromProfile(domainId, profileId int64) (*routing.Routing, error) {
	var rt routing.Routing
	err := r.db.Get(context.Background(), &rt, routingFromProfileSQL, pgx.NamedArgs{
		"ProfileId": profileId,
		"DomainId":  domainId,
	})
	if err != nil {
		return nil, fmt.Errorf("domainId=%v profileId=%v: %w", domainId, profileId, err)
	}
	return &rt, nil
}

const routingFromSchemaIdSQL = `select
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
where ars.id = @SchemaId and ars.domain_id = @DomainId`

func (r *ChatRepository) RoutingFromSchemaId(domainId int64, schemaId int32) (*routing.Routing, error) {
	var rt routing.Routing
	err := r.db.Get(context.Background(), &rt, routingFromSchemaIdSQL, pgx.NamedArgs{
		"SchemaId": schemaId,
		"DomainId": domainId,
	})
	if err != nil {
		return nil, fmt.Errorf("domainId=%v schemaId=%v: %w", domainId, schemaId, err)
	}
	return &rt, nil
}

const getMessagesByConversationSQL = `select
    case when m.text isnull or m.text = '' and m.file_name notnull then '[' || m.file_name || ']' else m.text end as msg,
    m.created_at,
    m.type,
    case when ch.name isnull then 'Bot' else ch.name end as name,
    case when ch.internal isnull then true else ch.internal end as internal
from chat.message m
    left join chat.channel ch on m.channel_id = ch.id
where m.conversation_id = @ConversationId::uuid
  and exists(select 1 from chat.conversation c where c.id = m.conversation_id and c.domain_id = @DomainId)
order by created_at asc
limit @Limit`

func (r *ChatRepository) GetMessagesByConversation(ctx context.Context, domainId int64, conversationId string, limit int64) ([]chatdomain.ChatMessage, error) {
	var messages []chatdomain.ChatMessage
	err := r.db.Select(ctx, &messages, getMessagesByConversationSQL, pgx.NamedArgs{
		"ConversationId": conversationId,
		"DomainId":       domainId,
		"Limit":          limit,
	})
	if err != nil {
		return nil, fmt.Errorf("domainId=%v conversationId=%v: %w", domainId, conversationId, err)
	}
	return messages, nil
}

func (r *ChatRepository) LastBridged(domainId int64, number, hours string, queueIds []int, mapRes flow.Variables) (flow.Variables, error) {
	f := make([]string, 0, len(mapRes))
	for k, vi := range mapRes {
		v, _ := vi.(string)
		var val string
		switch v {
		case "extension":
			val = "extension::varchar as " + pgsql.QuoteIdentifier(k)
		case "id":
			val = "id::varchar as " + pgsql.QuoteIdentifier(k)
		case "queue_id":
			val = "queue_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "agent_id":
			val = "agent_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "description":
			val = "description::varchar as " + pgsql.QuoteIdentifier(k)
		case "created_at":
			val = "created_at::varchar as " + pgsql.QuoteIdentifier(k)
		case "gateway_id":
			val = "gateway_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "destination":
			val = "destination::varchar as " + pgsql.QuoteIdentifier(k)
		default:
			if !strings.HasPrefix(v, "variables.") {
				continue
			}
			val = fmt.Sprintf("coalesce(regexp_replace((h.variables->%s)::text, '\n|\t', ' ', 'g'), '') as %s",
				pgsql.QuoteLiteral(v[10:]), pgsql.QuoteIdentifier(k))
		}
		f = append(f, val)
	}

	q := `select row_to_json(t) variables
from (select ` + strings.Join(f, ", ") + `
      from (select c.id::text as id,
               u.extension,
               to_char(c.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') created_at,
               a.id::text as agent_id,
               ''::text as gateway_id,
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
        where (c.props->>'user')::text = @Number
              and ch.user_id notnull
              and (c.domain_id = @DomainId and c.created_at > now() - (@Hours::varchar || ' hours')::interval)
              and (@QueueIds::int[] isnull or (ah.queue_id = any (@QueueIds) or ah.queue_id isnull))
        order by c.created_at desc
        limit 1) h
      order by h.created_at desc
      limit 1) t`

	var row lastBridgedRow
	if err := r.db.Get(context.Background(), &row, q, pgx.NamedArgs{
		"DomainId": domainId,
		"Hours":    hours,
		"Number":   number,
		"QueueIds": queueIds,
	}); err != nil {
		return nil, fmt.Errorf("domainId=%v number=%v: %w", domainId, number, err)
	}

	var vars flow.Variables
	if err := json.Unmarshal(row.Variables, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

type chatProfileTypeRow struct {
	Provider *string `db:"provider"`
}

const profileTypeSQL = `select provider from chat.bot where dc = @DomainId and id = @Id`

func (r *ChatRepository) ProfileType(domainId int64, profileId int) (string, error) {
	var row chatProfileTypeRow
	err := r.db.Get(context.Background(), &row, profileTypeSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"Id":       profileId,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("domainId=%v profileId=%v: %w", domainId, profileId, err)
	}
	if row.Provider == nil {
		return "", nil
	}
	return *row.Provider, nil
}
