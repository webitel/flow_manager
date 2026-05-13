package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/flow_manager/internal/domain/flow"
	userdomain "github.com/webitel/flow_manager/internal/domain/user"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	pgsql "github.com/webitel/flow_manager/internal/infrastructure/sql/pgsql"
	"github.com/webitel/flow_manager/store"
)

type UserRepository struct {
	db infraSql.Store
}

func NewUserRepository(db infraSql.Store) store.UserStore {
	return &UserRepository{db: db}
}

const getAgentIdByExtensionSQL = `
select a.id
from directory.wbt_user u
    inner join call_center.cc_agent a on a.user_id = u.id
where u.dc = @DomainId
    and u.extension = @Extension
limit 1
`

type agentIdRow struct {
	ID *int64 `db:"id"`
}

func (r *UserRepository) GetAgentIdByExtension(domainId int64, extension string) (*int32, error) {
	var row agentIdRow
	err := r.db.Get(context.Background(), &row, getAgentIdByExtensionSQL, pgx.NamedArgs{
		"DomainId":  domainId,
		"Extension": extension,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("domainId=%v extension=%v: %w", domainId, extension, err)
	}
	if row.ID == nil {
		return nil, nil
	}
	id := int32(*row.ID)
	return &id, nil
}

type userPropertiesRow struct {
	Variables json.RawMessage `db:"variables"`
}

func (r *UserRepository) GetProperties(domainId int64, search *userdomain.SearchUser, mapRes flow.Variables) (flow.Variables, error) {
	cols := make([]string, 0, len(mapRes))
	for k, vi := range mapRes {
		v, _ := vi.(string)
		var col string
		switch v {
		case "name":
			col = "coalesce(u.name, u.username)::varchar as " + pgsql.QuoteIdentifier(k)
		case "user_id":
			col = "a.user_id::varchar as " + pgsql.QuoteIdentifier(k)
		case "username":
			col = "coalesce(u.username, '')::varchar as " + pgsql.QuoteIdentifier(k)
		case "extension":
			col = "u.extension::varchar as " + pgsql.QuoteIdentifier(k)
		case "email":
			col = "coalesce(u.email::varchar, '') as " + pgsql.QuoteIdentifier(k)
		case "dnd":
			col = "u.dnd::varchar as " + pgsql.QuoteIdentifier(k)
		case "agent_id":
			col = `(select a.id::text
				from call_center.cc_agent a where a.user_id = u.id limit 1) ` + pgsql.QuoteIdentifier(k)
		case "team_id":
			col = `(select a.team_id::text
				from call_center.cc_agent a where a.user_id = u.id limit 1) ` + pgsql.QuoteIdentifier(k)
		case "team_name":
			col = `(select tm.name::text from call_center.cc_agent a left join call_center.cc_team tm on tm.id = a.team_id where a.user_id = u.id limit 1) ` + pgsql.QuoteIdentifier(k)
		case "bridged_calls":
			col = `(select count(*) from call_center.cc_calls c
where c.user_id = u.id and c.bridged_at notnull and c.hangup_at isnull) ` + pgsql.QuoteIdentifier(k)
		case "active_calls":
			col = `(select count(*) from call_center.cc_calls c
where c.user_id = u.id) ` + pgsql.QuoteIdentifier(k)
		case "agent_status":
			col = `(select a.status::text
				from call_center.cc_agent a where a.user_id = u.id limit 1) ` + pgsql.QuoteIdentifier(k)
		case "status_payload":
			col = `(select coalesce(a.status_payload, '')::text
				from call_center.cc_agent a where a.user_id = u.id limit 1) ` + pgsql.QuoteIdentifier(k)
		case "super_extension":
			col = `(select su.extension::text
from call_center.cc_agent a
    inner join call_center.cc_agent s on s.id = a.supervisor_ids[1]
    inner join directory.wbt_user su on su.id = s.user_id
where a.user_id = u.id limit 1) ` + pgsql.QuoteIdentifier(k)
		case "admin_extension":
			col = `(select su.extension::text
from call_center.cc_agent a
    inner join call_center.cc_team t on t.id = a.team_id
    inner join call_center.cc_agent s on s.id = t.admin_id
    inner join directory.wbt_user su on su.id = s.user_id
where a.user_id = u.id limit 1) ` + pgsql.QuoteIdentifier(k)
		default:
			if !strings.HasPrefix(v, "variables.") {
				continue
			}
			col = fmt.Sprintf("coalesce((u.profile->>%s)::text, '') as %s",
				pgsql.QuoteLiteral(v[10:]), pgsql.QuoteIdentifier(k))
		}
		cols = append(cols, col)
	}

	sql := `select row_to_json(t) as variables from (
		select ` + strings.Join(cols, ", ") + `
		from directory.wbt_user u
		where u.dc = @DomainId
		and (@Id::int8 isnull or u.id = @Id)
		and (case when @AgentId::int notnull then (u.id = (select a.user_id from call_center.cc_agent a where a.id = @AgentId and a.domain_id = @DomainId)) else true end)
		and (@Extension::varchar isnull or u.extension = @Extension)
		and (@Name::varchar isnull or coalesce(u.name, u.username) = @Name::varchar)
		limit 1
	) t`

	var row userPropertiesRow
	if err := r.db.Get(context.Background(), &row, sql, pgx.NamedArgs{
		"Id":        search.Id,
		"Name":      search.Name,
		"Extension": search.Extension,
		"AgentId":   search.AgentId,
		"DomainId":  domainId,
	}); err != nil {
		return nil, fmt.Errorf("domainId=%v: %w", domainId, err)
	}

	var result flow.Variables
	if err := json.Unmarshal(row.Variables, &result); err != nil {
		return nil, fmt.Errorf("unmarshal variables: %w", err)
	}
	return result, nil
}
