package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type ListRepository struct {
	db infraSql.Store
}

func NewListRepository(db infraSql.Store) store.ListStore {
	return &ListRepository{db: db}
}

type checkNumberRow struct {
	Exists bool `db:"exists"`
}

const checkNumberSQL = `select exists(
    select 1
    from call_center.cc_list_communications c
    where c.list_id = (
        select l.id from call_center.cc_list l
        where l.domain_id = @DomainId
          and (l.id = @ListId or l.name = @Name)
        limit 1
    )
    and c.number = @Number
)`

func (r *ListRepository) CheckNumber(domainId int64, number string, listId *int, listName *string) (bool, error) {
	var row checkNumberRow
	if err := r.db.Get(context.Background(), &row, checkNumberSQL, pgx.NamedArgs{
		"DomainId": domainId,
		"ListId":   listId,
		"Name":     listName,
		"Number":   number,
	}); err != nil {
		return false, fmt.Errorf("domainId=%v number=%v: %w", domainId, number, err)
	}
	return row.Exists, nil
}

const addDestinationSQL = `insert into call_center.cc_list_communications (list_id, number, description, expire_at)
select l.id, @Destination, @Description, @ExpireAt
from call_center.cc_list l
where l.domain_id = @DomainId
  and (l.id = @Id::int8 or l.name = @Name)
order by l.id
limit 1 on conflict do nothing`

func (r *ListRepository) AddDestination(domainId int64, entry *model.SearchEntity, comm *model.ListCommunication) error {
	return r.db.Exec(context.Background(), addDestinationSQL, pgx.NamedArgs{
		"DomainId":    domainId,
		"Id":          entry.Id,
		"Name":        entry.Name,
		"Destination": comm.Destination,
		"Description": comm.Description,
		"ExpireAt":    comm.ExpireAt,
	})
}

const cleanExpiredSQL = `delete from call_center.cc_list_communications where expire_at < now()`

func (r *ListRepository) CleanExpired() (int64, error) {
	return r.db.ExecResult(context.Background(), cleanExpiredSQL, pgx.NamedArgs{})
}
