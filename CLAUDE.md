# flow_manager — Claude context

## What this project is

Routing and execution engine for calls, chats, emails, webhooks, IM, and processing flows.

## Entrypoint and composition root

| Role | File |
|------|------|
| Entrypoint | `cmd/flow-manager/main.go` (fx wiring) |
| FlowManager | `internal/bootstrap/runtime/flow_manager.go` — embeds 7 outbound adapters + `*Dispatcher` |
| Dispatch loop | `internal/bootstrap/runtime/dispatcher.go` |
| Lifecycle hooks | `internal/bootstrap/fx/lifecycle.go` |
| Config loading | `internal/bootstrap/config/` |

## Channel inbound adapters

Each channel lives in `internal/adapters/inbound/<channel>/`:

| Channel | Router | Notes |
|---------|--------|-------|
| call | `internal/adapters/inbound/call/router.go` | ESL / FreeSWITCH |
| chat | `internal/adapters/inbound/chat/router.go` | gRPC chat server |
| email | `internal/adapters/inbound/email/router.go` | IMAP/SMTP |
| im | `internal/adapters/inbound/im/router.go` | IM gateway |
| grpc | `internal/adapters/inbound/grpc/router.go` | generic gRPC |
| channel | `internal/adapters/inbound/channel/router.go` | custom channel |
| processing | `internal/adapters/inbound/processing/router.go` | form processing |
| fs | `internal/adapters/inbound/fs/` | FreeSWITCH ESL transport |

Each router has a `deps.go` with a channel-specific narrow `Deps` interface.

## Flow execution contract

1. Transport creates `flow.Connection` (implements `internal/domain/flow.Connection`)
2. `Dispatcher.Listen` routes to the channel's `Router.Handle(conn)`
3. Router resolves schema via `SchemaAdapter.GetSchemaById`
4. `runtimekit.Bootstrap` builds a `Coordinator` + `Driver` pair
5. `Coordinator.Dispatch` parses schema into `tree.Tree`, runs via `interpreter.Driver`
6. `Driver` iterates `tree.Node`s, calls registered `ops.Op` handlers
7. Disconnect trigger fires via `runtimekit.Handle` after flow ends

Key types:
- `internal/domain/flow.Connection` — core connection interface
- `internal/domain/flow.Router` — `Handle(conn Connection) error`
- `internal/runtime/tree.Tree` / `.Node` — parsed schema
- `internal/runtime/ops.Op` — single operation handler
- `internal/runtime/coordinator.Coordinator` — dispatch + session recovery

## Runtime internals

```
internal/runtime/
  tree/          — JSON schema → Tree (parser + regression tests)
  interpreter/   — Driver: step execution, variable expansion
  coordinator/   — Dispatch, suspend/resume, session recovery
  ops/           — Op interface, registry, decode/expand helpers
  ops/builtin/   — httpRequest, sql, cache, global, list, generateLink, openLink
  ops/domain/    — call, chat, im, email, processing, contacts, notification, …
  runtimekit/    — Bootstrap helper (shared setup for all channel routers)
  sessionmgr/    — session watch lifecycle
  state/         — RuntimeState serialization
  persistence/   — Repository port (runtime state)
```

## Outbound adapters

```
internal/adapters/outbound/
  schema/        — SchemaAdapter: GetSchemaById, SchemaVariable, routing methods, GetSystemSettings
  store_adapter/ — StoreAdapter: store.Store delegates (media, log, queue, user, call, list, email, …)
  cc/            — CCAdapter: JoinToInboundQueue, JoinToAgent, AttemptResult, …
  cache_adapter/ — CacheAdapter: CacheGet/Set/Delete, CookieCache
  storage/       — FileAdapter: GeneratePreSignedLink, DownloadFile, GetFileTranscription
  event/         — EventBusAdapter: UserNotification, PushOpenLink, SendMQJson
  chat/          — ChatMgrAdapter: BroadcastChatMessage, SenChatAction, ParseChatMessages
  cases/         — Cases gRPC client
  contacts/      — Contacts gRPC client
  meeting/       — Meeting gRPC client
```

## Domain types

All domain types live in `internal/domain/*/` (canonical) and are re-exported from `model/` for backward compat:

| Package | Key types |
|---------|-----------|
| `internal/domain/flow/` | `Connection`, `Server`, `Router`, `Variables`, `Response`, `ApplicationRequest` |
| `internal/domain/call/` | `Call`, `CallDirection`, `PlaybackFile`, `GetSpeech`, `TTSSettings` |
| `internal/domain/routing/` | `Routing`, `Schema`, `SchemaVariable` |
| `internal/domain/chat/` | `ChatAction`, `ChatMessage`, `BroadcastChat`, `IMDialog` |
| `internal/domain/email/` | `Email`, `EmailConnection`, `EmailProfile`, `SmtSettings`, `OAuthConfig` |
| `internal/domain/queue/` | `SearchEntity`, `Member`, `CallbackMember` |
| `internal/bootstrap/config/` | `Config` and all settings structs |
| `internal/infrastructure/utils/` | `NewId`, `GetMillis`, `UrlEncoded`, `JsonString`, pointer helpers |
| `internal/infrastructure/cache/` | `ObjectCache`, `LRUCache`, `ThreadSafeStringMap` |
| `internal/infrastructure/errors/` | `StatusError` — carries HTTP status code for gRPC mapping |

`model/` is a re-export compatibility shim. New code should import from `internal/domain/*` directly.

## Workers

```
internal/workers/
  session_recovery/ — startup: claims orphaned checkpoints, closes/resumes sessions
  runtime_recovery/ — startup: recovers suspended runtime states after restart
  call_watcher/     — polls and stores hangup stats
  list_watcher/     — polls and cleans expired list communications
```

## Storage

```
internal/storage/postgres/ — pgx/v5 + squirrel repositories (no ORM)
internal/session/          — Checkpoint type + Repository port
migrations/postgres/       — goose SQL migrations (append-only, 0NNN_*.sql)
store/store.go             — legacy store.Store interface (SQL store, used by adapters)
```

## Infrastructure

```
internal/infrastructure/{resolver,discovery} — consul gRPC resolver + service discovery
internal/infrastructure/{mq,pubsub}          — RabbitMQ event bus + pub/sub
internal/infrastructure/grpcdial             — gRPC client factory
internal/infrastructure/sql                  — pgx SQL utilities
internal/infrastructure/watcher             — polling watcher (used by workers)
```

## Error handling

- All public function signatures return `error` (not `*model.AppError`)
- `model.NewAppError` is gone; use `fmt.Errorf` for internal errors
- Where HTTP status code matters (validation, not-found): `internal/infrastructure/errors.New(code, msg)`
- At gRPC boundary: `errors.As(err, &StatusError{})` → map to gRPC code via `apperrs.CodeOf(err)`

## Regression tests

`internal/runtime/tree/testdata/regression/` — real production schemas for parser regression.

```bash
# add fixtures, generate snapshots
go test ./internal/runtime/tree/... -run Regression -update

# CI: verify parser output matches snapshots
go test ./internal/runtime/tree/... -run Regression
```

## Hard rules

- Flow schema JSON format (`if`, `while`, `switch`, `goto`, `break`, `function`, `trigger`, `tag`, `limit`, `async`) — never break backward compat. Parser: `internal/runtime/tree/parser.go`.
- Disconnect trigger must fire after flow end or connection drop — verify when touching lifecycle code.
- `go build ./...` + `go test ./...` before every commit.
- One PR = one logical change. Never mix unrelated files.
- New domain types go in `internal/domain/<context>/`, not in `model/`.
- New errors: `fmt.Errorf` or `apperrs.New(code, msg)`. Never create `*model.AppError`.
- `migrations/postgres/` — only append new `0NNN_*.sql` files, never modify existing.
- `pkg/processing/` — public package imported by external services, do not move to `internal/`.
