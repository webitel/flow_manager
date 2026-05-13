package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/flow_manager/internal/domain/flow"
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/internal/session"
)

type CheckpointRepository struct {
	db infraSql.Store
}

func NewCheckpointRepository(db infraSql.Store) *CheckpointRepository {
	return &CheckpointRepository{db: db}
}

// checkpointRow is the scan target for DB rows. uuid columns are cast to text in SQL.
type checkpointRow struct {
	ID           string          `db:"id"`
	ConnectionID string          `db:"connection_id"`
	DomainID     int64           `db:"domain_id"`
	Channel      int16           `db:"channel"`
	SchemaID     int32           `db:"schema_id"`
	AppID        string          `db:"app_id"`
	Variables    json.RawMessage `db:"variables"`
	Status       string          `db:"status"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
	ClosedAt     *time.Time      `db:"closed_at"`
}

const insertCheckpointSQL = `
INSERT INTO flow.session_checkpoint
    (connection_id, domain_id, channel, schema_id, app_id, variables, status, created_at, updated_at)
VALUES (@conn_id, @domain_id, @channel, @schema_id, @app_id, @variables, @status, @created_at, @updated_at)
RETURNING id::text`

func (r *CheckpointRepository) Save(ctx context.Context, cp *session.Checkpoint) error {
	vars, err := marshalVars(cp.Variables)
	if err != nil {
		return fmt.Errorf("session.checkpoint.save: %w", err)
	}
	return r.db.Get(ctx, &cp.ID, insertCheckpointSQL, pgx.NamedArgs{
		"conn_id":    cp.ConnectionID,
		"domain_id":  cp.DomainID,
		"channel":    int16(cp.Channel),
		"schema_id":  int32(cp.SchemaID),
		"app_id":     cp.AppID,
		"variables":  vars,
		"status":     string(cp.Status),
		"created_at": cp.CreatedAt,
		"updated_at": cp.UpdatedAt,
	})
}

const updateCheckpointSQL = `
UPDATE flow.session_checkpoint
   SET variables = @variables, updated_at = @updated_at
 WHERE id = @id::uuid`

func (r *CheckpointRepository) Update(ctx context.Context, cp *session.Checkpoint) error {
	vars, err := marshalVars(cp.Variables)
	if err != nil {
		return fmt.Errorf("session.checkpoint.update: %w", err)
	}
	cp.UpdatedAt = time.Now().UTC()
	return r.db.Exec(ctx, updateCheckpointSQL, pgx.NamedArgs{
		"id":         cp.ID,
		"variables":  vars,
		"updated_at": cp.UpdatedAt,
	})
}

const closeCheckpointSQL = `
UPDATE flow.session_checkpoint
   SET status = 'closed', closed_at = now(), updated_at = now()
 WHERE connection_id = @conn_id AND status = 'active'`

func (r *CheckpointRepository) Close(ctx context.Context, connectionID string) error {
	return r.db.Exec(ctx, closeCheckpointSQL, pgx.NamedArgs{"conn_id": connectionID})
}

const touchByAppSQL = `
UPDATE flow.session_checkpoint
   SET updated_at = now()
 WHERE app_id = @app_id AND status = 'active'`

func (r *CheckpointRepository) Touch(ctx context.Context, appID string) error {
	return r.db.Exec(ctx, touchByAppSQL, pgx.NamedArgs{"app_id": appID})
}

const listActiveByAppSQL = `
SELECT id::text, connection_id, domain_id, channel, schema_id, app_id,
       variables, status, created_at, updated_at, closed_at
  FROM flow.session_checkpoint
 WHERE app_id = @app_id AND status = 'active'`

func (r *CheckpointRepository) ActiveByApp(ctx context.Context, appID string) ([]*session.Checkpoint, error) {
	var rows []checkpointRow
	if err := r.db.Select(ctx, &rows, listActiveByAppSQL, pgx.NamedArgs{"app_id": appID}); err != nil {
		return nil, err
	}
	return toCheckpoints(rows)
}

const claimOrphanedSQL = `
UPDATE flow.session_checkpoint
   SET app_id = @app_id, updated_at = now()
 WHERE status = 'active' AND updated_at < @stale_threshold
RETURNING id::text, connection_id, domain_id, channel, schema_id, app_id,
          variables, status, created_at, updated_at, closed_at`

func (r *CheckpointRepository) ClaimOrphaned(ctx context.Context, appID string, staleDuration time.Duration) ([]*session.Checkpoint, error) {
	staleThreshold := time.Now().UTC().Add(-staleDuration)
	var rows []checkpointRow
	if err := r.db.Select(ctx, &rows, claimOrphanedSQL, pgx.NamedArgs{
		"app_id":          appID,
		"stale_threshold": staleThreshold,
	}); err != nil {
		return nil, err
	}
	return toCheckpoints(rows)
}

// --- helpers ---

func toCheckpoints(rows []checkpointRow) ([]*session.Checkpoint, error) {
	out := make([]*session.Checkpoint, 0, len(rows))
	for _, row := range rows {
		cp, err := toCheckpoint(row)
		if err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	return out, nil
}

func toCheckpoint(row checkpointRow) (*session.Checkpoint, error) {
	vars, err := unmarshalVars(row.Variables)
	if err != nil {
		return nil, err
	}
	return &session.Checkpoint{
		ID:           row.ID,
		ConnectionID: row.ConnectionID,
		DomainID:     row.DomainID,
		Channel:      flow.ConnectionType(row.Channel),
		SchemaID:     int(row.SchemaID),
		AppID:        row.AppID,
		Variables:    vars,
		Status:       session.Status(row.Status),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		ClosedAt:     row.ClosedAt,
	}, nil
}

func marshalVars(vars map[string]string) (json.RawMessage, error) {
	if len(vars) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(vars)
	if err != nil {
		return nil, fmt.Errorf("marshal vars: %w", err)
	}
	return b, nil
}

func unmarshalVars(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshal vars: %w", err)
	}
	return m, nil
}
