//go:build integration

package sqlstore

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/webitel/flow_manager/model"
)

// masterOnly is the part of SqlSupplier the store uses; NewSqlSupplier would
// exit the process when the database is unreachable.
type masterOnly struct {
	SqlStore
	db *gorp.DbMap
}

func (s masterOnly) GetMaster() *gorp.DbMap { return s.db }

// emailEventStore applies the call_center migration to the database of
// FLOW_TEST_POSTGRES_DSN and empties the Inbox.
func emailEventStore(t *testing.T) (*SqlEmailEventStore, *sql.DB) {
	t.Helper()

	dsn, migration := os.Getenv("FLOW_TEST_POSTGRES_DSN"), os.Getenv("FLOW_EMAIL_EVENT_MIGRATION")
	if dsn == "" || migration == "" {
		t.Skip("set FLOW_TEST_POSTGRES_DSN and FLOW_EMAIL_EVENT_MIGRATION (path to call_center email_service_inbox.sql)")
	}

	ddl, err := os.ReadFile(migration)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, query := range []string{`create schema if not exists flow`, string(ddl), `truncate flow.email_event`} {
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("prepare database: %v", err)
		}
	}

	dbMap := &gorp.DbMap{Db: db, TypeConverter: typeConverter{}, Dialect: &PostgresJSONDialect{}}

	return &SqlEmailEventStore{masterOnly{db: dbMap}}, db
}

func newEmailEvent(t *testing.T, db *sql.DB) *model.EmailEvent {
	t.Helper()

	var id string
	if err := db.QueryRow(`select gen_random_uuid()::text`).Scan(&id); err != nil {
		t.Fatalf("event id: %v", err)
	}

	return &model.EmailEvent{
		EventId: id, DomainId: 1, EventType: "email.message.inbound", SchemaVersion: 1,
		ThreadId: 10, MessageId: 20, Payload: []byte(`{}`),
	}
}

func TestEmailEventStore(t *testing.T) {
	ctx := context.Background()
	s, db := emailEventStore(t)

	reset := func(t *testing.T) {
		if _, err := db.Exec(`truncate flow.email_event`); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	}

	t.Run("a redelivered event is stored once", func(t *testing.T) {
		reset(t)
		e := newEmailEvent(t, db)

		for i, want := range []bool{true, false} {
			inserted, appErr := s.Insert(ctx, e)
			if appErr != nil || inserted != want {
				t.Fatalf("insert %d: inserted %v, err %v; want %v", i+1, inserted, appErr, want)
			}
		}
	})

	t.Run("a worker that lost its lease cannot finish the row", func(t *testing.T) {
		reset(t)
		e := newEmailEvent(t, db)
		if _, appErr := s.Insert(ctx, e); appErr != nil {
			t.Fatalf("insert: %v", appErr)
		}

		// Two workers of one process: only the worker index tells them apart.
		const oldOwner, newOwner = "flow-1/run/0", "flow-1/run/1"
		claim := func(owner string) {
			t.Helper()
			got, appErr := s.Claim(ctx, owner, time.Minute)
			if appErr != nil || got == nil || got.EventId != e.EventId {
				t.Fatalf("claim by %s: %+v, %v", owner, got, appErr)
			}
		}

		claim(oldOwner)
		if _, err := db.Exec(`update flow.email_event set locked_until = now() - interval '1 second'`); err != nil {
			t.Fatalf("expire lease: %v", err)
		}
		if recovered, appErr := s.RecoverExpired(ctx); appErr != nil || recovered != 1 {
			t.Fatalf("recover: %d, %v", recovered, appErr)
		}
		claim(newOwner)

		if result, appErr := s.MarkAccepted(ctx, oldOwner, e.EventId); appErr != nil || result != model.EmailEventOwnershipLost {
			t.Fatalf("accept by the old owner: %v, %v", result, appErr)
		}
		if released, appErr := s.Release(ctx, oldOwner, e.EventId, "late", time.Now()); appErr != nil || released {
			t.Fatalf("release by the old owner: %v, %v", released, appErr)
		}

		// The repeat stands for a retry after a commit whose reply was lost.
		for _, want := range []model.EmailEventAcceptResult{model.EmailEventApplied, model.EmailEventAlreadyAccepted} {
			if result, appErr := s.MarkAccepted(ctx, newOwner, e.EventId); appErr != nil || result != want {
				t.Fatalf("accept by the new owner: %v, %v; want %v", result, appErr, want)
			}
		}
	})

	t.Run("cleanup removes only accepted rows past retention", func(t *testing.T) {
		reset(t)

		states := map[string]string{
			"old accepted":   `state = 'accepted', accepted_at = now() - interval '2 hours'`,
			"fresh accepted": `state = 'accepted', accepted_at = now()`,
			"old pending":    `created_at = now() - interval '2 hours'`,
			"old failed":     `state = 'failed', last_error = 'x', created_at = now() - interval '2 hours'`,
		}
		for _, set := range states {
			e := newEmailEvent(t, db)
			if _, appErr := s.Insert(ctx, e); appErr != nil {
				t.Fatalf("insert: %v", appErr)
			}
			if _, err := db.Exec(`update flow.email_event set `+set+` where event_id = $1`, e.EventId); err != nil {
				t.Fatalf("set state: %v", err)
			}
		}

		deleted, appErr := s.DeleteAcceptedBefore(ctx, time.Now().Add(-time.Hour), 10)
		if appErr != nil || deleted != 1 {
			t.Fatalf("cleanup: deleted %d, %v; want 1", deleted, appErr)
		}

		var oldAccepted int
		if err := db.QueryRow(`select count(*) from flow.email_event where accepted_at < now() - interval '1 hour'`).Scan(&oldAccepted); err != nil || oldAccepted != 0 {
			t.Fatalf("old accepted rows left: %d, %v", oldAccepted, err)
		}
	})
}
