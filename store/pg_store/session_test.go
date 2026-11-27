package sqlstore

import (
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// sessionTableDDL mirrors flow.session as created by the call_center migrations.
const sessionTableDDL = `create schema if not exists flow;
create unlogged table if not exists flow.session (
    id text not null primary key,
    app_id text,
    seq integer default 1 not null,
    created_at timestamp with time zone default now(),
    updated_at timestamp with time zone default now()
)`

type masterOnlyStore struct {
	SqlStore

	master *gorp.DbMap
}

func (s masterOnlyStore) GetMaster() *gorp.DbMap { return s.master }

// newSessionStoreForTest connects to the disposable database in FLOW_TEST_DSN and makes
// sure flow.session exists there. Never point it at a shared database.
func newSessionStoreForTest(t *testing.T) (store.SessionStore, *sql.DB) {
	t.Helper()

	dsn := os.Getenv("FLOW_TEST_DSN")
	if dsn == "" {
		t.Skip("FLOW_TEST_DSN is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if _, err = db.Exec(sessionTableDDL); err != nil {
		t.Fatalf("create flow.session: %v", err)
	}

	return NewSQLSessionStore(masterOnlyStore{
		master: &gorp.DbMap{Db: db, TypeConverter: typeConverter{}, Dialect: &PostgresJSONDialect{}},
	}), db
}

func newTestSession(t *testing.T, s store.SessionStore) (threadID, id string) {
	t.Helper()

	threadID = model.NewId()

	t.Cleanup(func() { _ = s.RemoveByThread(threadID) })

	return threadID, threadID + ".421"
}

func mustTouch(t *testing.T, s store.SessionStore, id, appID string) *int {
	t.Helper()

	seq, err := s.Touch(id, appID)
	if err != nil {
		t.Fatalf("Touch(%s): %v", appID, err)
	}

	return seq
}

func mustConsumed(t *testing.T, s store.SessionStore, id string, at int64) bool {
	t.Helper()

	consumed, err := s.Consumed(id, model.IMEventTypeMessage, at)
	if err != nil {
		t.Fatalf("Consumed: %v", err)
	}

	return consumed
}

func TestSQLSessionStore_ReleaseFreesClaimAndKeepsRecord(t *testing.T) {
	s, _ := newSessionStoreForTest(t)
	_, id := newTestSession(t, s)

	const lastSeen int64 = 1795089501148

	if mustTouch(t, s, id, "fm-a") == nil {
		t.Fatal("fresh session was not claimed")
	}

	if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: lastSeen}); err != nil {
		t.Fatalf("Release: %v", err)
	}

	tests := []struct {
		name string
		at   int64
		want bool
	}{
		{name: "older event", at: lastSeen - 1, want: true},
		{name: "same event", at: lastSeen, want: true},
		{name: "next millisecond", at: lastSeen + 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mustConsumed(t, s, id, tt.at); got != tt.want {
				t.Errorf("Consumed(%d) = %v, want %v", tt.at, got, tt.want)
			}
		})
	}

	if mustTouch(t, s, id, "fm-b") == nil {
		t.Error("claim is still held after Release")
	}
}

// The node that wins a claim freed by Release must already see the record Release wrote.
func TestSQLSessionStore_WinnerOfFreedClaimSeesRecord(t *testing.T) {
	s, _ := newSessionStoreForTest(t)

	for i := range 50 {
		_, id := newTestSession(t, s)
		at := int64(1000 + i)

		mustTouch(t, s, id, "fm-a")

		seen := make(chan error, 1)
		stop := make(chan struct{})

		t.Cleanup(func() { close(stop) })

		go func() {
			for {
				select {
				case <-stop:
					return
				default:
				}

				seq, err := s.Touch(id, "fm-b")
				if err != nil {
					seen <- err

					return
				}

				if seq == nil {
					continue
				}

				consumed, err := s.Consumed(id, model.IMEventTypeMessage, at)
				if err == nil && !consumed {
					err = errors.New("claim won before the record was visible")
				}

				seen <- err

				return
			}
		}()

		if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: at}); err != nil {
			t.Fatalf("Release: %v", err)
		}

		if err := <-seen; err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}

func TestSQLSessionStore_RecordsKindsApart(t *testing.T) {
	s, _ := newSessionStoreForTest(t)
	_, id := newTestSession(t, s)

	mustTouch(t, s, id, "fm-a")

	if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: 2000, Callbacks: 1000}); err != nil {
		t.Fatalf("Release: %v", err)
	}

	tests := []struct {
		kind string
		at   int64
		want bool
	}{
		{kind: model.IMEventTypeMessage, at: 1500, want: true},
		{kind: model.IMEventTypeCallback, at: 1500, want: false},
		{kind: model.IMEventTypeCallback, at: 1000, want: true},
	}

	for _, tt := range tests {
		consumed, err := s.Consumed(id, tt.kind, tt.at)
		if err != nil {
			t.Fatalf("Consumed: %v", err)
		}

		if consumed != tt.want {
			t.Errorf("Consumed(%s, %d) = %v, want %v", tt.kind, tt.at, consumed, tt.want)
		}
	}
}

func TestSQLSessionStore_ReleaseWithoutCallbacksWritesNoCallbackRecord(t *testing.T) {
	s, _ := newSessionStoreForTest(t)
	_, id := newTestSession(t, s)

	mustTouch(t, s, id, "fm-a")

	if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: 2000}); err != nil {
		t.Fatalf("Release: %v", err)
	}

	consumed, err := s.Consumed(id, model.IMEventTypeCallback, 1)
	if err != nil {
		t.Fatalf("Consumed: %v", err)
	}

	if consumed {
		t.Error("a callback record was written for a dialog that received no callbacks")
	}
}

func TestSQLSessionStore_ReleaseByOtherNodeKeepsClaim(t *testing.T) {
	s, _ := newSessionStoreForTest(t)
	_, id := newTestSession(t, s)

	mustTouch(t, s, id, "fm-a")

	if err := s.Release(id, "fm-b", model.IMConsumedUpTo{Messages: 1000}); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if mustTouch(t, s, id, "fm-b") != nil {
		t.Error("another node's Release dropped the claim of fm-a")
	}
}

func TestSQLSessionStore_RecordNeverMovesBack(t *testing.T) {
	s, _ := newSessionStoreForTest(t)
	_, id := newTestSession(t, s)

	for _, upTo := range []int64{2000, 1000} {
		mustTouch(t, s, id, "fm-a")

		if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: upTo}); err != nil {
			t.Fatalf("Release(%d): %v", upTo, err)
		}
	}

	if !mustConsumed(t, s, id, 2000) {
		t.Error("an older Release moved the record back")
	}
}

func TestSQLSessionStore_RemoveByThreadDropsRecord(t *testing.T) {
	s, _ := newSessionStoreForTest(t)
	threadID, id := newTestSession(t, s)

	mustTouch(t, s, id, "fm-a")

	if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: 1000}); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if err := s.RemoveByThread(threadID); err != nil {
		t.Fatalf("RemoveByThread: %v", err)
	}

	if mustConsumed(t, s, id, 1000) {
		t.Error("record survived RemoveByThread")
	}
}

func TestSQLSessionStore_RemoveAllKeepsRecords(t *testing.T) {
	s, _ := newSessionStoreForTest(t)
	_, id := newTestSession(t, s)

	mustTouch(t, s, id, "fm-a")

	if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: 1000}); err != nil {
		t.Fatalf("Release: %v", err)
	}

	if err := s.RemoveAll("fm-a"); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	if !mustConsumed(t, s, id, 1000) {
		t.Error("a node restart dropped another session's record")
	}
}

func TestSQLSessionStore_PruneConsumed(t *testing.T) {
	s, db := newSessionStoreForTest(t)
	_, oldID := newTestSession(t, s)
	_, freshID := newTestSession(t, s)
	_, claimID := newTestSession(t, s)

	for _, id := range []string{oldID, freshID} {
		mustTouch(t, s, id, "fm-a")

		if err := s.Release(id, "fm-a", model.IMConsumedUpTo{Messages: 1000}); err != nil {
			t.Fatalf("Release: %v", err)
		}
	}

	// Ages the record by its release time; the event time it holds plays no part.
	if _, err := db.Exec(`update flow.session set created_at = now() - interval '2 hours' where id = $1`, consumedKey(oldID, model.IMEventTypeMessage)); err != nil {
		t.Fatalf("age record: %v", err)
	}

	mustTouch(t, s, claimID, "fm-a")

	if _, err := db.Exec(`update flow.session set created_at = now() - interval '2 hours' where id = $1`, claimID); err != nil {
		t.Fatalf("age claim: %v", err)
	}

	if err := s.PruneConsumed(time.Hour); err != nil {
		t.Fatalf("PruneConsumed: %v", err)
	}

	if mustConsumed(t, s, oldID, 1000) {
		t.Error("record older than the TTL survived the prune")
	}

	if !mustConsumed(t, s, freshID, 1000) {
		t.Error("fresh record was pruned")
	}

	if mustTouch(t, s, claimID, "fm-b") != nil {
		t.Error("prune dropped a live claim")
	}
}
