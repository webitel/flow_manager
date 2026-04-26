package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	infraSql "github.com/webitel/flow_manager/infra/sql"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/state"
)

// RuntimeStateRepository implements persistence.Repository backed by pgx.
type RuntimeStateRepository struct {
	db infraSql.Store
}

func NewRuntimeStateRepository(db infraSql.Store) *RuntimeStateRepository {
	return &RuntimeStateRepository{db: db}
}

// runtimeStateRow is the pgx scan target. state column is raw JSON.
type runtimeStateRow struct {
	ID            string          `db:"id"`
	ConnectionID  string          `db:"connection_id"`
	DomainID      int64           `db:"domain_id"`
	Channel       int16           `db:"channel"`
	SchemaID      int32           `db:"schema_id"`
	SchemaVersion int64           `db:"schema_version"`
	AppID         string          `db:"app_id"`
	State         json.RawMessage `db:"state"`
	Status        string          `db:"status"`
	ResumeKey     *string         `db:"resume_key"`
	FailReason    *string         `db:"fail_reason"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
	SuspendedAt   *time.Time      `db:"suspended_at"`
	CompletedAt   *time.Time      `db:"completed_at"`
}

const insertRuntimeStateSQL = `
INSERT INTO flow.runtime_state
    (connection_id, domain_id, channel, schema_id, schema_version, app_id,
     state, status, created_at, updated_at)
VALUES
    (@conn_id, @domain_id, @channel, @schema_id, @schema_version, @app_id,
     @state, @status, @created_at, @updated_at)
RETURNING id::text`

func (r *RuntimeStateRepository) Create(ctx context.Context, rec *Record) error {
	stateJSON, err := json.Marshal(rec.State)
	if err != nil {
		return fmt.Errorf("runtime_state.create: marshal state: %w", err)
	}
	now := time.Now().UTC()
	rec.CreatedAt = now
	rec.UpdatedAt = now

	var rawID string
	err = r.db.Get(ctx, &rawID, insertRuntimeStateSQL, pgx.NamedArgs{
		"conn_id":        rec.ConnectionID,
		"domain_id":      rec.DomainID,
		"channel":        rec.Channel,
		"schema_id":      int32(rec.SchemaID),
		"schema_version": int64(rec.State.SchemaVersion),
		"app_id":         rec.AppID,
		"state":          stateJSON,
		"status":         string(rec.Status),
		"created_at":     now,
		"updated_at":     now,
	})
	if err != nil {
		return fmt.Errorf("runtime_state.create: %w", err)
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("runtime_state.create: parse uuid %q: %w", rawID, err)
	}
	rec.ID = id
	return nil
}

// selectFields is the column list used in all SELECT queries.
const selectFields = `
    id::text, connection_id, domain_id, channel, schema_id, schema_version,
    app_id, state, status, resume_key, fail_reason,
    created_at, updated_at, suspended_at, completed_at
`

const loadByIDSQL = `SELECT` + selectFields + `FROM flow.runtime_state WHERE id = @id::uuid`

func (r *RuntimeStateRepository) Load(ctx context.Context, id uuid.UUID) (*Record, error) {
	var row runtimeStateRow
	err := r.db.Get(ctx, &row, loadByIDSQL, pgx.NamedArgs{"id": id.String()})
	if err != nil {
		return nil, fmt.Errorf("runtime_state.load(%s): %w", id, err)
	}
	return toRecord(row)
}

const loadByResumeKeySQL = `
SELECT` + selectFields + `
  FROM flow.runtime_state
 WHERE resume_key = @key AND status = 'suspended'
 LIMIT 1`

func (r *RuntimeStateRepository) LoadByResumeKey(ctx context.Context, key string) (*Record, error) {
	var row runtimeStateRow
	err := r.db.Get(ctx, &row, loadByResumeKeySQL, pgx.NamedArgs{"key": key})
	if err != nil {
		return nil, fmt.Errorf("runtime_state.load_by_resume_key(%s): %w", key, err)
	}
	return toRecord(row)
}

const updateRuntimeStateSQL = `
UPDATE flow.runtime_state
   SET state      = @state,
       status     = @status,
       updated_at = @updated_at
 WHERE id = @id::uuid`

func (r *RuntimeStateRepository) Update(ctx context.Context, rec *Record) error {
	stateJSON, err := json.Marshal(rec.State)
	if err != nil {
		return fmt.Errorf("runtime_state.update: marshal state: %w", err)
	}
	rec.UpdatedAt = time.Now().UTC()
	err = r.db.Exec(ctx, updateRuntimeStateSQL, pgx.NamedArgs{
		"id":         rec.ID.String(),
		"state":      stateJSON,
		"status":     string(rec.Status),
		"updated_at": rec.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("runtime_state.update(%s): %w", rec.ID, err)
	}
	return nil
}

const suspendRuntimeStateSQL = `
UPDATE flow.runtime_state
   SET status       = 'suspended',
       resume_key   = @resume_key,
       suspended_at = now(),
       updated_at   = now()
 WHERE id = @id::uuid`

func (r *RuntimeStateRepository) Suspend(ctx context.Context, id uuid.UUID, resumeKey string) error {
	err := r.db.Exec(ctx, suspendRuntimeStateSQL, pgx.NamedArgs{
		"id":         id.String(),
		"resume_key": resumeKey,
	})
	if err != nil {
		return fmt.Errorf("runtime_state.suspend(%s): %w", id, err)
	}
	return nil
}

const completeRuntimeStateSQL = `
UPDATE flow.runtime_state
   SET status       = 'completed',
       completed_at = now(),
       updated_at   = now()
 WHERE id = @id::uuid`

func (r *RuntimeStateRepository) Complete(ctx context.Context, id uuid.UUID) error {
	err := r.db.Exec(ctx, completeRuntimeStateSQL, pgx.NamedArgs{"id": id.String()})
	if err != nil {
		return fmt.Errorf("runtime_state.complete(%s): %w", id, err)
	}
	return nil
}

const failRuntimeStateSQL = `
UPDATE flow.runtime_state
   SET status      = 'failed',
       fail_reason = @reason,
       updated_at  = now()
 WHERE id = @id::uuid`

func (r *RuntimeStateRepository) Fail(ctx context.Context, id uuid.UUID, reason string) error {
	err := r.db.Exec(ctx, failRuntimeStateSQL, pgx.NamedArgs{
		"id":     id.String(),
		"reason": reason,
	})
	if err != nil {
		return fmt.Errorf("runtime_state.fail(%s): %w", id, err)
	}
	return nil
}

// --- helpers ---

// Record is a package-level alias to avoid repeating the import path in
// this file. The canonical type is persistence.Record.
type Record = persistence.Record

func toRecord(row runtimeStateRow) (*Record, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parse uuid %q: %w", row.ID, err)
	}

	var execState state.ExecState
	if err := json.Unmarshal(row.State, &execState); err != nil {
		return nil, fmt.Errorf("unmarshal state for %s: %w", row.ID, err)
	}

	rec := &Record{
		ID:           id,
		ConnectionID: row.ConnectionID,
		DomainID:     row.DomainID,
		Channel:      row.Channel,
		SchemaID:     int(row.SchemaID),
		AppID:        row.AppID,
		State:        execState,
		Status:       state.Status(row.Status),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		SuspendedAt:  row.SuspendedAt,
		CompletedAt:  row.CompletedAt,
	}
	if row.ResumeKey != nil {
		rec.ResumeKey = *row.ResumeKey
	}
	if row.FailReason != nil {
		rec.FailReason = *row.FailReason
	}
	return rec, nil
}
