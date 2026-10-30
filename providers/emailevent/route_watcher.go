package emailevent

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
)

const (
	// routeJoinTimeout bounds one join of the watcher; a slower one is repeated later.
	routeJoinTimeout = 5 * time.Second
	// routeWriteTimeout bounds recording a route after its join, even during shutdown.
	routeWriteTimeout = 5 * time.Second
	routeRetryMin     = time.Second
)

// AttemptJoiner joins the Thread of an event into an Email queue of call_center.
type AttemptJoiner interface {
	EmailJoinToQueue(ctx context.Context, in *cc.EmailJoinToQueueRequest) (*cc.EmailJoinToQueueResponse, error)
}

// routeWatcher finishes the routes the Flow left, the fallback after the deadline and unfinished
// intents; it runs on every instance, and the lease of a claimed route keeps other watchers away.
type routeWatcher struct {
	*Server

	owner string
	log   *wlog.Logger
}

type routeCycle struct {
	claimed, routed, fallback, unrouted, failed, retried int
}

func newRouteWatcher(s *Server) *routeWatcher {
	owner := fmt.Sprintf("%s/%s/route", s.nodeId, s.runId)

	return &routeWatcher{
		Server: s,
		owner:  owner,
		log:    s.log.With(wlog.String("route_watcher_id", owner)),
	}
}

func (w *routeWatcher) run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.RouteWatcherInterval)
	defer ticker.Stop()

	for {
		w.cycle(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// cycle claims one batch; routes it cannot join before its lease ends go back at once.
func (w *routeWatcher) cycle(ctx context.Context) {
	start := time.Now()
	routes, appErr := w.store.ClaimRoutes(ctx, w.owner, w.cfg.RouteWatcherBatch, w.cfg.RouteLease)
	if appErr != nil {
		if ctx.Err() == nil {
			w.log.Error("email route claim failed", wlog.Err(appErr))
		}
		return
	}

	stats := routeCycle{claimed: len(routes)}
	for i, route := range routes {
		if ctx.Err() != nil {
			// The lease returns the rest.
			break
		}
		if time.Since(start)+routeJoinTimeout+routeWriteTimeout >= w.cfg.RouteLease {
			for _, rest := range routes[i:] {
				w.retry(ctx, rest, "lease ends before the join", time.Now(), false)
			}
			break
		}
		w.resolve(ctx, route, &stats)
	}

	if stats.claimed > 0 {
		w.log.Info("email route watcher cycle", wlog.Int("claimed", stats.claimed), wlog.Int("routed", stats.routed),
			wlog.Int("fallback", stats.fallback), wlog.Int("unrouted", stats.unrouted), wlog.Int("failed", stats.failed),
			wlog.Int("retried", stats.retried), wlog.Duration("duration", time.Since(start)))
	}
}

// routeJoinRequest repeats the stored JoinQueue values: the Flow intent while joining, the fallback
// defaults while resolving; a row written before them joins with the defaults. Both keep the initial From.
func routeJoinRequest(route *model.EmailRoute, queueId int32) *cc.EmailJoinToQueueRequest {
	req := &cc.EmailJoinToQueueRequest{
		QueueId:      queueId,
		ThreadId:     route.ThreadId,
		EmailEventId: route.EventId,
		DomainId:     route.DomainId,
		FromAddress:  route.FromAddress,
	}
	if route.Intent != nil {
		req.Priority = route.Intent.Priority
		req.StickyAgentId = route.Intent.StickyAgentId
		req.ExtraEmailCount = route.Intent.ExtraEmailCount
	}

	return req
}

// resolve joins the stored queue of the route: the intent of the Flow while joining,
// the fallback queue while resolving. It never chooses another queue.
func (w *routeWatcher) resolve(ctx context.Context, route *model.EmailRoute, stats *routeCycle) {
	log := w.log.With(
		wlog.String("email_event_id", route.EventId),
		wlog.Int64("domain_id", route.DomainId),
		wlog.Int64("email_thread_id", route.ThreadId),
		wlog.String("route_state", string(route.State)),
	)

	queueId, source := route.QueueId, model.EmailRouteSourceFlow
	if route.State == model.EmailRouteResolving {
		queueId, source = route.FallbackQueueId, model.EmailRouteSourceFallback
	}
	if queueId == nil {
		w.finish(ctx, route, model.EmailRouteResult{State: model.EmailRouteUnrouted, Reason: RouteReasonNoFallbackQueue}, stats, log)
		return
	}

	jctx, cancel := context.WithTimeout(ctx, routeJoinTimeout)
	res, err := w.joiner.EmailJoinToQueue(jctx, routeJoinRequest(route, *queueId))
	cancel()
	if err == nil && !validJoin(res) {
		err = fmt.Errorf("call_center returned an invalid email join response: %v", res)
	}

	switch {
	case err == nil:
		actual := res.GetQueueId()
		w.finish(ctx, route, model.EmailRouteResult{State: model.EmailRouteRouted, QueueId: &actual, Source: source}, stats,
			log.With(wlog.Int("queue_id", int(actual))))
	case JoinFailure(route.State, err) != nil:
		w.finish(ctx, route, *JoinFailure(route.State, err), stats, log.With(wlog.Err(err)))
	case !JoinTransient(err) && route.Attempts+1 >= int32(w.cfg.RouteMaxAttempts):
		w.finish(ctx, route, model.EmailRouteResult{State: model.EmailRouteFailed, Reason: RouteReasonUnexpectedError,
			LastError: err.Error()}, stats, log.With(wlog.Err(err)))
	default:
		// The join may have happened; only the same queue is tried again.
		if w.retry(ctx, route, err.Error(), time.Now().Add(w.retryDelay(route.Attempts+1)), true) {
			stats.retried++
		}
		log.Warn("email route join failed, retrying", wlog.Int("attempt", int(route.Attempts+1)), wlog.Err(err))
	}
}

// finish records the result; the counters follow only a committed write.
func (w *routeWatcher) finish(ctx context.Context, route *model.EmailRoute, res model.EmailRouteResult, stats *routeCycle, log *wlog.Logger) {
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), routeWriteTimeout)
	defer cancel()

	done, appErr := w.store.FinishRoute(wctx, w.owner, route.EventId, route.State, res)
	switch {
	case appErr != nil:
		log.Error("email route result was not recorded", wlog.Err(appErr))
		return
	case !done:
		log.Warn("email route lease lost")
		return
	}

	switch res.State {
	case model.EmailRouteRouted:
		stats.routed++
		if res.Source == model.EmailRouteSourceFallback {
			stats.fallback++
			RecordRouteOutcome(wctx, RouteOutcomeFallback)
			log.Info("email routed to the fallback queue")
		} else {
			log.Info("email routed to the queue of the flow")
		}
	case model.EmailRouteUnrouted:
		stats.unrouted++
		RecordRouteOutcome(wctx, RouteOutcomeUnrouted)
		log.Warn("email left without a queue", wlog.String("route_reason", res.Reason))
	case model.EmailRouteFailed:
		stats.failed++
		RecordRouteOutcome(wctx, RouteOutcomeFailed)
		log.Error("email route failed", wlog.String("route_reason", res.Reason))
	case model.EmailRoutePending:
		log.Warn("email route intent dropped; the fallback queue is used at the deadline")
	}
}

func (w *routeWatcher) retry(ctx context.Context, route *model.EmailRoute, reason string, next time.Time, counted bool) bool {
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), routeWriteTimeout)
	defer cancel()

	done, appErr := w.store.RetryRoute(wctx, w.owner, route.EventId, route.State, reason, next, counted)
	if appErr != nil {
		w.log.Error("email route was not given back", wlog.String("email_event_id", route.EventId), wlog.Err(appErr))
	}

	return appErr == nil && done
}

// retryDelay doubles from a second after each failed join, up to the configured maximum.
func (w *routeWatcher) retryDelay(attempts int32) time.Duration {
	return min(routeRetryMin<<min(max(attempts-1, 0), 16), w.cfg.RouteRetryMaxInterval)
}

// validJoin accepts only the known results with positive ids, as joinQueue does.
func validJoin(res *cc.EmailJoinToQueueResponse) bool {
	switch res.GetResult() {
	case cc.EmailJoinToQueueResponse_RESULT_CREATED, cc.EmailJoinToQueueResponse_RESULT_ALREADY_ACTIVE,
		cc.EmailJoinToQueueResponse_RESULT_REPLAYED:
		return res.GetAttemptId() > 0 && res.GetQueueId() > 0
	default:
		return false
	}
}
