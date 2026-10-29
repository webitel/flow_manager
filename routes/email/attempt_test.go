package email

import (
	"context"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/gen/cc"
)

type fakeCommander struct {
	errs      []error
	postpones []*cc.EmailAttemptPostponeRequest
	resumes   []*cc.ResumeAttemptRequest
}

func (f *fakeCommander) next(calls int) error {
	if len(f.errs) >= calls {
		return f.errs[calls-1]
	}
	return nil
}

func (f *fakeCommander) EmailAttemptPostpone(_ context.Context, in *cc.EmailAttemptPostponeRequest) (*cc.EmailAttemptResponse, error) {
	f.postpones = append(f.postpones, in)
	if err := f.next(len(f.postpones)); err != nil {
		return nil, err
	}
	return &cc.EmailAttemptResponse{AttemptId: 7, State: "postponed", AttemptVersion: 3, PostponedUntil: 1700000000000}, nil
}

func (f *fakeCommander) EmailAttemptResume(_ context.Context, in *cc.ResumeAttemptRequest) (*cc.ResumeAttemptResponse, error) {
	f.resumes = append(f.resumes, in)
	if err := f.next(len(f.resumes)); err != nil {
		return nil, err
	}
	return &cc.ResumeAttemptResponse{Email: &cc.EmailAttemptResponse{AttemptId: 7, State: "active", AttemptVersion: 4}}, nil
}

// The Flow form carries only the event ids, and a retry repeats the same request.
func TestPostponeSendsFlowFormAndRetries(t *testing.T) {
	conn, cli := &fakeEventConn{}, &fakeCommander{errs: []error{status.Error(codes.Unavailable, "down")}}

	if _, err := postponeEmailAttempt(context.Background(), cli, conn, PostponeParameters{Seconds: 600}, testRetry); err != nil {
		t.Fatal(err)
	}
	if len(cli.postpones) != 2 || cli.postpones[0] != cli.postpones[1] {
		t.Fatalf("calls = %d, want the same request twice", len(cli.postpones))
	}
	in := cli.postpones[0]
	if in.GetDomainId() != 1 || in.GetThreadId() != 42 || in.GetFlow().GetEmailEventId() != conn.EventId() || in.GetPostponeForSec() != 600 {
		t.Fatalf("request %v", in)
	}
	if conn.vars["cc_attempt_state"] != "postponed" || conn.vars["cc_postponed_until"] != "1700000000000" {
		t.Fatalf("vars %v", conn.vars)
	}

	for _, bad := range []PostponeParameters{{}, {Seconds: 1, Until: 1}, {Seconds: -1}} {
		if _, err := postponeEmailAttempt(context.Background(), cli, conn, bad, testRetry); err == nil || err.Id != "email.postpone.valid.deadline" {
			t.Errorf("%+v: %v", bad, err)
		}
	}
}

// Application errors reach the Flow with the id an operator gets and are not retried.
func TestResumeKeepsCallCenterErrorId(t *testing.T) {
	conn := &fakeEventConn{}
	cli := &fakeCommander{errs: []error{status.Error(codes.ResourceExhausted, "cc.email_attempt.slot_exhausted")}}

	_, err := resumeEmailAttempt(context.Background(), cli, conn, testRetry)
	if err == nil || err.Id != "cc.email_attempt.slot_exhausted" || err.StatusCode != http.StatusTooManyRequests || len(cli.resumes) != 1 {
		t.Fatalf("err %v after %d calls", err, len(cli.resumes))
	}
	if in := cli.resumes[0]; in.GetThreadId() != 42 || in.GetEmailEventId() != conn.EventId() || in.GetAttemptId() != 0 {
		t.Fatalf("request %v", in)
	}

	if _, err = resumeEmailAttempt(context.Background(), &fakeCommander{}, conn, testRetry); err != nil || conn.vars["cc_attempt_state"] != "active" {
		t.Fatalf("resume: %v, vars %v", err, conn.vars)
	}
}
