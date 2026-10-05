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

There is no `body_html`. The `reply` application returns an error, and the
applications that read `cc_email`, such as email properties, do not work with
these Flows yet; the contact is not linked by the router, `wbt_contact_id`
comes from email-service.

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
