// Package sessionmgr owns the lifecycle of a suspended runtime session: it
// keeps the connection alive while the flow waits for an external event,
// routes inbound messages to the runtime through the Coordinator, schedules
// an in-process wake-up timer for recv_message deadlines, replays an initial
// message when recovering from a process restart, and calls the channel-
// supplied teardown exactly once when the flow ends or the connection drops.
//
// The manager is channel-agnostic. Channel routers supply:
//   - Connection: the transport handle providing Id, Context, OnInboundMessage
//   - *persistence.Record: the suspended record (used to read pending intent)
//   - initialMsg: optional message to dispatch immediately after registration
//   - ContextDecorator: optional, injects channel-specific context values
//   - TeardownFunc: channel-specific cleanup invoked exactly once
package sessionmgr

import (
	"context"
	"sync"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/state"
)

// Connection is the minimal capability the session manager needs from a
// channel transport. *im.Connection satisfies it (via Id, Context, and
// OnInboundMessage); other channels can satisfy it too.
type Connection interface {
	Id() string
	Context() context.Context
	OnInboundMessage(handler func(text string)) (unregister func())
}

// Dispatcher resumes a suspended flow by routing payload through to the
// runtime. *coordinator.Coordinator satisfies it.
type Dispatcher interface {
	Dispatch(ctx context.Context, resumeKey string, payload map[string]string) error
}

// StateLoader returns the current Record for a connection (running, suspended,
// or nil). Used after a dispatch to detect whether the flow is still active.
type StateLoader interface {
	LoadByConnectionID(ctx context.Context, connectionID string) (*persistence.Record, error)
}

// ContextDecorator injects channel-specific values into the dispatch context
// (e.g. connection metadata for legacy ops). Called on every Dispatch with the
// connection context. May be nil — context is then used as-is.
type ContextDecorator func(ctx context.Context) context.Context

// TeardownFunc finalises channel-specific cleanup when the suspended session
// ends. It is invoked exactly once.
type TeardownFunc func()

// Manager owns the watch goroutines for suspended sessions. Construct once
// per channel router and call Watch for every suspended record.
type Manager struct {
	coord Dispatcher
	repo  StateLoader
	log   *wlog.Logger
}

// New constructs a Manager. log must be non-nil.
func New(coord Dispatcher, repo StateLoader, log *wlog.Logger) *Manager {
	return &Manager{coord: coord, repo: repo, log: log}
}

// Watch starts watching a suspended runtime session for the given connection.
// It returns immediately; the watching goroutines run in background.
//
//   - rec is the suspended record. rec.State.Pending is consulted to derive
//     the resume key and to schedule the recv_message wake_at timer.
//   - initialMsg, when non-empty, is dispatched in a goroutine immediately
//     after the inbound handler is registered. Used by the recovery path
//     where the message that triggered the resume arrived before the handler
//     was attached.
//   - decorator may be nil.
//   - teardown may be nil; when non-nil it is called exactly once.
func (m *Manager) Watch(
	conn Connection,
	rec *persistence.Record,
	initialMsg string,
	decorator ContextDecorator,
	teardown TeardownFunc,
) {
	var (
		once    sync.Once
		unregFn func()
	)
	done := make(chan struct{})

	teardownOnce := func() {
		once.Do(func() {
			close(done)
			if unregFn != nil {
				unregFn()
			}
			if teardown != nil {
				teardown()
			}
		})
	}

	connID := conn.Id()
	suspendKey := resolveSuspendKey(rec, connID)

	decorate := decorator
	if decorate == nil {
		decorate = func(ctx context.Context) context.Context { return ctx }
	}

	dispatch := func(payload map[string]string) {
		ctx := decorate(conn.Context())
		if err := m.coord.Dispatch(ctx, suspendKey, payload); err != nil {
			m.log.Warn("sessionmgr dispatch error",
				wlog.String("conn", connID),
				wlog.Err(err))
		}

		latest, loadErr := m.repo.LoadByConnectionID(conn.Context(), connID)
		if loadErr != nil {
			m.log.Warn("sessionmgr reload after dispatch failed",
				wlog.String("conn", connID),
				wlog.Err(loadErr))
			teardownOnce()
			return
		}
		if latest == nil || (latest.Status != state.StatusSuspended && latest.Status != state.StatusRunning) {
			teardownOnce()
		}
	}

	unregFn = conn.OnInboundMessage(func(text string) {
		dispatch(map[string]string{"msg": text})
	})

	// In-process wake_at timer for recv_message. Provides accurate short
	// timeouts and survives without the DB-polling worker. The atomic claim
	// inside LoadByResumeKey guarantees concurrent dispatches are no-ops.
	//
	// When initialMsg is set the user already responded before the timer
	// fires — dispatch the message and skip the timer entirely. An expired
	// timer racing with an initial-message dispatch would cause the timeout
	// to win non-deterministically and silently drop the user's reply.
	if initialMsg != "" {
		go dispatch(map[string]string{"msg": initialMsg})
	} else if rec != nil && rec.State.Pending != nil && rec.State.Pending.OpName == "recv_message" {
		if wakeAtStr, ok := rec.State.Pending.Args["wake_at"]; ok {
			if wakeAt, err := time.Parse(time.RFC3339, wakeAtStr); err == nil {
				delay := time.Until(wakeAt)
				if delay <= 0 {
					go dispatch(map[string]string{"timeout": "true"})
				} else {
					time.AfterFunc(delay, func() {
						dispatch(map[string]string{"timeout": "true"})
					})
				}
			}
		}
	}

	go func() {
		select {
		case <-conn.Context().Done():
		case <-done:
		}
		teardownOnce()
	}()
}

// resolveSuspendKey returns the resume key for the suspended record, falling
// back to "msg:<connID>" when pending intent does not carry one.
func resolveSuspendKey(rec *persistence.Record, connID string) string {
	if rec != nil && rec.State.Pending != nil && rec.State.Pending.ResumeKey != "" {
		return rec.State.Pending.ResumeKey
	}
	return "msg:" + connID
}
