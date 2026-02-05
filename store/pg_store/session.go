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

func (s *SQLSessionStore) Touch(id, appId string) (*int, error) {
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
		return nil, model.NewAppError("SQLSessionStore.Touch", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	if i.Valid {
		val := int(i.Int64)
		return &val, nil
	}

	return nil, nil
}

func (s *SQLSessionStore) Remove(id, appId string) error {
	_, err := s.GetMaster().Exec(`delete from flow.session
where id = :Id
    and app_id = :AppId`, map[string]any{
		"Id":    id,
		"AppId": appId,
	})
	if err != nil {
		return model.NewAppError("SQLSessionStore.Remove", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s *SQLSessionStore) RemoveAll(appId string) error {
	_, err := s.GetMaster().Exec(`delete from flow.session
where  app_id = :AppId`, map[string]any{
		"AppId": appId,
	})
	if err != nil {
		return model.NewAppError("SQLSessionStore.RemoveAll", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}
