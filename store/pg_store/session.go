package sqlstore

import (
	"time"

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

func (s *SQLSessionStore) Touch(id, appID string) (*int, error) {
	i, err := s.GetMaster().SelectNullInt(`insert into flow.session(id, app_id)
values (:Id, :AppId)
on conflict (id)
    DO UPDATE SET seq = session.seq + 1,
                  updated_at = now()
    where session.app_id=:AppId
returning session.seq`, map[string]any{
		"Id":    id,
		"AppId": appID,
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

func (s *SQLSessionStore) Remove(id, appID string) error {
	_, err := s.GetMaster().Exec(`delete from flow.session
where id = :Id
    and app_id = :AppId`, map[string]any{
		"Id":    id,
		"AppId": appID,
	})
	if err != nil {
		return model.NewAppError("SQLSessionStore.Remove", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

// consumedKeyInfix names the rows that keep how far a finished dialog read its thread, one per
// event kind: "<thread>.<sub>:consumed:<kind>". They share the "<thread>.<sub>" prefix with the
// session claim, so RemoveByThread drops them too, and never collide with a claim id, which
// ends in the numeric bot sub. On such a row app_id is null, updated_at holds the newest event
// time of that kind and created_at the database time of the last Release, which ages the row
// for PruneConsumed.
const consumedKeyInfix = ":consumed:"

func consumedKey(id, kind string) string { return id + consumedKeyInfix + kind }

// Release ends a session that finished naturally. In one statement it records consumedUpTo
// (the newest event time per kind, Unix ms, the dialog observed) and drops the claim. Every
// node receives every IM event, so a node that lags behind can see the claim gone while it
// still holds an event this dialog already observed; the record lets it tell that event apart
// from new input. Writing both in one statement guarantees that whoever wins the freed claim
// also sees the record. A record never moves back (greatest), and it is written even when
// appID no longer holds the claim: it only holds back events that were already received.
func (s *SQLSessionStore) Release(id, appID string, consumedUpTo model.IMConsumedUpTo) error {
	_, err := s.GetMaster().Exec(`with consumed as (
    insert into flow.session (id, app_id, created_at, updated_at)
    select v.id, null, now(), to_timestamp(v.up_to / 1000.0)
    from (values (:MessagesId, :Messages::bigint), (:CallbacksId, :Callbacks::bigint)) as v (id, up_to)
    where v.up_to > 0
    on conflict (id) do update
        set updated_at = greatest(session.updated_at, excluded.updated_at),
            created_at = excluded.created_at
)
delete from flow.session
where id = :Id
    and app_id = :AppId`, map[string]any{
		"MessagesId":  consumedKey(id, model.IMEventTypeMessage),
		"Messages":    consumedUpTo.Messages,
		"CallbacksId": consumedKey(id, model.IMEventTypeCallback),
		"Callbacks":   consumedUpTo.Callbacks,
		"Id":          id,
		"AppId":       appID,
	})
	if err != nil {
		return model.NewAppError("SQLSessionStore.Release", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

// Consumed reports whether an event of the given kind and time (Unix ms) was already observed
// by a dialog of this session that has since finished.
func (s *SQLSessionStore) Consumed(id, kind string, at int64) (bool, error) {
	n, err := s.GetMaster().SelectInt(`select count(*)
from flow.session
where id = :ConsumedId
    and updated_at >= to_timestamp(:At::bigint / 1000.0)`, map[string]any{
		"ConsumedId": consumedKey(id, kind),
		"At":         at,
	})
	if err != nil {
		return false, model.NewAppError("SQLSessionStore.Consumed", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return n > 0, nil
}

// PruneConsumed drops consumption records last released more than olderThan ago. A record
// only has to outlive the delivery lag between nodes, so this is garbage collection, not part
// of the check.
func (s *SQLSessionStore) PruneConsumed(olderThan time.Duration) error {
	_, err := s.GetMaster().Exec(`delete from flow.session
where id like :Pattern
    and app_id is null
    and created_at < now() - make_interval(secs => :Seconds::bigint)`, map[string]any{
		"Pattern": "%" + consumedKeyInfix + "%",
		"Seconds": int64(olderThan / time.Second),
	})
	if err != nil {
		return model.NewAppError("SQLSessionStore.PruneConsumed", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s *SQLSessionStore) RemoveByThread(threadID string) error {
	if threadID == "" {
		return nil
	}

	_, err := s.GetMaster().Exec(`delete from flow.session
where id like :Prefix`, map[string]any{
		"Prefix": threadID + ".%",
	})
	if err != nil {
		return model.NewAppError("SQLSessionStore.RemoveByThread", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}

func (s *SQLSessionStore) RemoveAll(appID string) error {
	_, err := s.GetMaster().Exec(`delete from flow.session
where  app_id = :AppId`, map[string]any{
		"AppId": appID,
	})
	if err != nil {
		return model.NewAppError("SQLSessionStore.RemoveAll", "store.sql_session.app_error", nil, err.Error(), extractCodeFromErr(err))
	}

	return nil
}
