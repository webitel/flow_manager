package sqlstore

import (
	"fmt"
	"github.com/lib/pq"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
	"strings"
)

type SqlUserStore struct {
	SqlStore
}

func NewSqlUserStore(sqlStore SqlStore) store.UserStore {
	st := &SqlUserStore{sqlStore}
	return st
}

type properties struct {
	model.Variables `json:"variables" db:"variables"`
}

// TODO
func (s SqlUserStore) GetProperties(domainId int64, search *model.SearchUser, mapRes model.Variables) (model.Variables, *model.AppError) {
	f := make([]string, 0)

	for k, v := range mapRes {
		var val = ""
		switch v {
		case "name":
			val = "coalesce(u.name, u.username)::varchar as " + pq.QuoteIdentifier(k)
		case "extension":
			val = "u.extension::varchar as " + pq.QuoteIdentifier(k)
		case "dnd":
			val = "u.dnd::varchar as " + pq.QuoteIdentifier(k)
		case "agent_id":
			val = `(select a.id::text
				from call_center.cc_agent a where a.user_id = u.id limit 1)` + pq.QuoteIdentifier(k)
		case "agent_status":
			val = `(select a.status::text
				from call_center.cc_agent a where a.user_id = u.id limit 1)` + pq.QuoteIdentifier(k)
		case "super_extension":
			val = `(select su.extension::text
from cc_agent a
    inner join cc_agent s on s.id = a.supervisor_ids[1]
    inner join directory.wbt_user su on su.id = s.user_id
where a.user_id = u.id limit 1) ` + pq.QuoteIdentifier(k)

		case "admin_extension":
			val = `(select su.extension::text
from cc_agent a
    inner join cc_team t on t.id = a.team_id
    inner join cc_agent s on s.id = t.admin_id
    inner join directory.wbt_user su on su.id = s.user_id
where a.user_id = u.id limit 1) ` + pq.QuoteIdentifier(k)
		default:

			if !strings.HasPrefix(fmt.Sprintf("%s", v), "variables.") {
				continue
			}

			val = fmt.Sprintf("(u.profile->%s) as %s", pq.QuoteLiteral(fmt.Sprintf("%s", v)[10:]), pq.QuoteIdentifier(k))
		}

		f = append(f, val)
	}

	var t *properties

	err := s.GetReplica().SelectOne(&t, `select row_to_json(t) variables
from (
	select  `+strings.Join(f, ", ")+`
		from directory.wbt_user u
		where u.dc = :DomainId
		and (:Id::int8 isnull or u.id = :Id)
		and (:Extension::varchar isnull or u.extension = :Extension)
		and (:Name::varchar isnull or coalesce(u.name, u.username) = :Name::varchar)
		limit 1
) t`, map[string]interface{}{
		"Id":        search.Id,
		"Name":      search.Name,
		"Extension": search.Extension,
		"DomainId":  domainId,
	})

	if err != nil {
		return nil, model.NewAppError("SqlUserStore.Get", "store.sql_user.get.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return t.Variables, nil
}

func (s SqlUserStore) GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError) {
	res, err := s.GetReplica().SelectNullInt(`select a.id
from directory.wbt_user u
    inner join call_center.cc_agent a on a.user_id = u.id
where u.dc = :DomainId
    and u.extension = :Extension
limit 1`, map[string]interface{}{
		"DomainId":  domainId,
		"Extension": extension,
	})

	if err != nil {
		return nil, model.NewAppError("SqlUserStore.GetAgentIdByExtension", "store.sql_user.get_agent.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	if res.Valid && res.Int64 > 0 {
		r := int32(res.Int64)
		return &r, nil
	}

	return nil, nil
}
