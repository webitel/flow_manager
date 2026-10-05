//go:build integration

package emailevent

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/webitel/flow_manager/model"
	sqlstore "github.com/webitel/flow_manager/store/pg_store"
)

type masterOnly struct {
	sqlstore.SqlStore
	db *gorp.DbMap
}

func (s masterOnly) GetMaster() *gorp.DbMap { return s.db }

// The worker records the launch result of the router in the Inbox.
func TestWorkerRecordsLaunch(t *testing.T) {
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
	defer db.Close()
	if _, err := db.Exec(`create schema if not exists flow`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(string(ddl)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	st := sqlstore.NewSqlEmailEventStore(masterOnly{db: &gorp.DbMap{Db: db, Dialect: &sqlstore.PostgresJSONDialect{}}})
	flowId := int64(3)

	tests := []struct {
		name      string
		attempts  int32
		launchErr int // 0: the router starts the Flow
		state     model.EmailEventState
		wantTries int32
	}{
		{name: "a started flow is accepted", state: model.EmailEventAccepted},
		{name: "a failed launch is retried", launchErr: http.StatusInternalServerError, state: model.EmailEventPending, wantTries: 1},
		{name: "a schema still missing at the limit fails", attempts: missingSchemaAttempts - 1,
			launchErr: http.StatusNotFound, state: model.EmailEventFailed, wantTries: missingSchemaAttempts},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if _, err := db.Exec(`truncate flow.email_event`); err != nil {
				t.Fatalf("truncate: %v", err)
			}

			e := &model.EmailEvent{EventId: "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60", DomainId: 1, EventType: TypeInboundMessage,
				SchemaVersion: 1, ThreadId: 4, MessageId: 5, FlowId: &flowId, Payload: []byte(validEventWithFlow)}
			if _, appErr := st.Insert(ctx, e); appErr != nil {
				t.Fatalf("insert: %v", appErr)
			}
			if _, err := db.Exec(`update flow.email_event set attempts = $1`, tt.attempts); err != nil {
				t.Fatalf("set attempts: %v", err)
			}

			s := &Server{nodeId: "flow-1", runId: "run", store: st, log: testLog, conns: make(chan model.Connection), ctx: ctx}
			w := newWorker(s, 0)

			// A router that answers like EmailRouter.launch.
			go func() {
				conn := (<-s.conns).(model.EmailLaunchConnection)
				if tt.launchErr != 0 {
					conn.ReportLaunch(model.NewAppError("test", "test", nil, "launch", tt.launchErr))
					return
				}
				conn.MarkStarted()
				conn.ReportLaunch(nil)
			}()

			row, appErr := st.Claim(ctx, w.owner, leaseDuration)
			if appErr != nil || row == nil {
				t.Fatalf("claim: %v, %v", row, appErr)
			}
			w.process(ctx, row)

			var (
				state    model.EmailEventState
				attempts int32
			)
			if err := db.QueryRow(`select state, attempts from flow.email_event`).Scan(&state, &attempts); err != nil {
				t.Fatalf("read row: %v", err)
			}
			if state != tt.state || attempts != tt.wantTries {
				t.Fatalf("state %s, attempts %d; want %s, %d", state, attempts, tt.state, tt.wantTries)
			}
		})
	}
}

const validEventWithFlow = `{"event_id": "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60", "event_type": "email.message.inbound",
  "schema_version": 1, "domain_id": 1, "profile_id": 2, "flow_id": 3, "thread_id": 4, "message_id": 5,
  "snapshot": {"message_kind": "regular", "thread_kind": "regular"}}`
