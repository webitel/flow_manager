package sqlstore

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlListStore struct {
	SqlStore
}

func NewSqlListStore(sqlStore SqlStore) store.ListStore {
	st := &SqlListStore{sqlStore}
	return st
}

func (s SqlListStore) CheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError) {
	var exists bool
	err := s.GetReplica().SelectOne(&exists, `select exists(
	   select 1
	   from cc_list_communications c
	   where c.list_id = (
		   select l.id from cc_list l
		   where l.domain_id = :DomainId and
				 (l.id = :ListId or l.name = :Name)
		   limit 1
	   )
	   and c.number = :Number
	)`, map[string]interface{}{
		"DomainId": domainId,
		"ListId":   listId,
		"Name":     listName,
		"Number":   number,
	})

	if err != nil {
		return false, model.NewAppError("SqlListStore.CheckNumber", "store.sql_list.check_number.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), extractCodeFromErr(err))
	}

	return exists, nil
}
