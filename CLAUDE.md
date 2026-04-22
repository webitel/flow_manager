# flow_manager — Claude context

## What this project is

Routing and execution engine for calls, chats, emails, webhooks, IM, and processing flows.

## Core runtime

| Role | File |
|------|------|
| Entrypoint | `server.go` |
| Composition root | `app/app.go` |
| Dispatcher | `app/listener.go` |
| Flow parser | `flow/flow.go` |
| Flow executor | `flow/handler.go` |
| Apps registry | `flow/applications.go` |
| Control flow | `flow/{if,while,switch,goto,break}.go` |

## Channels

| Channel | Router | Apps |
|---------|--------|------|
| call | `routes/call/router.go` | `routes/call/applications.go` |
| chat | `routes/chat/router.go` | `routes/chat/applications.go` |
| email | `routes/email/router.go` | `routes/email/applications.go` |
| grpc | `routes/grpc/router.go` | `routes/grpc/applications.go` |
| processing | `routes/processing/router.go` | `routes/processing/applications.go` |
| channel | `routes/channel/router.go` | — |
| webhook | `routes/webhook/hook.go` | — |
| im | `routes/im/router.go` | — |

## Flow execution contract

1. Provider creates `model.Connection`
2. `FlowManager.Listen` dispatches to domain router
3. Router resolves schema and builds `flow.Flow`
4. `flow.Route` iterates `ApplicationRequest` nodes
5. Application handler executes, can mutate scope variables
6. Triggers may run after disconnect or flow end

Key types: `model.Connection`, `model.Router`, `flow.Flow`, `flow.ApplicationRequest`

## Active long-term plan: migration to fx + clean architecture

Strategy: **strangler-incremental** — keep old API stable, move internals phase by phase, one vertical slice at a time.

### Phase 1 (current) — bootstrap wiring
```
server.go                → cmd/flow-manager/main.go
app/config.go            → internal/bootstrap/config/
app/app.go               → internal/bootstrap/runtime/flow_manager.go
app/server.go            → internal/bootstrap/runtime/servers.go
app/listener.go          → internal/bootstrap/runtime/dispatcher.go
app/*_cli.go             → internal/adapters/outbound/clients/
app/callback.go          → internal/usecase/callback/
```

### Phase 2 — domain + adapters
```
flow/                    → internal/domain/flow/runtime/ + internal/usecase/flow/
routes/*/                → internal/adapters/inbound/*/
providers/*/             → internal/adapters/inbound/transports/*/
model/                   → internal/domain/shared/ (keep model/ as compat layer)
```

### Phase 3 — storage + messaging
```
store/pg_store/          → internal/adapters/outbound/storage/postgres/
store/cachelayer/        → internal/adapters/outbound/storage/cache/
store/store.go           → internal/domain/shared/ports/storage.go
mq/                      → internal/adapters/outbound/mq/
cases/                   → internal/adapters/outbound/cases/
pkg/processing/          → internal/domain/processing/
```

### Phase 4
```
gen/                     → api/gen/
```

### fx introduction
- **Done (narrow entry):** `cmd/flow-manager/main.go` uses fx — `app.NewFlowManager` + `flow.NewRouter` as providers, router init + lifecycle hooks as invokes. `fx.NopLogger` suppresses fx startup noise.
- Phase 2: extract `internal/bootstrap/fx/` modules for config/logger/store/mq/providers/routers
- Phase 3: bind provider consume loops via fx invokes (remove manual goroutine spawning from main)

### Session recovery (phase 2+)
Target components:
- `internal/domain/session/entity/session_state.go`
- `internal/domain/session/ports/repository.go`
- `internal/usecase/session/checkpoint.go` + `restore.go`
- `internal/workers/session_recovery/worker.go`
- `internal/adapters/outbound/storage/postgres/session_repository.go`

Existing assets: `store/pg_store/session.go`, `store/pg_store/socket_session.go`, `model/socket_session.go`

## Active iteration: session-recovery-01

**Priority:** Solve chat/IM state loss on process restart.
**Do not touch:** call/webhook/processing (stateless, no recovery needed)

| Task | Description | Status |
|------|-------------|--------|
| SR-T1 | `cmd/flow-manager/main.go` — new entrypoint, `server.go` lives as shim | ✓ done |
| SR-T2 | `internal/session/` — `Checkpoint` type + `Repository` port interface | ✓ done |
| SR-T3 | `internal/storage/postgres/checkpoint_repository.go` + `migrations/postgres/0001_session_checkpoints.sql` + wired in `FlowManager` | ✓ done |
| SR-T4 | Checkpoint hooks in `routes/chat/router.go` (save/update/close) | ✓ done |
| SR-T5 | Same hooks in `routes/im/router.go` | ✓ done |
| SR-T6 | `internal/workers/session_recovery/` — startup worker claims orphaned checkpoints and acts on them | ✓ done |

**Verification:** smoke-test: start chat flow → restart service → orphaned checkpoint exists in DB → worker logs + closes it within 90s.

## Completed iteration: fx-sqlc-01

| Task | Description | Status |
|------|-------------|--------|
| SQLC-T1 | sqlc setup: `sqlc.yaml`, `internal/storage/postgres/schema/`, `queries/session_checkpoint.sql` | ✓ done |
| SQLC-T2 | Generated `internal/storage/postgres/sqlcgen/` — typed queries via `pqtype.NullRawMessage` | ✓ done |
| SQLC-T3 | Rewrote `checkpoint_repository.go` to use generated querier; `//go:generate sqlc generate` | ✓ done |
| FX-T1 | `cmd/flow-manager/main.go` — narrow fx entry: `NewFlowManager` + `NewRouter` as providers, lifecycle hook for Listen/Shutdown | ✓ done |

**sqlc workflow:** edit `internal/storage/postgres/queries/*.sql` → `go generate ./internal/storage/postgres/` → commit both query file and `sqlcgen/`.

## Completed iteration: infra-01

| Task | Description | Status |
|------|-------------|--------|
| INFRA-T1 | Swap `mbobakov/grpc-consul-resolver` → `infra/resolver` in `app/app.go` — adds `wbt_round_robin` balancer | ✓ done |
| INFRA-T2 | `infra/consul` for cluster registration — deferred: `discovery.ServiceDiscovery` also needed for `chatManager.GetByName`/Watcher; can't replace without Phase 2 decomposition | ⏸ deferred |
| INFRA-T3 | Extract `session.Save/Update/Close` to `internal/session/hooks.go` — removes duplication from `routes/chat` and `routes/im` | ✓ done |

**infra/ package status:**
- `infra/consul` — active, used in `app/cluster.go`
- `infra/resolver` — active, blank-imported in `app/app.go`
- `infra/grpc_client` — deferred (Phase 2, replaces `engine/pkg/wbt` usages)
- `infra/grpc_srv` — deferred (Phase 2, gRPC server factory)
- `infra/sql` — deleted (incompatible with current `lib/pq`+`database/sql` stack)

## Refactoring backlog (parallel tracks)

| Track | Priority | Suggested first step |
|-------|----------|----------------------|
| flow-engine-safety | HIGH | Clarify cancellation/limiter in `flow/flow.go` + `flow/while.go` |
| router-unification | medium | Checkpoint hooks done; next: extract shared Request+AddApplications across `routes/*/` |
| observability-and-errors | medium | Standardize error codes and log fields in `flow/handler.go` |
| provider-boundaries | medium | Isolate provider concerns in `providers/*/server.go` |

**Next default session focus:** Phase 1 bootstrap wiring — `app/*_cli.go` → `internal/adapters/outbound/clients/` (one file at a time).

## Hard rules

- Backward compatibility for existing flow schemas — never break.
- Verify disconnected trigger behavior whenever touching lifecycle code.
- Small refactors one channel/router at a time.
- Verification baseline: `go test ./...` + smoke check call/chat/email flow startup.
