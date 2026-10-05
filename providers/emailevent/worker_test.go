package emailevent

import (
	"context"
	"net/http"
	"testing"
	"time"

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
