package email

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/webitel/wlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/emailevent"
)

type QueueParameters struct {
	Queue struct {
		Id   int32  `json:"id"`
		Name string `json:"name"`
	} `json:"queue"`
	Priority any `json:"priority"`
	Agent    *struct {
		Id        any     `json:"id"`
		Extension *string `json:"extension"`
	} `json:"agent"`
	ExtraEmailCount bool `json:"extraEmailCount"`
}

type emailQueueJoiner interface {
	EmailJoinToQueue(ctx context.Context, in *cc.EmailJoinToQueueRequest) (*cc.EmailJoinToQueueResponse, error)
}

type emailQueueFinder interface {
	FindQueueByName(domainId int64, name string) (int32, *model.AppError)
}

type emailQueueService interface {
	emailQueueJoiner
	emailQueueFinder
	GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError)
}

// joinRetry bounds retries of transient call_center errors; the join is idempotent
// by email_event_id, so a retry after a lost reply is safe.
type joinRetry struct {
	attempts    int
	callTimeout time.Duration
	budget      time.Duration
	baseDelay   time.Duration
	maxDelay    time.Duration
}

var defaultJoinRetry = joinRetry{
	attempts:    5,
	callTimeout: 5 * time.Second,
	budget:      25 * time.Second,
	baseDelay:   500 * time.Millisecond,
	maxDelay:    8 * time.Second,
}

var joinResults = map[cc.EmailJoinToQueueResponse_Result]string{
	cc.EmailJoinToQueueResponse_RESULT_CREATED:        "created",
	cc.EmailJoinToQueueResponse_RESULT_ALREADY_ACTIVE: "already_active",
	cc.EmailJoinToQueueResponse_RESULT_REPLAYED:       "replayed",
}

func (r *Router) joinQueue(ctx context.Context, scope *flow.Flow, conn model.EmailConnection, args interface{}) (model.Response, *model.AppError) {
	event, ok := conn.(model.EmailEventConnection)
	if !ok {
		return nil, model.NewAppError("Email.JoinQueue", "email.join_queue.not_supported", nil,
			"joinQueue needs a connection started by an email-service event", http.StatusBadRequest)
	}

	var argv QueueParameters
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	routes := r.fm.Store.EmailEvent()
	req, appErr := storedFlowRequest(ctx, routes, r.fm, event, argv)
	if appErr != nil {
		return nil, appErr
	}
	if req == nil {
		if req, appErr = emailJoinRequest(r.fm, event, argv); appErr != nil {
			return nil, appErr
		}
	}
	if handled, res, err := joinEmailRoute(ctx, routes, r.fm, event, req, defaultJoinRetry); handled {
		return res, err
	}

	return joinWithRequest(ctx, r.fm, event, req, defaultJoinRetry)
}

// Keep validation before narrowing the generic Flow decoder's numeric values.
func emailQueueNumber(value any, maximum int32, field string) (int32, *model.AppError) {
	if value == nil {
		return 0, nil
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > float64(maximum) || n != math.Trunc(n) {
		return 0, model.NewAppError("Email.JoinQueue", "email.join_queue.valid."+field, nil,
			fmt.Sprintf("%s must be an integer between 0 and %d", field, maximum), http.StatusBadRequest)
	}
	return int32(n), nil
}

// joinEmailQueue freezes routing values once before any transport retry.
func joinEmailQueue(ctx context.Context, cli emailQueueService, conn model.EmailEventConnection, argv QueueParameters, rt joinRetry) (model.Response, *model.AppError) {
	req, appErr := emailJoinRequest(cli, conn, argv)
	if appErr != nil {
		return nil, appErr
	}

	return joinWithRequest(ctx, cli, conn, req, rt)
}

// emailJoinRequest validates the arguments and resolves the queue and the Sticky agent once.
func emailJoinRequest(cli emailQueueService, conn model.EmailEventConnection, argv QueueParameters) (*cc.EmailJoinToQueueRequest, *model.AppError) {
	priority, appErr := emailQueueNumber(argv.Priority, math.MaxInt32, "priority")
	if appErr != nil {
		return nil, appErr
	}
	queueId := argv.Queue.Id
	if queueId == 0 && argv.Queue.Name != "" {
		var err *model.AppError
		if queueId, err = cli.FindQueueByName(conn.DomainId(), argv.Queue.Name); err != nil {
			return nil, err
		}
	}
	if queueId <= 0 {
		return nil, queueRequiredError()
	}
	var stickyAgentId int32
	if argv.Agent != nil {
		if argv.Agent.Id != nil {
			stickyAgentId, appErr = emailQueueNumber(argv.Agent.Id, math.MaxInt32, "agent")
			if appErr != nil {
				return nil, appErr
			}
		} else if argv.Agent.Extension != nil && strings.TrimSpace(*argv.Agent.Extension) != "" {
			id, err := cli.GetAgentIdByExtension(conn.DomainId(), strings.TrimSpace(*argv.Agent.Extension))
			if err != nil {
				return nil, err
			}
			if id != nil {
				stickyAgentId = *id
			}
		}
	}

	req := &cc.EmailJoinToQueueRequest{
		QueueId:         queueId,
		ThreadId:        conn.ThreadId(),
		EmailEventId:    conn.EventId(),
		DomainId:        conn.DomainId(),
		Priority:        priority,
		StickyAgentId:   stickyAgentId,
		ExtraEmailCount: argv.ExtraEmailCount,
		FromAddress:     model.NormalizeEmailRoutingAddress(conn.InitialFromAddress()),
	}

	return req, nil
}

// joinWithRequest joins with a frozen request and stores the cc_* variables.
func joinWithRequest(ctx context.Context, cli emailQueueJoiner, conn model.EmailEventConnection, req *cc.EmailJoinToQueueRequest, rt joinRetry) (model.Response, *model.AppError) {
	res, err := callEmailJoin(ctx, cli, req, rt, conn.Log())
	if err != nil {
		return nil, joinError(err)
	}

	return setJoinVariables(ctx, conn, res)
}

// storedFlowRequest settles a repeated JoinQueue by the stored route before any agent lookup;
// nil, nil takes the usual path.
func storedFlowRequest(ctx context.Context, routes routeStore, queues emailQueueFinder, conn model.EmailEventConnection, argv QueueParameters) (*cc.EmailJoinToQueueRequest, *model.AppError) {
	route, appErr := routes.Route(ctx, conn.EventId())
	if appErr != nil {
		return nil, model.NewAppError("Email.JoinQueue", "email_route.unavailable", nil, appErr.Error(), http.StatusServiceUnavailable)
	}
	flowRoute := route != nil && route.Intent != nil && route.Source == model.EmailRouteSourceFlow &&
		(route.State == model.EmailRouteJoining || route.State == model.EmailRouteRouted)
	if !flowRoute && !routeWindowClosed(route) {
		return nil, nil
	}
	// Changed values are ignored only when valid; an invalid one fails as on the first call.
	if _, appErr = emailQueueNumber(argv.Priority, math.MaxInt32, "priority"); appErr != nil {
		return nil, appErr
	}
	if argv.Agent != nil && argv.Agent.Id != nil {
		if _, appErr = emailQueueNumber(argv.Agent.Id, math.MaxInt32, "agent"); appErr != nil {
			return nil, appErr
		}
	}
	queueId := argv.Queue.Id
	if queueId < 0 || (queueId == 0 && argv.Queue.Name == "") {
		return nil, queueRequiredError()
	}
	if !flowRoute {
		emailevent.RecordRouteOutcome(ctx, emailevent.RouteOutcomeWindowClosed)
		conn.Log().Warn("email route window is closed", wlog.String("route_state", string(route.State)))
		return nil, windowClosedError()
	}
	if route.State == model.EmailRouteJoining {
		if queueId == 0 {
			if queueId, appErr = queues.FindQueueByName(conn.DomainId(), argv.Queue.Name); appErr != nil {
				return nil, appErr
			}
		}
		switch {
		case queueId <= 0:
			return nil, queueRequiredError()
		case queueId != route.Intent.QueueId:
			conn.Log().Warn("email route already joins another queue", wlog.Int("route_queue_id", int(route.Intent.QueueId)))
			return nil, queueConflictError(route.Intent.QueueId)
		}
	}

	in := route.Intent
	return &cc.EmailJoinToQueueRequest{
		QueueId:         in.QueueId,
		ThreadId:        conn.ThreadId(),
		EmailEventId:    conn.EventId(),
		DomainId:        conn.DomainId(),
		Priority:        in.Priority,
		StickyAgentId:   in.StickyAgentId,
		ExtraEmailCount: in.ExtraEmailCount,
		FromAddress:     model.NormalizeEmailRoutingAddress(conn.InitialFromAddress()),
	}, nil
}

// routeWindowClosed reports a route the Flow can no longer choose a queue for; none of these
// states goes back to pending or joining.
func routeWindowClosed(route *model.EmailRoute) bool {
	if route == nil {
		return false
	}
	switch route.State {
	case model.EmailRoutePending:
		return route.DuePassed
	case model.EmailRouteResolving, model.EmailRouteUnrouted, model.EmailRouteFailed:
		return true
	case model.EmailRouteRouted:
		return route.Source == model.EmailRouteSourceFallback
	}
	return false
}

func queueRequiredError() *model.AppError {
	return model.NewAppError("Email.JoinQueue", "email.join_queue.valid.queue", nil,
		"queue id or name is required", http.StatusBadRequest)
}

func queueConflictError(queueId int32) *model.AppError {
	return model.NewAppError("Email.JoinQueue", "email_route.queue_conflict", nil,
		fmt.Sprintf("the email is already joining queue %d", queueId), http.StatusConflict)
}

func windowClosedError() *model.AppError {
	return model.NewAppError("Email.JoinQueue", "email_route.window_closed", nil,
		"the email is already routed or its time to choose a queue is over", http.StatusConflict)
}

// routeStore is the part of the Inbox that keeps the route of an event.
type routeStore interface {
	Route(ctx context.Context, eventId string) (*model.EmailRoute, *model.AppError)
	BeginFlowRoute(ctx context.Context, owner, eventId string, intent model.EmailRouteIntent, lease time.Duration) (bool, *model.EmailRoute, *model.AppError)
	FinishRoute(ctx context.Context, owner, eventId string, from model.EmailRouteState, res model.EmailRouteResult) (bool, *model.AppError)
	RetryRoute(ctx context.Context, owner, eventId string, from model.EmailRouteState, reason string, next time.Time, counted bool) (bool, *model.AppError)
}

// routeWriteTimeout bounds a route write after the join, even when the Flow is gone.
const routeWriteTimeout = 5 * time.Second

// joinEmailRoute writes the Flow intent before the join, so a late Flow cannot race the fallback;
// handled is false for events without a tracked route, which join as before.
func joinEmailRoute(ctx context.Context, routes routeStore, cli emailQueueJoiner, conn model.EmailEventConnection, req *cc.EmailJoinToQueueRequest, rt joinRetry) (bool, model.Response, *model.AppError) {
	queueId := req.QueueId
	owner := conn.NodeId() + "/flow/" + model.NewId()
	log := conn.Log().With(wlog.Int("queue_id", int(queueId)))

	intent := model.EmailRouteIntent{QueueId: queueId, Priority: req.Priority, StickyAgentId: req.StickyAgentId,
		ExtraEmailCount: req.ExtraEmailCount}
	began, route, appErr := routes.BeginFlowRoute(ctx, owner, conn.EventId(), intent, rt.budget+routeWriteTimeout)
	switch {
	case appErr != nil:
		return true, nil, model.NewAppError("Email.JoinQueue", "email_route.unavailable", nil, appErr.Error(), http.StatusServiceUnavailable)
	case route == nil:
		// Cleanup removes only finished routes, so the deadline of this event is long over.
		emailevent.RecordRouteOutcome(ctx, emailevent.RouteOutcomeWindowClosed)
		log.Warn("email route is no longer kept")
		return true, nil, model.NewAppError("Email.JoinQueue", "email_route.window_closed", nil,
			"the route of the email is no longer kept; its time to choose a queue is over", http.StatusConflict)
	case route.State == model.EmailRouteUnset || route.State == model.EmailRouteNotNeeded:
		return false, nil, nil
	case began:
		// A repeated JoinQueue into the same queue joins with the first intent, as the watcher would.
		if in := route.Intent; in != nil {
			req = withQueue(req, in.QueueId)
			req.Priority, req.StickyAgentId, req.ExtraEmailCount = in.Priority, in.StickyAgentId, in.ExtraEmailCount
		}
		res, err := joinFlowRoute(ctx, routes, cli, conn, owner, req, rt, log)
		return true, res, err
	case route.State == model.EmailRouteJoining:
		log.Warn("email route already joins another queue", wlog.Int("route_queue_id", int(*route.QueueId)))
		return true, nil, queueConflictError(*route.QueueId)
	case route.State == model.EmailRouteRouted && route.Source == model.EmailRouteSourceFlow:
		// A repeated join answers from call_center, which replays the stored join.
		res, err := callEmailJoin(ctx, cli, withQueue(req, *route.QueueId), rt, conn.Log())
		if err != nil {
			return true, nil, joinError(err)
		}
		resp, joinErr := setJoinVariables(ctx, conn, res)
		return true, resp, joinErr
	default:
		emailevent.RecordRouteOutcome(ctx, emailevent.RouteOutcomeWindowClosed)
		log.Warn("email route window is closed", wlog.String("route_state", string(route.State)))
		return true, nil, windowClosedError()
	}
}

// joinFlowRoute joins the queue of the intent and records the outcome; an error that
// may hide a join keeps the intent for the route watcher.
func joinFlowRoute(ctx context.Context, routes routeStore, cli emailQueueJoiner, conn model.EmailEventConnection, owner string, req *cc.EmailJoinToQueueRequest, rt joinRetry, log *wlog.Logger) (model.Response, *model.AppError) {
	res, err := callEmailJoin(ctx, cli, req, rt, conn.Log())

	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), routeWriteTimeout)
	defer cancel()

	if err != nil {
		if failure := emailevent.JoinFailure(model.EmailRouteJoining, err); failure != nil {
			if _, appErr := routes.FinishRoute(wctx, owner, conn.EventId(), model.EmailRouteJoining, *failure); appErr != nil {
				log.Warn("email route intent was not dropped", wlog.Err(appErr))
			}
		} else if _, appErr := routes.RetryRoute(wctx, owner, conn.EventId(), model.EmailRouteJoining, err.Error(), time.Now(), false); appErr != nil {
			log.Warn("email route was not handed to the watcher", wlog.Err(appErr))
		}

		return nil, joinError(err)
	}

	resp, joinErr := setJoinVariables(ctx, conn, res)
	if joinErr != nil {
		if _, appErr := routes.RetryRoute(wctx, owner, conn.EventId(), model.EmailRouteJoining, joinErr.Error(), time.Now(), false); appErr != nil {
			log.Warn("email route was not handed to the watcher", wlog.Err(appErr))
		}

		return nil, joinErr
	}

	actual := res.GetQueueId()
	done, appErr := routes.FinishRoute(wctx, owner, conn.EventId(), model.EmailRouteJoining,
		model.EmailRouteResult{State: model.EmailRouteRouted, QueueId: &actual, Source: model.EmailRouteSourceFlow})
	if appErr != nil || !done {
		// The join is done; the watcher repeats it into the same queue and records the route.
		log.Warn("email route was joined but not recorded; the watcher records it", wlog.Err(appErr))
	}

	return resp, nil
}

// withQueue is the frozen request for another queue; call_center replays a routed event whatever it gets.
func withQueue(req *cc.EmailJoinToQueueRequest, queueId int32) *cc.EmailJoinToQueueRequest {
	return &cc.EmailJoinToQueueRequest{
		QueueId:         queueId,
		ThreadId:        req.ThreadId,
		EmailEventId:    req.EmailEventId,
		DomainId:        req.DomainId,
		Priority:        req.Priority,
		StickyAgentId:   req.StickyAgentId,
		ExtraEmailCount: req.ExtraEmailCount,
		FromAddress:     req.FromAddress,
	}
}

// setJoinVariables stores the cc_* variables of a valid join response.
func setJoinVariables(ctx context.Context, conn model.EmailEventConnection, res *cc.EmailJoinToQueueResponse) (model.Response, *model.AppError) {
	result, ok := joinResults[res.GetResult()]
	if res == nil || !ok || res.GetAttemptId() <= 0 || res.GetQueueId() <= 0 {
		return nil, invalidJoinResponse(res)
	}

	return conn.Set(ctx, model.Variables{
		"cc_attempt_id":  strconv.FormatInt(res.GetAttemptId(), 10),
		"cc_queue_id":    strconv.FormatInt(int64(res.GetQueueId()), 10),
		"cc_join_result": result,
	})
}

func invalidJoinResponse(res *cc.EmailJoinToQueueResponse) *model.AppError {
	return model.NewAppError("Email.JoinQueue", "email.join_queue.invalid_response", nil,
		fmt.Sprintf("call_center returned an invalid email join response: attempt_id=%d queue_id=%d result=%s",
			res.GetAttemptId(), res.GetQueueId(), res.GetResult()),
		http.StatusInternalServerError)
}

func callEmailJoin(ctx context.Context, cli emailQueueJoiner, req *cc.EmailJoinToQueueRequest, rt joinRetry, log *wlog.Logger) (*cc.EmailJoinToQueueResponse, error) {
	return callEmailRetry(ctx, cli.EmailJoinToQueue, req, rt, log, "email join to queue")
}

// callEmailRetry repeats the same request on transient errors; call_center deduplicates it by email_event_id.
func callEmailRetry[Req, Res any](ctx context.Context, call func(context.Context, Req) (Res, error), req Req, rt joinRetry, log *wlog.Logger, what string) (Res, error) {
	ctx, cancel := context.WithTimeout(ctx, rt.budget)
	defer cancel()

	for attempt := 1; ; attempt++ {
		callCtx, callCancel := context.WithTimeout(ctx, rt.callTimeout)
		res, err := call(callCtx, req)
		callCancel()
		if err == nil {
			return res, nil
		}

		var none Res
		code := status.Code(err)
		if (code != codes.Unavailable && code != codes.DeadlineExceeded) || attempt >= rt.attempts || ctx.Err() != nil {
			return none, err
		}

		delay := min(rt.baseDelay<<(attempt-1), rt.maxDelay)
		delay = delay/2 + rand.N(delay/2+1)
		log.Warn(what+" failed, retrying", wlog.Err(err), wlog.Int("attempt", attempt),
			wlog.Duration("delay", delay))

		select {
		case <-ctx.Done():
			return none, err
		case <-time.After(delay):
		}
	}
}

func joinError(err error) *model.AppError {
	id, code := "app_error", http.StatusInternalServerError
	switch status.Code(err) {
	case codes.InvalidArgument:
		id, code = "invalid", http.StatusBadRequest
	case codes.NotFound:
		id, code = "queue_not_found", http.StatusNotFound
	case codes.FailedPrecondition:
		id, code = "queue_precondition", http.StatusPreconditionFailed
	case codes.AlreadyExists:
		id, code = "event_conflict", http.StatusConflict
	case codes.Unavailable, codes.DeadlineExceeded:
		id, code = "unavailable", http.StatusServiceUnavailable
	case codes.Canceled:
		id = "canceled"
	}

	return model.NewAppError("Email.JoinQueue", "email.join_queue."+id, nil, err.Error(), code)
}
