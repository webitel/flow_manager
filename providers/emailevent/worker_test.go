package emailevent

import (
	"context"
	"net/http"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// acceptStore answers MarkAccepted with results in turn; a zero result is a
// database error.
type acceptStore struct {
	store.EmailEventStore
	results []model.EmailEventAcceptResult
	calls   int
}

func (s *acceptStore) MarkAccepted(context.Context, string, string) (model.EmailEventAcceptResult, *model.AppError) {
	result := s.results[s.calls]
	s.calls++
	if result == 0 {
		return 0, model.NewAppError("test", "test", nil, "db", http.StatusInternalServerError)
	}

	return result, nil
}

// Only a lost lease may cancel a Flow that already started.
func TestWorkerAcceptKeepsStartedFlow(t *testing.T) {
	tests := []struct {
		name      string
		results   []model.EmailEventAcceptResult
		cancelled bool
	}{
		{"ownership lost", []model.EmailEventAcceptResult{model.EmailEventOwnershipLost}, true},
		{"already accepted", []model.EmailEventAcceptResult{model.EmailEventAlreadyAccepted}, false},
		{"database error is retried", []model.EmailEventAcceptResult{0, model.EmailEventApplied}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &acceptStore{results: tt.results}
			w := &worker{Server: &Server{store: st}, log: testLog, minBackoff: time.Millisecond, maxBackoff: time.Millisecond}
			flowId := int64(1)
			conn := NewConnection(context.Background(), "flow-1", &InboundMessage{FlowId: &flowId}, testLog)
			conn.MarkStarted()

			w.accept(context.Background(), &model.EmailEvent{EventId: "e"}, conn, testLog)

			if st.calls != len(tt.results) || (conn.Context().Err() != nil) != tt.cancelled {
				t.Fatalf("calls %d, flow cancelled %v; want %d, %v", st.calls, conn.Context().Err() != nil, len(tt.results), tt.cancelled)
			}
		})
	}
}

type fakeWaker struct {
	err   error
	calls []*cc.EmailWakeUpRequest
}

func (f *fakeWaker) EmailWakeUp(_ context.Context, in *cc.EmailWakeUpRequest) (*cc.EmailWakeUpResponse, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	return &cc.EmailWakeUpResponse{Result: cc.EmailWakeUpResponse_RESULT_NOOP}, nil
}

// settleStore records which write settled the event.
type settleStore struct {
	store.EmailEventStore
	settled string
}

func (s *settleStore) MarkAccepted(context.Context, string, string) (model.EmailEventAcceptResult, *model.AppError) {
	s.settled = "accepted"
	return model.EmailEventApplied, nil
}

func (s *settleStore) Release(context.Context, string, string, string, time.Time) (bool, *model.AppError) {
	s.settled = "released"
	return true, nil
}

func (s *settleStore) MarkFailed(context.Context, string, string, string) (bool, *model.AppError) {
	s.settled = "failed"
	return true, nil
}

// Wake-up runs for every event before its Flow; only an unretriable error fails the event.
func TestWorkerWakesUpBeforeFlow(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{
		"accepted without a flow": {nil, "accepted"},
		"unavailable is retried":  {status.Error(codes.Unavailable, "down"), "released"},
		"deadline is retried":     {status.Error(codes.DeadlineExceeded, "slow"), "released"},
		"event conflict fails":    {status.Error(codes.AlreadyExists, "cc.email_wakeup.event_conflict"), "failed"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			st, waker := &settleStore{}, &fakeWaker{err: tt.err}
			w := &worker{Server: &Server{store: st, waker: waker}, log: testLog, minBackoff: time.Millisecond, maxBackoff: time.Millisecond}

			w.process(context.Background(), &model.EmailEvent{EventId: "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60", Payload: []byte(eventWithoutFlow)})

			if st.settled != tt.want || len(waker.calls) != 1 {
				t.Fatalf("settled %q after %d wake-ups; want %q after 1", st.settled, len(waker.calls), tt.want)
			}
			if in := waker.calls[0]; in.GetDomainId() != 1 || in.GetThreadId() != 4 || in.GetEmailEventId() != "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60" {
				t.Fatalf("wake-up request %v", in)
			}
		})
	}
}

const eventWithoutFlow = `{"event_id": "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60", "event_type": "email.message.inbound",
  "schema_version": 1, "domain_id": 1, "profile_id": 2, "thread_id": 4, "message_id": 5,
  "snapshot": {"message_kind": "regular", "thread_kind": "regular"}}`
