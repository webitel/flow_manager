package email

import (
	"context"
	"testing"
	"time"

	"github.com/webitel/wlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
)

type fakeEventConn struct {
	model.EmailEventConnection
	vars model.Variables
}

func (c *fakeEventConn) EventId() string   { return "7c9e6679-7425-40de-944b-e07fc1f90ae7" }
func (c *fakeEventConn) ThreadId() int64   { return 42 }
func (c *fakeEventConn) DomainId() int64   { return 1 }
func (c *fakeEventConn) Log() *wlog.Logger { return testLog }
func (c *fakeEventConn) Set(_ context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.vars = vars
	return model.CallResponseOK, nil
}

type fakeJoiner struct {
	errs  []error
	calls []*cc.EmailJoinToQueueRequest
	res   *cc.EmailJoinToQueueResponse
	set   bool // res is returned as is, even nil
}

func (j *fakeJoiner) EmailJoinToQueue(_ context.Context, in *cc.EmailJoinToQueueRequest) (*cc.EmailJoinToQueueResponse, error) {
	j.calls = append(j.calls, in)
	if len(j.errs) >= len(j.calls) {
		return nil, j.errs[len(j.calls)-1]
	}
	if j.set {
		return j.res, nil
	}
	return &cc.EmailJoinToQueueResponse{AttemptId: 7, QueueId: 3, Result: cc.EmailJoinToQueueResponse_RESULT_REPLAYED}, nil
}

var testLog = wlog.NewLogger(&wlog.LoggerConfiguration{})

var testRetry = joinRetry{attempts: 3, callTimeout: time.Second, budget: 5 * time.Second, baseDelay: time.Millisecond, maxDelay: time.Millisecond}

func TestJoinEmailQueueRetriesUnavailable(t *testing.T) {
	conn := &fakeEventConn{}
	cli := &fakeJoiner{errs: []error{status.Error(codes.Unavailable, "down"), status.Error(codes.DeadlineExceeded, "slow")}}

	if _, err := joinEmailQueue(context.Background(), cli, conn, 3, testRetry); err != nil {
		t.Fatal(err)
	}
	if len(cli.calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(cli.calls))
	}
	req := cli.calls[0]
	if req.EmailEventId != conn.EventId() || req.ThreadId != 42 || req.DomainId != 1 || req.QueueId != 3 {
		t.Fatalf("request = %+v", req)
	}
	if conn.vars["cc_attempt_id"] != "7" || conn.vars["cc_queue_id"] != "3" || conn.vars["cc_join_result"] != "replayed" {
		t.Fatalf("vars = %v", conn.vars)
	}
}

func TestJoinEmailQueueDoesNotRetryPermanentError(t *testing.T) {
	cli := &fakeJoiner{errs: []error{status.Error(codes.FailedPrecondition, "disabled")}}

	_, err := joinEmailQueue(context.Background(), cli, &fakeEventConn{}, 3, testRetry)
	if err == nil || err.Id != "email.join_queue.queue_precondition" || len(cli.calls) != 1 {
		t.Fatalf("err = %v, calls = %d", err, len(cli.calls))
	}
}

func TestJoinQueueRejectsLegacyConnection(t *testing.T) {
	var legacy model.EmailConnection
	_, err := (&Router{}).joinQueue(context.Background(), nil, legacy, nil)
	if err == nil || err.Id != "email.join_queue.not_supported" {
		t.Fatalf("err = %v", err)
	}
}

func TestJoinEmailQueueRejectsInvalidResponse(t *testing.T) {
	cases := map[string]*cc.EmailJoinToQueueResponse{
		"nil":            nil,
		"unspecified":    {AttemptId: 7, QueueId: 3},
		"unknown result": {AttemptId: 7, QueueId: 3, Result: 99},
		"no attempt":     {QueueId: 3, Result: cc.EmailJoinToQueueResponse_RESULT_CREATED},
		"no queue":       {AttemptId: 7, Result: cc.EmailJoinToQueueResponse_RESULT_CREATED},
	}
	for name, res := range cases {
		conn := &fakeEventConn{}
		_, err := joinEmailQueue(context.Background(), &fakeJoiner{res: res, set: true}, conn, 3, testRetry)
		if err == nil || err.Id != "email.join_queue.invalid_response" || conn.vars != nil {
			t.Errorf("%s: err = %v, vars = %v", name, err, conn.vars)
		}
	}
}
