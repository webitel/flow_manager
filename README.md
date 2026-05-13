# flow_manager

Routing and execution engine for calls, chats, emails, webhooks, IM, and processing flows.
Connects FreeSWITCH (calls), gRPC chat server (chat/IM), IMAP/SMTP (email), and custom
channel providers to a resumable JSON-based flow runtime.

## Prerequisites

- Go 1.25+
- PostgreSQL 14+
- RabbitMQ 3.x
- Consul (service discovery and registration)
- FreeSWITCH with ESL enabled (for call channel)

## Build

```bash
go build -o flow-manager ./cmd/flow-manager
```

With version/build info:

```bash
go build \
  -ldflags "-X github.com/webitel/flow_manager/internal/bootstrap/version.BuildNumber=$(git rev-parse --short HEAD)" \
  -o flow-manager ./cmd/flow-manager
```

## Configuration

Configuration is loaded in priority order: **environment variables** → **flags** → **config file** → **defaults**.

### Required

| Env | Flag | Description |
|-----|------|-------------|
| `DATA_SOURCE` | `--data_source` | PostgreSQL DSN — `postgres://user:pass@host:5432/db?sslmode=disable` |
| `CONSUL` | `--consul` | Consul address — `consul:8500` |
| `AMQP` | `--amqp` | RabbitMQ URL — `amqp://user:pass@rabbit:5672` |

### Common optional

| Env | Flag | Default | Description |
|-----|------|---------|-------------|
| `ID` | `--id` | `1` | Service instance ID |
| `LOG_LVL` | `--log_lvl` | `debug` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_JSON` | `--log_json` | `false` | JSON log format |
| `GRPC_ADDR` | `--grpc_addr` | `localhost` | gRPC server listen host |
| `GRPC_PORT` | `--grpc_port` | `0` (random) | gRPC server listen port |
| `ESL_HOST` | `--esl_host` | `localhost` | FreeSWITCH ESL host |
| `ESL_PORT` | `--esl_port` | `10030` | FreeSWITCH ESL port |
| `WEB_ADDR` | `--web_addr` | `localhost:5689` | WebHook HTTP listen address |
| `PRESIGNED_CERT` | `--presigned_cert` | `/opt/storage/key.pem` | Pre-signed link certificate |
| `ALLOW_USE_MQ` | `--allow_use_mq` | `false` | Enable MQ publishing from flows |
| `EXTERNAL_SQL` | `--external_sql` | `false` | Enable `sql` op (external DB queries) |

### Config file

Pass `--config_file=path/to/config.json` to load from JSON (overridden by env/flags).

## Running

```bash
DATA_SOURCE="postgres://postgres:postgres@localhost/webitel?sslmode=disable" \
CONSUL="localhost:8500" \
AMQP="amqp://admin:admin@localhost:5672" \
./flow-manager
```

## Database migrations

Migrations run automatically on startup. To run manually:

```bash
goose -dir migrations/postgres postgres "$DATA_SOURCE" up
```

Never edit existing migration files — append new `0NNN_*.sql` files only.

## Architecture

```
cmd/flow-manager/             — fx entrypoint (dependency wiring)
internal/
  bootstrap/
    config/                   — Config struct, env/flag loading, service constants
    runtime/                  — FlowManager lifecycle + Dispatcher (transport loops)
    fx/                       — fx providers and lifecycle hooks
    cluster/ servers/ version/
  adapters/
    inbound/                  — channel routers: call, chat, email, im, grpc, channel, processing
    outbound/                 — adapters: schema, store, cc, cache, storage, event, chat, ai
  domain/                     — canonical types: flow, call, routing, chat, email, queue, …
  runtime/
    tree/                     — JSON schema → Tree parser
    interpreter/              — step executor + variable expansion
    coordinator/              — dispatch, suspend/resume, session recovery
    ops/                      — Op interface, builtin ops, domain-specific ops
    runtimekit/               — shared Bootstrap for all channel routers
  storage/postgres/           — pgx/v5 repositories (no ORM)
  workers/                    — session_recovery, runtime_recovery, call/list watchers
  infrastructure/             — discovery, mq, grpcdial, cache, utils, errors
store/store.go                — legacy store.Store interface (SQL)
migrations/postgres/          — goose SQL migrations
pkg/processing/               — public types (imported by external services)
api/gen/                      — generated gRPC stubs
```

### Flow schema format

Flows are JSON arrays stored in `acr_routing_scheme.scheme`. Each element is a node:

```json
[
  { "tag": "start", "sendText": "Hello!", "break": false },
  {
    "if": {
      "expression": "${lang} == 'en'",
      "then": [{ "goto": "start" }],
      "else": [{ "hangup": {} }]
    }
  }
]
```

Supported control flow: `if`, `while`, `switch`, `goto`, `break`, `function`, `trigger`, `tag`, `limit`, `async`.
Common ops: `httpRequest`, `sql`, `cache`, `sendText`, `recvMessage`, `joinQueue`, `playback`, `tts`, `js`, and more.

## Testing

```bash
# all tests
go test ./...

# runtime only (fast, no external deps)
go test ./internal/runtime/...

# parser regression (requires fixtures in testdata/regression/)
go test ./internal/runtime/tree/... -run Regression

# update snapshots after intentional parser changes
go test ./internal/runtime/tree/... -run Regression -update
```

Add production schema JSON files to `internal/runtime/tree/testdata/regression/<channel>/`
and run with `-update` once to generate snapshots.

## Code generation

```bash
# regenerate gRPC stubs
buf generate

# regenerate SQL queries (sqlc)
go generate ./internal/storage/postgres/
```
