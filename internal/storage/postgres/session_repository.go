package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/internal/storage"
)

type SessionRepository struct {
	db infraSql.Store
}

func NewSessionRepository(db infraSql.Store) storage.SessionStore {
	return &SessionRepository{db: db}
}

const touchSessionSQL = `
insert into flow.session(id, app_id)
values (@Id, @AppId)
on conflict (id)
    DO UPDATE SET seq        = session.seq + 1,
                  updated_at = now()
    where session.app_id = @AppId
returning session.seq
`

const (
	removeSessionSQL    = `delete from flow.session where id = @Id and app_id = @AppId`
	removeAllSessionSQL = `delete from flow.session where app_id = @AppId`
)

type touchRow struct {
	Seq *int `db:"seq"`
}

func (r *SessionRepository) Touch(id, appId string) (*int, error) {
	var row touchRow
	err := r.db.Get(context.Background(), &row, touchSessionSQL, pgx.NamedArgs{
		"Id":    id,
		"AppId": appId,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("id=%v appId=%v: %w", id, appId, err)
	}
	return row.Seq, nil
}

func (r *SessionRepository) Remove(id, appId string) error {
	return r.db.Exec(context.Background(), removeSessionSQL, pgx.NamedArgs{
		"Id":    id,
		"AppId": appId,
	})
}

func (r *SessionRepository) RemoveAll(appId string) error {
	return r.db.Exec(context.Background(), removeAllSessionSQL, pgx.NamedArgs{
		"AppId": appId,
	})
}
