package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/lib/pq"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// lastErrorLimit matches the length check of flow.email_event.last_error.
const lastErrorLimit = 1024

type SqlEmailEventStore struct {
	SqlStore
}

func NewSqlEmailEventStore(sqlStore SqlStore) store.EmailEventStore {
	return &SqlEmailEventStore{sqlStore}
}

func (s *SqlEmailEventStore) Insert(ctx context.Context, e *model.EmailEvent) (bool, *model.AppError) {
	res, err := s.GetMaster().WithContext(ctx).Exec(`insert into flow.email_event (
    event_id, domain_id, event_type, schema_version, thread_id, message_id, flow_id, payload
)
values ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
on conflict (event_id) do nothing`,
		e.EventId, e.DomainId, e.EventType, e.SchemaVersion, e.ThreadId, e.MessageId, e.FlowId, string(e.Payload))

	affected, appErr := emailEventAffected("SqlEmailEventStore.Insert", res, err)

	return affected == 1, appErr
}

func (s *SqlEmailEventStore) RecoverExpired(ctx context.Context) (int64, *model.AppError) {
	// SKIP LOCKED: workers recovering at the same time do not wait for each other.
	res, err := s.GetMaster().WithContext(ctx).Exec(`update flow.email_event
set state = 'pending', locked_by = null, locked_until = null, updated_at = now()
where event_id in (
    select event_id from flow.email_event
    where state = 'processing' and locked_until < now()
    for update skip locked
)`)

	return emailEventAffected("SqlEmailEventStore.RecoverExpired", res, err)
}

func (s *SqlEmailEventStore) Claim(ctx context.Context, owner string, lease time.Duration) (*model.EmailEvent, *model.AppError) {
	var (
		e        model.EmailEvent
		flowId   sql.NullInt64
		payload  string
		lockedBy string
		until    time.Time
	)

	err := s.GetMaster().WithContext(ctx).QueryRow(`with due as (
    select event_id from flow.email_event
    where state = 'pending' and next_attempt_at <= now()
    order by next_attempt_at, created_at, event_id
    limit 1
    for update skip locked
)
update flow.email_event e
set state = 'processing', locked_by = $1,
    locked_until = now() + $2 * interval '1 millisecond', updated_at = now()
from due
where e.event_id = due.event_id
returning e.event_id, e.domain_id, e.event_type, e.schema_version, e.thread_id, e.message_id,
    e.flow_id, e.payload::text, e.attempts, e.next_attempt_at, e.locked_by, e.locked_until, e.created_at`,
		owner, lease.Milliseconds()).Scan(
		&e.EventId, &e.DomainId, &e.EventType, &e.SchemaVersion, &e.ThreadId, &e.MessageId,
		&flowId, &payload, &e.Attempts, &e.NextAttemptAt, &lockedBy, &until, &e.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, emailEventError("SqlEmailEventStore.Claim", err)
	}

	if flowId.Valid {
		e.FlowId = &flowId.Int64
	}
	e.Payload = []byte(payload)
	e.State = model.EmailEventProcessing
	e.LockedBy = lockedBy
	e.LockedUntil = &until

	return &e, nil
}

func (s *SqlEmailEventStore) MarkAccepted(ctx context.Context, owner, eventId string) (model.EmailEventAcceptResult, *model.AppError) {
	var (
		applied int
		state   sql.NullString
	)

	// Both parts read one snapshot: a row this update missed shows why.
	err := s.GetMaster().WithContext(ctx).QueryRow(`with upd as (
    update flow.email_event
    set state = 'accepted', accepted_at = now(), locked_by = null, locked_until = null,
        last_error = null, updated_at = now()
    where event_id = $1 and state = 'processing' and locked_by = $2
    returning 1
)
select (select count(*) from upd), (select state from flow.email_event where event_id = $1)`,
		eventId, owner).Scan(&applied, &state)
	if err != nil {
		return 0, emailEventError("SqlEmailEventStore.MarkAccepted", err)
	}

	switch {
	case applied == 1:
		return model.EmailEventApplied, nil
	case state.String == string(model.EmailEventAccepted):
		return model.EmailEventAlreadyAccepted, nil
	default:
		return model.EmailEventOwnershipLost, nil
	}
}

func (s *SqlEmailEventStore) Release(ctx context.Context, owner, eventId, reason string, next time.Time) (bool, *model.AppError) {
	res, err := s.GetMaster().WithContext(ctx).Exec(`update flow.email_event
set state = 'pending', attempts = attempts + 1, last_error = left($3, $5), next_attempt_at = $4,
    locked_by = null, locked_until = null, updated_at = now()
where event_id = $1 and state = 'processing' and locked_by = $2`,
		eventId, owner, reason, next, lastErrorLimit)

	affected, appErr := emailEventAffected("SqlEmailEventStore.Release", res, err)

	return affected == 1, appErr
}

func (s *SqlEmailEventStore) MarkFailed(ctx context.Context, owner, eventId, reason string) (bool, *model.AppError) {
	res, err := s.GetMaster().WithContext(ctx).Exec(`update flow.email_event
set state = 'failed', attempts = attempts + 1, last_error = left($3, $4),
    locked_by = null, locked_until = null, updated_at = now()
where event_id = $1 and state = 'processing' and locked_by = $2`,
		eventId, owner, reason, lastErrorLimit)

	affected, appErr := emailEventAffected("SqlEmailEventStore.MarkFailed", res, err)

	return affected == 1, appErr
}

func (s *SqlEmailEventStore) DeleteAcceptedBefore(ctx context.Context, before time.Time, limit int) (int64, *model.AppError) {
	// Batches with SKIP LOCKED keep cleanups of several instances from waiting on each other.
	res, err := s.GetMaster().WithContext(ctx).Exec(`delete from flow.email_event
where event_id in (
    select event_id from flow.email_event
    where state = 'accepted' and accepted_at < $1
    order by accepted_at, event_id
    limit $2
    for update skip locked
)`, before, limit)

	return emailEventAffected("SqlEmailEventStore.DeleteAcceptedBefore", res, err)
}

func emailEventAffected(where string, res sql.Result, err error) (int64, *model.AppError) {
	if err != nil {
		return 0, emailEventError(where, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, emailEventError(where, err)
	}

	return affected, nil
}

// emailEventError reports data and constraint errors as 400: retrying the same
// event cannot fix them.
func emailEventError(where string, err error) *model.AppError {
	code := extractCodeFromErr(err)

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && (pqErr.Code.Class() == "22" || pqErr.Code.Class() == "23") {
		code = http.StatusBadRequest
	}

	return model.NewAppError(where, "store.sql_email_event.app_error", nil, err.Error(), code)
}
