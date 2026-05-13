# flow_manager — Claude context

## What this project is

Routing and execution engine for calls, chats, emails, webhooks, IM, and processing flows.
Connects FreeSWITCH/ESL (calls), gRPC chat server (chat/IM), IMAP/SMTP (email), and
custom channel providers to a resumable flow runtime.

## Entrypoint and composition root

| Role | File |
|------|------|
| Entrypoint | `cmd/flow-manager/main.go` — fx dependency graph |
| FlowManager | `internal/bootstrap/runtime/flow_manager.go` — lifecycle coordinator (embeds 7 adapters + Dispatcher) |
| Dispatch loop | `internal/bootstrap/runtime/dispatcher.go` — goroutine-per-channel listen loops |
| Lifecycle hooks | `internal/bootstrap/fx/lifecycle.go` — fx.Lifecycle wiring |
| Config loading | `internal/bootstrap/config/` — env/flag/file loading |

## Channel inbound adapters

Each channel lives in `internal/adapters/inbound/<channel>/`:

| Channel | Transport | Router file |
|---------|-----------|-------------|
| call | FreeSWITCH ESL | `call/router.go` |
| chat | gRPC chat-server | `chat/router.go` |
| email | IMAP/SMTP | `email/router.go` |
| im | gRPC IM gateway | `im/router.go` |
| grpc | generic gRPC | `grpc/router.go` |
| channel | custom | `channel/router.go` |
| processing | gRPC forms | `processing/router.go` |
| fs | ESL transport layer | `fs/` |

Each router has a `deps.go` declaring a channel-specific narrow `Deps` interface.
`runtimekit.Bootstrap` wires the shared `Coordinator` + `Driver` from those deps.

## Flow execution contract

1. Transport creates a `flow.Connection`
2. `Dispatcher.Listen` routes to the channel's `Router.Handle(conn)`
3. Router resolves routing schema via `SchemaAdapter.GetSchemaById`
4. `runtimekit.Bootstrap` builds a `Coordinator` + `Driver` pair
5. `Coordinator.Dispatch` parses schema JSON → `tree.Tree`, drives via `interpreter.Driver`
6. `Driver` iterates `tree.Node`s, executes registered `ops.Op` handlers
7. Disconnect trigger fires via `runtimekit.Handle` after flow ends or connection drops

Key types:
- `internal/domain/flow.Connection` — core connection interface (`Set`, `Get`, `Close`, …)
- `internal/domain/flow.Router` — `Handle(conn Connection) error`
- `internal/runtime/tree.Tree` / `.Node` — parsed schema graph
- `internal/runtime/ops.Op` — single operation handler
- `internal/runtime/coordinator.Coordinator` — dispatch, suspend/resume, recovery

## Runtime internals

```
internal/runtime/
  tree/          — JSON schema parser → Tree; regression snapshots in testdata/regression/
  interpreter/   — Driver: step-by-step execution, variable expansion
  coordinator/   — Dispatch, OpKindSuspendable handling, session recovery
  ops/           — Op interface, registry, decode/expand, connctx
  ops/builtin/   — httpRequest, sql, cache, global, list, generateLink, openLink, js, …
  ops/domain/    — call, chat, im, email, processing, contacts, notification, queue, …
  runtimekit/    — Bootstrap: builds Driver + Coordinator for a channel router
  sessionmgr/    — session watch lifecycle (active connections)
  state/         — RuntimeState serialization (suspend/resume payload)
  persistence/   — Repository port for runtime state
```

## Outbound adapters

```
internal/adapters/outbound/
  schema/        — SchemaAdapter: GetSchemaById, SchemaVariable, routing, GetSystemSettings
  store_adapter/ — StoreAdapter: thin store.Store delegates (media, log, queue, user, call, …)
  cc/            — CCAdapter: JoinToInboundQueue, JoinToAgent, AttemptResult, …
  cache_adapter/ — CacheAdapter: CacheGet/Set/Delete, CookieCache
  storage/       — FileAdapter: GeneratePreSignedLink, DownloadFile, GetFileTranscription
  event/         — EventBusAdapter: UserNotification, PushOpenLink, SendMQJson
  chat/          — ChatMgrAdapter: BroadcastChatMessage, SenChatAction, ParseChatMessages
  cases/         — Cases gRPC client
  contacts/      — Contacts gRPC client
  meeting/       — Meeting gRPC client
  aibridge/      — AI bots gRPC client (STT, LLM)
```

## Domain types

All domain types live in `internal/domain/*/`:

| Package | Key types |
|---------|-----------|
| `internal/domain/flow/` | `Connection`, `Server`, `Router`, `Variables`, `Response`, `ApplicationRequest` |
| `internal/domain/call/` | `Call`, `CallDirection`, `PlaybackFile`, `GetSpeech`, `TTSSettings` |
| `internal/domain/routing/` | `Routing`, `Schema`, `SchemaVariable`, `ErrNotFoundRoute` |
| `internal/domain/chat/` | `ChatAction`, `ChatMessage`, `BroadcastChat`, `IMDialog` |
| `internal/domain/email/` | `Email`, `EmailConnection`, `EmailProfile`, `SmtSettings` |
| `internal/domain/queue/` | `SearchEntity`, `Member`, `CallbackMember` |
| `internal/domain/files/` | `File`, `SearchFile`, `FileLinkRequest` |
| `internal/bootstrap/config/` | `Config` and all settings structs, service constants |
| `internal/infrastructure/utils/` | `NewId`, `GetMillis`, `UrlEncoded`, pointer helpers |
| `internal/infrastructure/cache/` | `ObjectCache`, `LRUCache` |
| `internal/infrastructure/errors/` | `StatusError` — HTTP status code for gRPC mapping |

## Workers

```
internal/workers/
  session_recovery/ — on startup: claims orphaned checkpoints, closes/resumes sessions
  runtime_recovery/ — on startup: recovers suspended runtime states after process restart
  call_watcher/     — polls ESL for hangup stats, stores to DB
  list_watcher/     — polls and cleans expired list communications
```

## Storage

```
internal/storage/postgres/ — pgx/v5 + squirrel repositories (no ORM)
internal/session/          — Checkpoint type + Repository port
migrations/postgres/       — goose SQL migrations (0NNN_*.sql, append-only)
store/store.go             — store.Store interface (legacy SQL layer used by adapters)
```

## Infrastructure

```
internal/infrastructure/resolver   — consul gRPC resolver (blank-imported for wbt_round_robin)
internal/infrastructure/discovery  — service discovery (register + watch)
internal/infrastructure/grpcdial   — gRPC client factory
internal/infrastructure/mq         — RabbitMQ event bus
internal/infrastructure/pubsub     — pub/sub primitives
internal/infrastructure/sql        — pgx SQL utilities
internal/infrastructure/watcher    — polling watcher (used by workers)
```

## Error handling

- All public function signatures return `error`
- Internal errors: `fmt.Errorf("context: %w", err)`
- Where HTTP status code matters: `apperrs.New(http.StatusBadRequest, "msg")` (`internal/infrastructure/errors`)
- At gRPC boundary: `apperrs.CodeOf(err)` → maps to gRPC status code

## Regression tests

Parser regression: `internal/runtime/tree/testdata/regression/<channel>/<n>.json`

```bash
# generate snapshots after adding fixtures
go test ./internal/runtime/tree/... -run Regression -update

# CI: verify snapshots match
go test ./internal/runtime/tree/... -run Regression
```

## Hard rules

- **Flow schema format** (`if`, `while`, `switch`, `goto`, `break`, `function`, `trigger`, `tag`, `limit`, `async`) — never break backward compat. Parser: `internal/runtime/tree/parser.go`.
- **Disconnect trigger** must fire after flow end or connection drop — test when touching lifecycle.
- `go build ./...` + `go test ./...` before every commit.
- One PR = one logical change.
- New domain types → `internal/domain/<context>/`, not anywhere else.
- New errors → `fmt.Errorf` or `apperrs.New(code, msg)`. No typed error structs.
- `migrations/postgres/` — append-only. Never edit existing migration files.
- `pkg/processing/` — public package imported by external services, do not move to `internal/`.
- `store/store.go` — legacy interface; prefer `internal/storage/postgres/` for new repositories.
