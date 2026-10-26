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
      and coalesce(route_state, '') not in ('pending', 'joining', 'resolving')
    order by accepted_at, event_id
    limit $2
    for update skip locked
)`, before, limit)

	return emailEventAffected("SqlEmailEventStore.DeleteAcceptedBefore", res, err)
}

// routeColumns reads the route part of a row; the deadline is compared on the database clock.
const routeColumns = `event_id, domain_id, thread_id, coalesce(route_state, ''), coalesce(route_source, ''),
    route_queue_id, fallback_queue_id, coalesce(route_due_at <= now(), false), route_attempts,
    route_priority, route_sticky_agent_id, route_extra_email_count, coalesce(route_from_address, '')`

func (s *SqlEmailEventStore) InitRoute(ctx context.Context, owner, eventId string, needed bool, fallbackQueueId *int32, fromAddress string, due time.Duration) (*model.EmailRoute, *model.AppError) {
	// Both parts read one snapshot: the second shows a route written before.
	rows, err := s.GetMaster().WithContext(ctx).Query(`with upd as (
    update flow.email_event
    set route_state = case when $3 then 'pending' else 'not_needed' end,
        fallback_queue_id = case when $3 then $4::integer end,
        route_due_at = case when $3 then now() + $5 * interval '1 millisecond' end,
        route_next_at = case when $3 then now() + $5 * interval '1 millisecond' end,
        route_from_address = case when $3 then $6::text end,
        -- Fallback defaults until the Flow writes its intent: priority 0, no Sticky agent, no Extra.
        route_priority = case when $3 then 0 end,
        route_sticky_agent_id = case when $3 then 0 end,
        route_extra_email_count = case when $3 then false end,
        updated_at = now()
    where event_id = $1 and route_state is null and state = 'processing' and locked_by = $2
    returning `+routeColumns+`
)
select * from upd
union all
select `+routeColumns+` from flow.email_event where event_id = $1 and not exists (select 1 from upd)`,
		eventId, owner, needed, fallbackQueueId, due.Milliseconds(), fromAddress)
	if err != nil {
		return nil, emailEventError("SqlEmailEventStore.InitRoute", err)
	}

	routes, err := scanEmailRoutes(rows)
	if err != nil {
		return nil, emailEventError("SqlEmailEventStore.InitRoute", err)
	}
	if len(routes) == 0 {
		return &model.EmailRoute{EventId: eventId}, nil
	}

	return routes[0], nil
}

func (s *SqlEmailEventStore) BeginFlowRoute(ctx context.Context, owner, eventId string, intent model.EmailRouteIntent, lease time.Duration) (bool, *model.EmailRoute, *model.AppError) {
	// The deadline and the watcher claim are one condition each on the same row, so only
	// one of them can move a pending route.
	rows, err := s.GetMaster().WithContext(ctx).Query(`with upd as (
    update flow.email_event
    set route_state = 'joining', route_queue_id = $3, route_source = 'flow',
        -- The first intent stays; a repeated JoinQueue into the same queue does not change it.
        route_priority = case when route_state = 'pending' then $5::integer else route_priority end,
        route_sticky_agent_id = case when route_state = 'pending' then $6::integer else route_sticky_agent_id end,
        route_extra_email_count = case when route_state = 'pending' then $7::boolean else route_extra_email_count end,
        route_locked_by = $2, route_locked_until = now() + $4 * interval '1 millisecond',
        route_next_at = now(), updated_at = now()
    where event_id = $1
      and ((route_state = 'pending' and now() < route_due_at) or (route_state = 'joining' and route_queue_id = $3))
    returning true as applied, `+routeColumns+`
)
select * from upd
union all
select false, `+routeColumns+` from flow.email_event where event_id = $1 and not exists (select 1 from upd)`,
		eventId, owner, intent.QueueId, lease.Milliseconds(), intent.Priority, intent.StickyAgentId, intent.ExtraEmailCount)
	if err != nil {
		return false, nil, emailEventError("SqlEmailEventStore.BeginFlowRoute", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, nil, emailEventError("SqlEmailEventStore.BeginFlowRoute", err)
		}

		return false, nil, nil
	}

	var applied bool
	route, err := scanEmailRoute(rows, &applied)
	if err != nil {
		return false, nil, emailEventError("SqlEmailEventStore.BeginFlowRoute", err)
	}

	return applied, route, nil
}

func (s *SqlEmailEventStore) Route(ctx context.Context, eventId string) (*model.EmailRoute, *model.AppError) {
	rows, err := s.GetMaster().WithContext(ctx).Query(`select `+routeColumns+` from flow.email_event where event_id = $1`, eventId)
	if err != nil {
		return nil, emailEventError("SqlEmailEventStore.Route", err)
	}

	routes, err := scanEmailRoutes(rows)
	if err != nil {
		return nil, emailEventError("SqlEmailEventStore.Route", err)
	}
	if len(routes) == 0 {
		return nil, nil
	}

	return routes[0], nil
}

func (s *SqlEmailEventStore) ClaimRoutes(ctx context.Context, owner string, limit int, lease time.Duration) ([]*model.EmailRoute, *model.AppError) {
	// A pending route is taken only after its deadline; a leased one only once its lease ends.
	rows, err := s.GetMaster().WithContext(ctx).Query(`with due as (
    select event_id from flow.email_event
    where route_state in ('pending', 'joining', 'resolving') and route_next_at <= now()
      and (route_locked_until is null or route_locked_until < now())
      and (route_state <> 'pending' or route_due_at <= now())
    order by route_next_at
    limit $3
    for update skip locked
)
update flow.email_event e
set route_state = case when e.route_state = 'pending' then 'resolving' else e.route_state end,
    route_locked_by = $1, route_locked_until = now() + $2 * interval '1 millisecond', updated_at = now()
from due
where e.event_id = due.event_id
returning e.event_id, e.domain_id, e.thread_id, e.route_state, coalesce(e.route_source, ''),
    e.route_queue_id, e.fallback_queue_id, true, e.route_attempts,
    e.route_priority, e.route_sticky_agent_id, e.route_extra_email_count, coalesce(e.route_from_address, '')`,
		owner, lease.Milliseconds(), limit)
	if err != nil {
		return nil, emailEventError("SqlEmailEventStore.ClaimRoutes", err)
	}

	routes, err := scanEmailRoutes(rows)
	if err != nil {
		return nil, emailEventError("SqlEmailEventStore.ClaimRoutes", err)
	}

	return routes, nil
}

func (s *SqlEmailEventStore) FinishRoute(ctx context.Context, owner, eventId string, from model.EmailRouteState, res model.EmailRouteResult) (bool, *model.AppError) {
	// Back to pending resets the intent to the fallback defaults; the watcher takes it at the deadline.
	r, err := s.GetMaster().WithContext(ctx).Exec(`update flow.email_event
set route_state = $4,
    route_queue_id = case $4 when 'routed' then $5::integer when 'pending' then null else route_queue_id end,
    route_source = case $4 when 'routed' then $6 when 'pending' then null else route_source end,
    route_priority = case when $4 = 'pending' then 0 else route_priority end,
    route_sticky_agent_id = case when $4 = 'pending' then 0 else route_sticky_agent_id end,
    route_extra_email_count = case when $4 = 'pending' then false else route_extra_email_count end,
    route_reason = nullif($7, ''),
    route_last_error = left(nullif($8, ''), $9),
    route_next_at = case when $4 = 'pending' then route_due_at end,
    route_locked_by = null, route_locked_until = null, updated_at = now()
where event_id = $1 and route_state = $3 and route_locked_by = $2`,
		eventId, owner, string(from), string(res.State), res.QueueId, res.Source, res.Reason, res.LastError, lastErrorLimit)

	affected, appErr := emailEventAffected("SqlEmailEventStore.FinishRoute", r, err)

	return affected == 1, appErr
}

func (s *SqlEmailEventStore) RetryRoute(ctx context.Context, owner, eventId string, from model.EmailRouteState, reason string, next time.Time, counted bool) (bool, *model.AppError) {
	r, err := s.GetMaster().WithContext(ctx).Exec(`update flow.email_event
set route_attempts = route_attempts + case when $6 then 1 else 0 end,
    route_last_error = left($4, $7), route_next_at = $5,
    route_locked_by = null, route_locked_until = null, updated_at = now()
where event_id = $1 and route_state = $3 and route_locked_by = $2`,
		eventId, owner, string(from), reason, next, counted, lastErrorLimit)

	affected, appErr := emailEventAffected("SqlEmailEventStore.RetryRoute", r, err)

	return affected == 1, appErr
}

func scanEmailRoutes(rows *sql.Rows) ([]*model.EmailRoute, error) {
	defer rows.Close()

	var routes []*model.EmailRoute
	for rows.Next() {
		route, err := scanEmailRoute(rows)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}

	return routes, rows.Err()
}

// scanEmailRoute reads routeColumns, after the leading columns given.
func scanEmailRoute(rows *sql.Rows, lead ...any) (*model.EmailRoute, error) {
	var (
		r                 model.EmailRoute
		state             string
		queueId, fallback sql.NullInt32
		priority, sticky  sql.NullInt32
		extra             sql.NullBool
	)
	dest := append(lead, &r.EventId, &r.DomainId, &r.ThreadId, &state, &r.Source, &queueId, &fallback, &r.DuePassed, &r.Attempts,
		&priority, &sticky, &extra, &r.FromAddress)
	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}

	r.State = model.EmailRouteState(state)
	if queueId.Valid {
		r.QueueId = &queueId.Int32
	}
	if fallback.Valid {
		r.FallbackQueueId = &fallback.Int32
	}
	if priority.Valid && sticky.Valid && extra.Valid {
		r.Intent = &model.EmailRouteIntent{QueueId: queueId.Int32, Priority: priority.Int32, StickyAgentId: sticky.Int32,
			ExtraEmailCount: extra.Bool}
	}

	return &r, nil
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
