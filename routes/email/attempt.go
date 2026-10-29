package email

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
)

// PostponeParameters takes exactly one deadline: seconds from now or until, in Unix milliseconds.
type PostponeParameters struct {
	Seconds int32 `json:"seconds"`
	Until   int64 `json:"until"`
}

type emailAttemptCommander interface {
	EmailAttemptPostpone(ctx context.Context, in *cc.EmailAttemptPostponeRequest) (*cc.EmailAttemptResponse, error)
	EmailAttemptResume(ctx context.Context, in *cc.ResumeAttemptRequest) (*cc.ResumeAttemptResponse, error)
}

func (r *Router) postpone(ctx context.Context, scope *flow.Flow, conn model.EmailConnection, args interface{}) (model.Response, *model.AppError) {
	event, ok := conn.(model.EmailEventConnection)
	if !ok {
		return nil, notEventConnection("Email.Postpone", "postpone")
	}

	var argv PostponeParameters
	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return postponeEmailAttempt(ctx, r.fm, event, argv, defaultJoinRetry)
}

func (r *Router) resume(ctx context.Context, _ *flow.Flow, conn model.EmailConnection, _ interface{}) (model.Response, *model.AppError) {
	event, ok := conn.(model.EmailEventConnection)
	if !ok {
		return nil, notEventConnection("Email.Resume", "resume")
	}

	return resumeEmailAttempt(ctx, r.fm, event, defaultJoinRetry)
}

// postponeEmailAttempt sends the Flow form: call_center finds the attempt and keys the command by the event.
func postponeEmailAttempt(ctx context.Context, cli emailAttemptCommander, conn model.EmailEventConnection, argv PostponeParameters, rt joinRetry) (model.Response, *model.AppError) {
	req := &cc.EmailAttemptPostponeRequest{
		DomainId: conn.DomainId(),
		ThreadId: conn.ThreadId(),
		Caller:   &cc.EmailAttemptPostponeRequest_Flow{Flow: &cc.EmailAttemptFlow{EmailEventId: conn.EventId()}},
	}
	switch {
	case argv.Seconds > 0 && argv.Until == 0:
		req.Deadline = &cc.EmailAttemptPostponeRequest_PostponeForSec{PostponeForSec: argv.Seconds}
	case argv.Until > 0 && argv.Seconds == 0:
		req.Deadline = &cc.EmailAttemptPostponeRequest_PostponedUntil{PostponedUntil: argv.Until}
	default:
		return nil, model.NewAppError("Email.Postpone", "email.postpone.valid.deadline", nil,
			"exactly one of positive seconds or until (Unix milliseconds) is required", http.StatusBadRequest)
	}

	res, err := callEmailRetry(ctx, cli.EmailAttemptPostpone, req, rt, conn.Log(), "email attempt postpone")
	if err != nil {
		return nil, emailAttemptError("Email.Postpone", "email.postpone.", err)
	}

	return setAttemptVariables(ctx, conn, "Email.Postpone", "email.postpone.", res)
}

func resumeEmailAttempt(ctx context.Context, cli emailAttemptCommander, conn model.EmailEventConnection, rt joinRetry) (model.Response, *model.AppError) {
	req := &cc.ResumeAttemptRequest{
		DomainId:     conn.DomainId(),
		ThreadId:     conn.ThreadId(),
		EmailEventId: conn.EventId(),
	}

	res, err := callEmailRetry(ctx, cli.EmailAttemptResume, req, rt, conn.Log(), "email attempt resume")
	if err != nil {
		return nil, emailAttemptError("Email.Resume", "email.resume.", err)
	}

	return setAttemptVariables(ctx, conn, "Email.Resume", "email.resume.", res.GetEmail())
}

func setAttemptVariables(ctx context.Context, conn model.EmailEventConnection, where, prefix string, res *cc.EmailAttemptResponse) (model.Response, *model.AppError) {
	if res.GetAttemptId() <= 0 || res.GetState() == "" {
		return nil, model.NewAppError(where, prefix+"invalid_response", nil,
			fmt.Sprintf("call_center returned an invalid email attempt response: attempt_id=%d state=%q",
				res.GetAttemptId(), res.GetState()), http.StatusInternalServerError)
	}

	// Set merges into the variables: an empty deadline replaces the one of an earlier command.
	vars := model.Variables{
		"cc_attempt_id":      strconv.FormatInt(res.GetAttemptId(), 10),
		"cc_attempt_state":   res.GetState(),
		"cc_attempt_version": strconv.FormatInt(res.GetAttemptVersion(), 10),
		"cc_postponed_until": "",
	}
	if res.GetPostponedUntil() > 0 {
		vars["cc_postponed_until"] = strconv.FormatInt(res.GetPostponedUntil(), 10)
	}

	return conn.Set(ctx, vars)
}

// emailAttemptError keeps the stable call_center id, the same an operator gets, when there is one.
func emailAttemptError(where, prefix string, err error) *model.AppError {
	st := status.Convert(err)
	id, code := prefix+"app_error", http.StatusInternalServerError
	switch st.Code() {
	case codes.InvalidArgument:
		id, code = prefix+"invalid", http.StatusBadRequest
	case codes.NotFound:
		id, code = prefix+"not_found", http.StatusNotFound
	case codes.FailedPrecondition:
		id, code = prefix+"precondition", http.StatusPreconditionFailed
	case codes.Aborted, codes.AlreadyExists:
		id, code = prefix+"conflict", http.StatusConflict
	case codes.ResourceExhausted:
		id, code = prefix+"exhausted", http.StatusTooManyRequests
	case codes.Unavailable, codes.DeadlineExceeded:
		id, code = prefix+"unavailable", http.StatusServiceUnavailable
	case codes.Canceled:
		id = prefix + "canceled"
	}
	if strings.HasPrefix(st.Message(), "cc.email_attempt.") {
		id = st.Message()
	}

	return model.NewAppError(where, id, nil, err.Error(), code)
}

func notEventConnection(where, app string) *model.AppError {
	return model.NewAppError(where, "email."+app+".not_supported", nil,
		app+" needs a connection started by an email-service event", http.StatusBadRequest)
}
