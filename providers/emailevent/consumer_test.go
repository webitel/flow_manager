package emailevent

import (
	"context"
	"net/http"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/webitel/flow_manager/model"
)

type settlement struct {
	acks, nacks int
	requeue     bool
}

func (s *settlement) Ack(uint64, bool) error { s.acks++; return nil }

func (s *settlement) Nack(_ uint64, _ bool, requeue bool) error {
	s.nacks++
	s.requeue = requeue
	return nil
}

func (s *settlement) Reject(_ uint64, requeue bool) error { return s.Nack(0, false, requeue) }

// fakeInbox returns errs in turn, then succeeds; onCall runs before each insert.
type fakeInbox struct {
	errs   []int
	calls  int
	onCall func(call int)
}

func (f *fakeInbox) Insert(context.Context, *model.EmailEvent) (bool, *model.AppError) {
	f.calls++
	if f.onCall != nil {
		f.onCall(f.calls)
	}
	if f.calls > len(f.errs) {
		return true, nil
	}

	return false, model.NewAppError("test", "test", nil, "db", f.errs[f.calls-1])
}

const validEvent = `{"event_id": "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60", "event_type": "email.message.inbound",
  "schema_version": 1, "domain_id": 1, "profile_id": 2, "thread_id": 4, "message_id": 5,
  "snapshot": {"message_kind": "regular", "thread_kind": "regular"}}`

func TestConsumerSettlesDeliveries(t *testing.T) {
	const transient, invalidData = http.StatusInternalServerError, http.StatusBadRequest
	forever := make([]int, 1000)
	for i := range forever {
		forever[i] = transient
	}

	tests := []struct {
		name       string
		body       string
		errs       []int
		cancelAt   int // the generation ends on this insert, like a closed channel
		inserts    int
		acks       int
		deadLetter bool
	}{
		{name: "invalid payload", body: `{"event_type": "x"}`, deadLetter: true},
		{name: "stored", body: validEvent, inserts: 1, acks: 1},
		{name: "transient errors are retried", body: validEvent, errs: []int{transient, transient}, inserts: 3, acks: 1},
		{name: "invalid data", body: validEvent, errs: []int{invalidData}, inserts: 1, deadLetter: true},
		{name: "closed channel stops retries", body: validEvent, errs: forever, cancelAt: 3, inserts: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			inbox := &fakeInbox{errs: tt.errs, onCall: func(call int) {
				if call == tt.cancelAt {
					cancel()
				}
			}}
			c := &consumer{inbox: inbox, log: testLog, minBackoff: time.Millisecond, maxBackoff: time.Millisecond}
			s := &settlement{}

			if err := c.handle(ctx, amqp.Delivery{Acknowledger: s, Body: []byte(tt.body)}); err != nil {
				t.Fatalf("handle: %v", err)
			}

			nacks := 0
			if tt.deadLetter {
				nacks = 1
			}
			if inbox.calls != tt.inserts || s.acks != tt.acks || s.nacks != nacks || s.requeue {
				t.Fatalf("inserts %d, acks %d, nacks %d (requeue %v); want %d, %d, %d (no requeue)",
					inbox.calls, s.acks, s.nacks, s.requeue, tt.inserts, tt.acks, nacks)
			}
		})
	}
}
