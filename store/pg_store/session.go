package sqlstore

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SQLSessionStore struct {
	SqlStore
}

func NewSQLSessionStore(sqlStore SqlStore) store.SessionStore {
	s := &SQLSessionStore{sqlStore}
	return s
}

func (s *SQLSessionStore) TouchSession(id, appId string) (*int, error) {
	i, err := s.GetMaster().SelectNullInt(`insert into flow.session(id, app_id)
values (:Id, :AppId)
on conflict (id)
    DO UPDATE SET seq = session.seq + 1,
                  updated_at = now()
    where session.app_id=:AppId
returning session.seq`, map[string]any{
		"Id":    id,
		"AppId": appId,
	})

	if err != nil {
		return nil, model.NewAppError("SQLSessionStore.TouchSession", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	if i.Valid {
		val := int(i.Int64)
		return &val, nil
	}

	return nil, nil
}
