package emailevent

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/webitel/wlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
)

const (
	// missingSchemaAttempts fails an event whose schema is still missing after
	// this many launches; earlier ones may hit a lagging replica.
	missingSchemaAttempts = 5
	maxRetryDelay         = 30 * time.Second
)

type launchOutcome int

const (
	outcomeStarted launchOutcome = iota
	outcomeFailed
	outcomeStopped
)

// worker claims Inbox rows one at a time and starts their Flows.
type worker struct {
	*Server

	owner      string
	log        *wlog.Logger
	minBackoff time.Duration
	maxBackoff time.Duration
}

func newWorker(s *Server, index int) *worker {
	// fm.id is shared by every worker and by restarts, so it cannot own a lease alone.
	owner := fmt.Sprintf("%s/%s/%d", s.nodeId, s.runId, index)

	return &worker{
		Server:     s,
		owner:      owner,
		log:        s.log.With(wlog.String("worker_id", owner)),
		minBackoff: retryInitialInterval,
		maxBackoff: retryMaxInterval,
	}
}

func (w *worker) run(ctx context.Context) {
	var backoff time.Duration

	for ctx.Err() == nil {
		row, appErr := w.store.Claim(ctx, w.owner, leaseDuration)
		switch {
		case appErr != nil:
			if ctx.Err() != nil {
				return
			}
			backoff = nextBackoff(backoff, w.minBackoff, w.maxBackoff)
			w.log.Warn("email event claim failed", wlog.Duration("retry_in", backoff), wlog.Err(appErr))
			sleep(ctx, backoff)
		case row == nil:
			backoff = 0
			sleep(ctx, pollInterval)
		default:
			backoff = 0
			w.process(ctx, row)
		}
	}
}

func (w *worker) process(ctx context.Context, row *model.EmailEvent) {
	log := w.log.With(
		wlog.String("email_event_id", row.EventId),
		wlog.Int64("domain_id", row.DomainId),
		wlog.Int64("email_thread_id", row.ThreadId),
		wlog.Int64("email_message_id", row.MessageId),
		wlog.Int("attempts", int(row.Attempts)),
	)

	// The consumer validated it already; a mismatch here means the contract moved.
	event, err := Decode(row.Payload)
	if err != nil {
		w.fail(ctx, row, err.Error(), log)
		return
	}

	wakeup, fallback, appErr := w.wakeUp(ctx, event)
	if appErr != nil {
		switch {
		case ctx.Err() != nil:
			// The lease returns the row; a shutdown does not spend an attempt.
		case appErr.StatusCode == http.StatusConflict || appErr.StatusCode == http.StatusBadRequest:
			w.fail(ctx, row, appErr.Error(), log)
		default:
			w.retry(ctx, row, appErr, "email attempt wake-up failed", log)
		}
		return
	}
	if wakeup == wakeUpNoop || wakeup == wakeUpNotActive {
		log.Debug("email attempt wake-up", wlog.String("result", wakeup))
	} else {
		log.Info("email attempt wake-up", wlog.String("result", wakeup))
	}

	// Without a Flow the deadline is now, and the route watcher joins the fallback queue.
	due := w.cfg.RouteTimeout
	if event.FlowId == nil {
		due = 0
	}
	route, appErr := w.store.InitRoute(ctx, w.owner, row.EventId, wakeup == wakeUpNotActive, fallback, RoutingFromAddress(event), due)
	if appErr != nil {
		switch {
		case ctx.Err() != nil:
			// The lease returns the row.
		case appErr.StatusCode == http.StatusBadRequest:
			w.fail(ctx, row, appErr.Error(), log)
		default:
			w.retry(ctx, row, appErr, "email route start failed", log)
		}
		return
	}
	if route.State == model.EmailRouteUnset {
		log.Warn("email event lease lost")
		return
	}

	if event.FlowId == nil {
		if !w.accept(ctx, row, nil, log) {
			return
		}
		// A Thread with an attempt needs no route: the wake-up handled the email.
		if route.State != model.EmailRouteNotNeeded {
			log.Info("email thread has no flow; the fallback queue routes it", wlog.String("route_state", string(route.State)))
		} else {
			log.Info("email event accepted without a flow", wlog.String("result", wakeup))
		}
		return
	}

	if !launchFlow(route) {
		// The route is decided or past the deadline; a Flow could not change it.
		if w.accept(ctx, row, nil, log) {
			log.Info("email flow not started again", wlog.String("route_state", string(route.State)))
		}
		return
	}

	conn := NewConnection(w.ctx, w.nodeId, event, w.Server.log)
	conn.Set(ctx, model.Variables{"cc_wakeup_result": wakeup, "cc_fallback_queue_id": queueVariable(route.FallbackQueueId)})
	switch outcome, launchErr := w.launch(ctx, conn); outcome {
	case outcomeStarted:
		w.accept(ctx, row, conn, log)
	case outcomeFailed:
		w.retry(ctx, row, launchErr, "email flow launch failed", log)
	case outcomeStopped:
		// The lease returns the row; a shutdown does not spend an attempt.
	}
}

const (
	wakeUpNoop      = "noop"
	wakeUpNotActive = "not_active"
)

var wakeUpResults = map[cc.EmailWakeUpResponse_Result]string{
	cc.EmailWakeUpResponse_RESULT_WOKEN:      "woken",
	cc.EmailWakeUpResponse_RESULT_QUEUED:     "queued",
	cc.EmailWakeUpResponse_RESULT_REOPENED:   "reopened",
	cc.EmailWakeUpResponse_RESULT_NOOP:       wakeUpNoop,
	cc.EmailWakeUpResponse_RESULT_NOT_ACTIVE: wakeUpNotActive,
}

// launchFlow starts a Flow for a Thread with an attempt, or while the Flow may still
// choose the queue; a retried event never extends the deadline.
func launchFlow(route *model.EmailRoute) bool {
	return route.State == model.EmailRouteNotNeeded || (route.State == model.EmailRoutePending && !route.DuePassed)
}

// queueVariable is empty without a queue, so the Flow never reads a stale value.
func queueVariable(queueId *int32) string {
	if queueId == nil {
		return ""
	}

	return strconv.Itoa(int(*queueId))
}

// wakeUp runs before the Flow and without one; call_center keeps the result and the
// fallback queue by event id, so a retried event does not wake the attempt again.
func (w *worker) wakeUp(ctx context.Context, event *InboundMessage) (string, *int32, *model.AppError) {
	ctx, cancel := context.WithTimeout(ctx, wakeUpTimeout)
	defer cancel()

	res, err := w.waker.EmailWakeUp(ctx, &cc.EmailWakeUpRequest{
		DomainId:     event.DomainId,
		ThreadId:     event.ThreadId,
		EmailEventId: event.EventId,
	})
	if err != nil {
		return "", nil, wakeUpError(err)
	}

	result, ok := wakeUpResults[res.GetResult()]
	if !ok {
		return "", nil, model.NewAppError("EmailEvents.WakeUp", "email_events.wakeup.invalid_response", nil,
			"call_center returned an unknown wake-up result "+res.GetResult().String(), http.StatusBadGateway)
	}

	return result, res.FallbackQueueId, nil
}

// wakeUpError keeps 400 and 409 for errors a retry cannot fix; the worker fails those events.
func wakeUpError(err error) *model.AppError {
	id, code := "app_error", http.StatusInternalServerError
	switch status.Code(err) {
	case codes.InvalidArgument:
		id, code = "invalid", http.StatusBadRequest
	case codes.AlreadyExists:
		id, code = "event_conflict", http.StatusConflict
	case codes.Unavailable, codes.DeadlineExceeded:
		id, code = "unavailable", http.StatusServiceUnavailable
	}

	return model.NewAppError("EmailEvents.WakeUp", "email_events.wakeup."+id, nil, err.Error(), code)
}

// launch hands the connection to the email router and waits for the Flow to
// start; one handshake timer covers both steps.
func (w *worker) launch(ctx context.Context, conn *Connection) (launchOutcome, *model.AppError) {
	timer := time.NewTimer(handshakeTimeout)
	defer timer.Stop()

	timeout := model.NewAppError("EmailEvents.Launch", "email_events.launch.timeout", nil,
		"email router did not start the flow in time", http.StatusGatewayTimeout)

	select {
	case w.conns <- conn:
	case <-timer.C:
		conn.Close()
		return outcomeFailed, timeout
	case <-ctx.Done():
		conn.Close()
		return outcomeStopped, nil
	}

	select {
	case appErr := <-conn.LaunchResult():
		if appErr != nil {
			conn.Close()
			return outcomeFailed, appErr
		}
		return outcomeStarted, nil
	case <-timer.C:
		// The router may have started the Flow just now; then it counts.
		if !conn.Abandon() {
			return outcomeStarted, nil
		}
		conn.Close()
		return outcomeFailed, timeout
	case <-ctx.Done():
		conn.Close()
		return outcomeStopped, nil
	}
}

// accept records a started (or routeless) event, retrying database errors past
// the lease; only a lost lease cancels the Flow. conn is nil without a Flow.
func (w *worker) accept(ctx context.Context, row *model.EmailEvent, conn *Connection, log *wlog.Logger) bool {
	var backoff time.Duration

	for {
		result, appErr := w.store.MarkAccepted(ctx, w.owner, row.EventId)
		if appErr == nil {
			if result == model.EmailEventOwnershipLost {
				if conn != nil {
					conn.Close()
				}
				log.Warn("email event lease lost before it was accepted; its flow is cancelled")
				return false
			}

			log.Info("email event accepted", wlog.String("result", result.String()))
			return true
		}
		if ctx.Err() != nil {
			return false
		}

		backoff = nextBackoff(backoff, w.minBackoff, w.maxBackoff)
		log.Warn("email event accept failed", wlog.Duration("retry_in", backoff), wlog.Err(appErr))
		if !sleep(ctx, backoff) {
			return false
		}
	}
}

// retry returns the row to pending after a failed wake-up or launch, or fails it once a
// missing schema outlasts replica lag.
func (w *worker) retry(ctx context.Context, row *model.EmailEvent, launchErr *model.AppError, msg string, log *wlog.Logger) {
	attempts := row.Attempts + 1
	if launchErr.StatusCode == http.StatusNotFound && attempts >= missingSchemaAttempts {
		w.fail(ctx, row, launchErr.Error(), log)
		return
	}

	next := time.Now().Add(retryDelay(attempts))
	released, appErr := w.store.Release(ctx, w.owner, row.EventId, launchErr.Error(), next)
	if !w.settled(released, appErr, log) {
		return
	}

	log.Warn(msg, wlog.Int("attempt", int(attempts)),
		wlog.String("retry_at", next.Format(time.RFC3339)), wlog.Err(launchErr))
}

func (w *worker) fail(ctx context.Context, row *model.EmailEvent, reason string, log *wlog.Logger) {
	failed, appErr := w.store.MarkFailed(ctx, w.owner, row.EventId, reason)
	if !w.settled(failed, appErr, log) {
		return
	}

	log.Error("email event failed", wlog.String("reason", reason))
}

// settled logs why a guarded write did not apply; no Flow runs at that point,
// and the lease returns a row the write could not reach.
func (w *worker) settled(applied bool, appErr *model.AppError, log *wlog.Logger) bool {
	switch {
	case appErr != nil:
		log.Error("email event update failed", wlog.Err(appErr))
	case !applied:
		log.Warn("email event lease lost")
	}

	return appErr == nil && applied
}

// retryDelay is 1, 2, 4… seconds after the given failed attempt, capped.
func retryDelay(attempts int32) time.Duration {
	return min(time.Second<<min(max(attempts-1, 0), 5), maxRetryDelay)
}
