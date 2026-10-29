//go:build integration

package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/webitel/flow_manager/model"
)

// Uses the actual migrated schema and existing isolated fixtures, never creates replacement tables.
func TestEmailLastBridgedPostgres(t *testing.T) {
	dsn, domain := os.Getenv("FLOW_TEST_POSTGRES_DSN"), os.Getenv("FLOW_EMAIL_LAST_BRIDGED_DOMAIN_ID")
	if dsn == "" || domain == "" {
		t.Skip("set FLOW_TEST_POSTGRES_DSN and FLOW_EMAIL_LAST_BRIDGED_DOMAIN_ID for an em21_step6 database with two agents and queues")
	}
	domainId, err := strconv.ParseInt(domain, 10, 64)
	if err != nil || domainId <= 0 {
		t.Fatal("invalid fixture domain")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var database string
	if err := db.QueryRow(`select current_database()`).Scan(&database); err != nil || !strings.HasPrefix(database, "em21_step6") {
		t.Fatalf("requires a disposable em21_step6 database: %q, %v", database, err)
	}
	var agents, queues pq.Int64Array
	if err := db.QueryRow(`select
    array(select id from call_center.cc_agent where domain_id = $1 order by id limit 2),
    array(select id from call_center.cc_queue where domain_id = $1 and type = 11 order by id limit 2)`, domainId).Scan(&agents, &queues); err != nil || len(agents) != 2 || len(queues) != 2 {
		t.Fatalf("requires two fixture agents and Email queues: %v, %v, %v", agents, queues, err)
	}
	wants := make([]model.Variables, len(agents))
	for i, agent := range agents {
		var extension string
		if err := db.QueryRow(`select coalesce(u.extension, '') from call_center.cc_agent a
left join directory.wbt_user u on u.id = a.user_id and u.dc = a.domain_id where a.id = $1`, agent).Scan(&extension); err != nil {
			t.Fatal(err)
		}
		wants[i] = model.Variables{"owner": fmt.Sprint(agent), "ext": extension, "queue": fmt.Sprint(queues[0])}
	}
	s := &SqlEmailStore{masterOnly{db: &gorp.DbMap{Db: db, Dialect: &PostgresJSONDialect{}}}}
	ctx := context.Background()
	fields := model.Variables{"owner": "agent_id", "ext": "extension", "queue": "queue_id"}
	empty := model.Variables{"owner": "", "ext": "", "queue": ""}
	var ids []int64
	defer func() {
		for _, table := range []string{"cc_member_attempt", "cc_member_attempt_history"} {
			if _, err := db.Exec(`delete from call_center.`+table+` where id = any($1::bigint[])`, pq.Array(ids)); err != nil {
				t.Errorf("cleanup %s: %v", table, err)
			}
		}
	}()
	baseId := -time.Now().UnixNano()
	for i, tc := range []struct {
		name, result, age                                                                            string
		active, unbridged, missingAddress, missingAgent, otherDomain, otherChannel, otherQueue, tied bool
		latest                                                                                       bool
	}{
		{name: "history", result: "processed", age: "5 minutes", latest: true},
		{name: "leaving before history", active: true, result: "processing_timeout", age: "5 minutes", latest: true},
		{name: "transfer excluded", result: "transfer", age: "5 minutes"},
		{name: "active transfer excluded", active: true, result: "transfer", age: "5 minutes"},
		{name: "other result excluded", active: true, result: "return", age: "5 minutes"},
		{name: "unbridged excluded", result: "processed", age: "5 minutes", unbridged: true},
		{name: "missing agent excluded", result: "processed", age: "5 minutes", missingAgent: true},
		{name: "missing address excluded", result: "processed", age: "5 minutes", missingAddress: true},
		{name: "domain isolated", result: "processed", age: "5 minutes", otherDomain: true},
		{name: "channel isolated", result: "processed", age: "5 minutes", otherChannel: true},
		{name: "queue filter", result: "processed", age: "5 minutes", otherQueue: true},
		{name: "hours filter", result: "processed", age: "48 hours"},
		{name: "equal timestamps ordered by id", active: true, result: "processed", age: "10 minutes", tied: true, latest: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			address := fmt.Sprintf("lookup-%d-%d@example.com", -baseId, i)
			id := baseId + int64(i*2)
			ids = append(ids, id, id+1)
			_, err := db.Exec(`insert into call_center.cc_member_attempt_history
    (id, domain_id, channel, queue_id, agent_id, result, joined_at, leaving_at, bridged_at, destination)
values ($1, $2, 'email', $3, $4, 'processed', now() - interval '1 hour', now() - interval '10 minutes', now() - interval '20 minutes', jsonb_build_object('number', $5::text))`,
				id, domainId, queues[0], agents[0], address)
			if err != nil {
				t.Fatal(err)
			}
			table, state := "cc_member_attempt_history", ""
			if tc.active {
				table, state = "cc_member_attempt", ", state, member_call_id"
			}
			dc, channel, queue, number := domainId, "email", queues[0], address
			if tc.otherDomain {
				dc++
			}
			if tc.otherChannel {
				channel = "chat"
			}
			if tc.otherQueue {
				queue = queues[1]
			}
			if tc.missingAddress {
				number = ""
			}
			suffix := ""
			if tc.active {
				suffix = ", 'leaving', $12"
			}
			var tieId any
			if tc.tied {
				tieId = id
			}
			// An explicit baseline timestamp makes the tie deterministic across transactions.
			query := `insert into call_center.` + table + `
    (id, domain_id, channel, queue_id, agent_id, result, joined_at, leaving_at, bridged_at, destination` + state + `)
select $1, $2, $3, $4, case when $10::boolean then null else $5::integer end, $6,
    now() - interval '3 days', coalesce((select leaving_at from call_center.cc_member_attempt_history where id = $11), now() - $8::interval),
    case when $9::boolean then null else now() - interval '1 hour' end,
    jsonb_build_object('number', $7::text)` + suffix
			args := []any{id + 1, dc, channel, queue, agents[1], tc.result, number, tc.age, tc.unbridged, tc.missingAgent, tieId}
			if tc.active {
				args = append(args, strconv.FormatInt(-id, 10))
			}
			if _, err := db.Exec(query, args...); err != nil {
				t.Fatal(err)
			}
			want := wants[0]
			if tc.latest {
				want = wants[1]
			}
			got, appErr := s.LastBridged(ctx, domainId, " "+strings.ToUpper(address)+" ", "24", []int{int(queues[0])}, fields)
			if appErr != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, err %v; want %v", got, appErr, want)
			}
			if tc.active {
				// Mimic the atomic move using only columns this lookup consumes.
				if _, err := db.Exec(`with moved as (delete from call_center.cc_member_attempt where id=$1 returning *)
insert into call_center.cc_member_attempt_history (id,domain_id,channel,queue_id,agent_id,result,joined_at,leaving_at,bridged_at,destination)
select id,domain_id,channel,queue_id,agent_id,result,joined_at,leaving_at,bridged_at,destination from moved`, id+1); err != nil {
					t.Fatal(err)
				}
				got, appErr = s.LastBridged(ctx, domainId, address, "24", []int{int(queues[0])}, fields)
				if appErr != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("after move: %v, %v", got, appErr)
				}
			}
			got, appErr = s.LastBridged(ctx, domainId, address, "24", []int{}, fields)
			if appErr != nil || !reflect.DeepEqual(got, empty) {
				t.Fatalf("empty queue filter: %v, %v", got, appErr)
			}
			if tc.otherQueue {
				got, appErr = s.LastBridged(ctx, domainId, address, "24", nil, fields)
				wantAnyQueue := model.Variables{"owner": wants[1]["owner"], "ext": wants[1]["ext"], "queue": fmt.Sprint(queues[1])}
				if appErr != nil || !reflect.DeepEqual(got, wantAnyQueue) {
					t.Fatalf("no queue filter: %v, %v", got, appErr)
				}
			}
		})
	}
	got, appErr := s.LastBridged(ctx, domainId, fmt.Sprintf("absent-%d@example.com", -baseId), "24", nil, fields)
	if appErr != nil || !reflect.DeepEqual(got, empty) {
		t.Fatalf("no history: %v, %v", got, appErr)
	}
	_, appErr = s.LastBridged(ctx, domainId, "valid@example.com", "invalid", nil, fields)
	if appErr == nil || appErr.Id != "store.sql_email.get_last_bridged.app_error" {
		t.Fatalf("database error hidden: %v", appErr)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, appErr = s.LastBridged(canceled, domainId, "valid@example.com", "24", nil, fields)
	if appErr == nil {
		t.Fatal("cancellation hidden")
	}
}
