# flow_manager

## Email-service events

The `EmailEvents` server starts Flows for new inbound emails of email-service.
It runs only with `email_events=true` and leaves the IMAP email path unchanged.

- A consumer reads `flow.email.inbound` from the RabbitMQ of `AMQP` and stores
  each event in the Inbox, the `flow.email_event` table, before acknowledging it.
- Inbox workers claim the stored events and hand them to the email router, which
  starts the Flow of the event's `flow_id`. An event without `flow_id` is
  accepted with no Flow.

The event contract, the RabbitMQ topology and the rollout order are described in
`docs/events.md` of email-service (the `webitel-emails` repository). The Inbox
table is created by the `call_center` migration `26.dev/email_service_inbox.sql`.

### Configuration

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `email_events` | `EMAIL_EVENTS` | `false` | Enable the server |
| `email_events_prefetch` | `EMAIL_EVENTS_PREFETCH` | `20` | Unacknowledged deliveries per consumer, at most 65535 |
| `email_events_delivery_limit` | `EMAIL_EVENTS_DELIVERY_LIMIT` | `20` | Deliveries before an event goes to the dead letter queue |
| `email_events_workers` | `EMAIL_EVENTS_WORKERS` | `4` | Inbox workers per instance |
| `email_events_retention` | `EMAIL_EVENTS_RETENTION` | `720h` | How long accepted events are kept for deduplication |
| `email_events_cleanup_interval` | `EMAIL_EVENTS_CLEANUP_INTERVAL` | `1h` | Interval between Inbox cleanups |
| `email_events_cleanup_batch` | `EMAIL_EVENTS_CLEANUP_BATCH` | `1000` | Accepted events deleted per statement, at most 10000 |

Durations use Go syntax (`720h`, not `30d`). Startup fails when a value is zero,
negative or unparsable, or exceeds its limit.

### Flow variables

The variables keep the names and formats of the IMAP path: `id` (the Message ID
in email-service), `message_id`, `from`, `to`, `cc`, `sender` and `reply_to`
(addresses without display names, comma separated), `in_reply_to`, `subject`,
`body` (plain text, possibly truncated), `attachments` (JSON of stored files)
and `wbt_contact_id` (when the contact is known).

New variables: `email_event_id`, `email_profile_id`, `email_thread_id`,
`email_message_id` and `body_truncated`.
An undefined variable in these event Flows expands to `""`, not `"<nil>"`.

There is no `body_html`. The `reply` application returns an error, and the
applications that read `cc_email`, such as email properties, do not work with
these Flows yet; the contact is not linked by the router, `wbt_contact_id`
comes from email-service.

### Joining a queue

`joinQueue` puts the email-service Thread of the event into an Inbound Email
Queue of `call_center`, given as `queue.id` or `queue.name`. It returns as soon
as `call_center` creates the `waiting` attempt or finds the existing active
attempt of the Thread, and the Flow goes on with its next step. When Email
distribution is enabled, `call_center` assigns the attempt asynchronously; the
Flow neither waits for an agent nor receives queue events.

Routing arguments: `priority` is an integer `0–100` (default `0`, also for an
empty/whitespace string after variable substitution),
`extraEmailCount` allows Extra capacity (default `false`), and optional
`agent.id` or `agent.extension` supplies the Sticky candidate. An explicit ID,
including `0` or an empty Flow value (no candidate), takes precedence over
extension. Extension lookup is domain-scoped; not found means no candidate, but database errors fail the
application. Without a candidate there is no implicit history lookup.
Invalid priority or agent ID fails with `email.join_queue.valid.priority` or
`email.join_queue.valid.agent` before the join RPC; numeric Flow variables are
validated without integer truncation. Agent IDs must be in `0–2147483647`.

The join address is the trimmed, lowercased first `snapshot.from[].address`,
captured when the connection starts, not the editable `from` Flow variable.
Missing/invalid first address or `from_truncated=true` sends an empty address;
later addresses are not a fallback. Missing address does not stop the Flow.

The Thread and `email_event_id` come from the event, never from Flow variables.
`call_center` keeps one active attempt per Thread and one result per event, so a
Flow started twice for the same event, or another email of a Thread that is
already queued, creates no second attempt and moves no queue.

On success the Flow gets `cc_attempt_id`, `cc_queue_id` (the queue of the
attempt, which may differ from the requested one) and `cc_join_result`:
`created`, `already_active` or `replayed`; an `already_active` or `replayed`
attempt may already be assigned to an agent. `call_center` being unavailable is
retried up to 5 times within 25 seconds; any other error, and the IMAP path,
fail the application.

Each transport retry reuses the same normalized request, including routing
arguments and the initial address; it does not repeat extension lookup.
The CC contract is generated from local EM-21 sources using
`buf generate ../protos/cc --template buf/buf.gen.cc.yaml --path ../protos/cc/cc_member.proto`.
Deploy the compatible `call_center` schema and binary before enabling these
arguments: an older server may silently ignore the new protobuf fields.
Step 6 verification (23.10.2026): service binary build, focused unit tests,
PostgreSQL 18 integration and actual `SaveToHistory` commit/rollback races PASS.
Full `go build ./...` fails in an unchanged generated IM package; vet reports
three unchanged Flow issues. Both failures were reproduced on unmodified HEAD;
baseline acceptance remains pending. No production code was fixed during tests.
See the workspace report `tmp/Emails_Routing_Step6_Verification.md` for logs,
snapshot, scope and the remaining step-7 gates; this is not rollout approval.

Local step-7 update (2026-10-23): actual RabbitMQ → Inbox → Flow → CC join and
allocation passed defaults, priority validation, explicit candidate/Extra,
immutable From and duplicate/replay checks. Inbox `accepted` only means Flow
started: an invalid JoinQueue argument errors the node, and the existing Flow
error policy may continue to later nodes. It does not prove successful join.
The separate Online repeat required two existing CC migration functions in the
test DB; CC full-schema mismatch is still unresolved, not fixed by Flow.
Local 7.4 FS-reader tests have four header-case failures, reproduced on unchanged
HEAD; generated IM build and prior vet findings also remain baseline issues.
No full Call/Chat/IM/Task E2E or baseline waiver is claimed. Evidence is in the
workspace `tmp/Emails_Routing_Step7_2_2_Verification.md`,
`tmp/Emails_Routing_Step7_2_2_Online_Verification.md` and
`tmp/Emails_Routing_Step7_4_Verification.md`, not shipped with this repository.

### Email lastBridged

Email Flows use an Email-specific lookup, not Call history. Supply `number`
(the email address), required `hours` (the existing hours-string format), optional
`filter.queue_ids`, and `set`, e.g. `{"sticky_agent":"agent_id","sticky_ext":"extension","previous_queue":"queue_id"}`.
Only these three result fields are supported; unknown fields return
`store.sql_email.get_last_bridged.invalid_field`; Chat selectors such as `id`,
`description` and `variables.*` cannot be reused in Email subflows. Values are strings; no match
or an invalid/empty normalized address clears the requested variables to `""`.
Omitted queue IDs mean all queues; an explicit empty array matches none.

One parameterized primary-database query combines completed `leaving` attempts
and history (`UNION ALL`) using domain, `channel='email'`, `destination.number`,
the time window on `leaving_at`, queue IDs, non-null bridge/agent and results
`processed`/`processing_timeout`. The final order is `leaving_at DESC, id DESC`;
Transfer is excluded in both sources. A deleted or wrong-domain agent in the
latest attempt yields no candidate; agent status/capacity is checked later by
`call_center`, not by history lookup. Database errors/cancellation are not
treated as no match (`store.sql_email.get_last_bridged.app_error`).
For a valid lookup address, empty/invalid `hours` remains a database error,
not a default period or a validation error; no new hours parser is added.

The Flow must explicitly pass the lookup result to `joinQueue.agent`; the lookup
does not change routing or replace the immutable join address. Call/Chat/IM
lookup behavior is unchanged. Flow adds no schema migration or index itself.
Narrowed 7.3 measured actual lookup SQL on 50,000 synthetic history rows: about
17–24 ms without a queue filter and 5–8 ms with two queues, all shared-cache hits.
Without the filter about 43,000 history rows were examined; this is not an
address-indexed or bounded lookup and not proof of production scalability.
See workspace `tmp/Emails_Routing_Step7_3_Verification.md`; that measured snapshot
had no address indexes. The later L4 follow-up adds two address indexes in
call_center's full schema and `26.dev/email_last_bridged_indexes.sql`, without
changing this query. Prior cloud results support the index definitions; the
local follow-up was not rerun at the user's request. Deploy using the CC runbook.
Before rollout, the remaining gate includes `EXPLAIN (ANALYZE, BUFFERS)`
on representative history with and without queue IDs, including no match.
Pass `filter.queue_ids` when the intended lookup is limited to those queues;
do not narrow its business scope solely for speed. This query runs only when
the Flow calls `lastBridged`, not automatically for every inbound email.

The focused PostgreSQL test uses the existing `integration` tag,
`FLOW_TEST_POSTGRES_DSN` and `FLOW_EMAIL_LAST_BRIDGED_DOMAIN_ID`. It requires a
disposable `em21_step6*` database with the actual migrated schema, two agents and
two type-11 queues in that domain. It removes only rows it inserted. Its atomic
move does not test actual `SaveToHistory`; the separate verification run did,
including controlled commit/rollback concurrency. Real Email completion remains
a stage-11 lifecycle check. Test rows and the temporary gate were removed, the
isolated database preserved and its container stopped.

### Operations

An Inbox row is `pending` until a worker claims it, `processing` while a worker
holds its lease, `accepted` once its Flow started (or it has no `flow_id`), and
`failed` when it can never start: an unsupported event, or a schema still
missing after 5 launches. Failed rows are kept until removed by hand:

```sql
SELECT event_id, domain_id, thread_id, message_id, flow_id, attempts, last_error, updated_at
FROM flow.email_event
WHERE state = 'failed'
ORDER BY updated_at DESC;
```

Events that could not be stored at all are in the `flow.email.inbound.dlq`
queue. A lease of a stopped worker expires after a minute, and another worker
takes the event over.
