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
	   from call_center.cc_list_communications c
	   where c.list_id = (
		   select l.id from call_center.cc_list l
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

func (s SqlListStore) AddDestination(domainId int64, entry *model.SearchEntity, comm *model.ListCommunication) *model.AppError {
	_, err := s.GetMaster().Exec(`insert into call_center.cc_list_communications (list_id, number, description, expire_at)
select l.id, :Destination, :Description, :ExpireAt
from call_center.cc_list l
where l.domain_id = :DomainId
    and (l.id = :Id::int8 or l.name = :Name)
order by l.id
limit 1 on conflict do nothing`, map[string]interface{}{
		"DomainId":    domainId,
		"Id":          entry.Id,
		"Name":        entry.Name,
		"Destination": comm.Destination,
		"Description": comm.Description,
		"ExpireAt":    comm.ExpireAt,
	})

	if err != nil {
		return model.NewAppError("SqlListStore.AddDestination", "store.sql_list.add_destination.error", nil,
			fmt.Sprintf("domainId=%v %v", domainId, err.Error()), extractCodeFromErr(err))
	}

	return nil
}

func (s SqlListStore) CleanExpired() (int64, *model.AppError) {
	res, err := s.GetMaster().Exec(`delete 
from call_center.cc_list_communications
where expire_at < now()`)

	if err != nil {
		return 0, model.NewAppError("SqlListStore.CleanExpired", "store.sql_list.clean_expired.error", nil,
			err.Error(), extractCodeFromErr(err))
	}

	count, _ := res.RowsAffected()
	return count, nil
}
