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
