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

type emailQueueService interface {
	emailQueueJoiner
	FindQueueByName(domainId int64, name string) (int32, *model.AppError)
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

	return joinEmailQueue(ctx, r.fm, event, argv, defaultJoinRetry)
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
	priority, appErr := emailQueueNumber(argv.Priority, 100, "priority")
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
		return nil, model.NewAppError("Email.JoinQueue", "email.join_queue.valid.queue", nil,
			"queue id or name is required", http.StatusBadRequest)
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

	res, err := callEmailJoin(ctx, cli, req, rt, conn.Log())
	if err != nil {
		return nil, joinError(err)
	}

	// Only the three known results with positive ids count as a join.
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
	ctx, cancel := context.WithTimeout(ctx, rt.budget)
	defer cancel()

	for attempt := 1; ; attempt++ {
		callCtx, callCancel := context.WithTimeout(ctx, rt.callTimeout)
		res, err := cli.EmailJoinToQueue(callCtx, req)
		callCancel()
		if err == nil {
			return res, nil
		}

		code := status.Code(err)
		if (code != codes.Unavailable && code != codes.DeadlineExceeded) || attempt >= rt.attempts || ctx.Err() != nil {
			return nil, err
		}

		delay := min(rt.baseDelay<<(attempt-1), rt.maxDelay)
		delay = delay/2 + rand.N(delay/2+1)
		log.Warn("email join to queue failed, retrying", wlog.Err(err), wlog.Int("attempt", attempt),
			wlog.Duration("delay", delay))

		select {
		case <-ctx.Done():
			return nil, err
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
