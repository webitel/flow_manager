//go:generate sqlc generate

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/sqlc-dev/pqtype"

	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/internal/storage/postgres/sqlcgen"
	"github.com/webitel/flow_manager/model"
)

// dbGetter is the minimal interface satisfied by *sqlstore.SqlSupplier.
type dbGetter interface {
	GetMaster() *gorp.DbMap
}

type CheckpointRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewCheckpointRepository(provider dbGetter) *CheckpointRepository {
	db := provider.GetMaster().Db
	return &CheckpointRepository{db: db, q: sqlcgen.New(db)}
}

// Migrate creates the session_checkpoint table if it does not exist.
// Safe to call on every startup.
func (r *CheckpointRepository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS flow.session_checkpoint
		(
		    id            uuid        NOT NULL DEFAULT gen_random_uuid(),
		    connection_id text        NOT NULL,
		    domain_id     bigint      NOT NULL,
		    channel       smallint    NOT NULL,
		    schema_id     int         NOT NULL,
		    app_id        text        NOT NULL,
		    variables     jsonb,
		    status        text        NOT NULL DEFAULT 'active',
		    created_at    timestamptz NOT NULL DEFAULT now(),
		    updated_at    timestamptz NOT NULL DEFAULT now(),
		    closed_at     timestamptz,
		    CONSTRAINT flow_session_checkpoint_pkey PRIMARY KEY (id)
		);
		CREATE INDEX IF NOT EXISTS flow_session_checkpoint_conn_idx
		    ON flow.session_checkpoint (connection_id);
		CREATE INDEX IF NOT EXISTS flow_session_checkpoint_app_active_idx
		    ON flow.session_checkpoint (app_id, updated_at)
		    WHERE status = 'active';
	`)
	return err
}

func (r *CheckpointRepository) Save(ctx context.Context, cp *session.Checkpoint) error {
	vars, err := marshalVars(cp.Variables)
	if err != nil {
		return fmt.Errorf("session.checkpoint.save: %w", err)
	}
	id, err := r.q.InsertCheckpoint(ctx, sqlcgen.InsertCheckpointParams{
		ConnectionID: cp.ConnectionID,
		DomainID:     cp.DomainID,
		Channel:      int16(cp.Channel),
		SchemaID:     int32(cp.SchemaID),
		AppID:        cp.AppID,
		Variables:    vars,
		Status:       string(cp.Status),
		CreatedAt:    cp.CreatedAt,
		UpdatedAt:    cp.UpdatedAt,
	})
	if err != nil {
		return err
	}
	cp.ID = id
	return nil
}

func (r *CheckpointRepository) Update(ctx context.Context, cp *session.Checkpoint) error {
	vars, err := marshalVars(cp.Variables)
	if err != nil {
		return fmt.Errorf("session.checkpoint.update: %w", err)
	}
	cp.UpdatedAt = time.Now().UTC()
	return r.q.UpdateCheckpointVars(ctx, sqlcgen.UpdateCheckpointVarsParams{
		ID:        cp.ID,
		Variables: vars,
		UpdatedAt: cp.UpdatedAt,
	})
}

func (r *CheckpointRepository) Close(ctx context.Context, connectionID string) error {
	return r.q.CloseCheckpoint(ctx, connectionID)
}

func (r *CheckpointRepository) Touch(ctx context.Context, appID string) error {
	return r.q.TouchByApp(ctx, appID)
}

func (r *CheckpointRepository) ActiveByApp(ctx context.Context, appID string) ([]*session.Checkpoint, error) {
	rows, err := r.q.ListActiveByApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	return toCheckpoints(rows)
}

func (r *CheckpointRepository) ClaimOrphaned(ctx context.Context, appID string, staleDuration time.Duration) ([]*session.Checkpoint, error) {
	staleThreshold := time.Now().UTC().Add(-staleDuration)
	rows, err := r.q.ClaimOrphaned(ctx, sqlcgen.ClaimOrphanedParams{
		AppID:     appID,
		UpdatedAt: staleThreshold,
	})
	if err != nil {
		return nil, err
	}
	return toCheckpoints(rows)
}

// --- helpers ---

func toCheckpoints(rows []sqlcgen.FlowSessionCheckpoint) ([]*session.Checkpoint, error) {
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

func toCheckpoint(row sqlcgen.FlowSessionCheckpoint) (*session.Checkpoint, error) {
	vars, err := unmarshalVars(row.Variables)
	if err != nil {
		return nil, err
	}
	cp := &session.Checkpoint{
		ID:           row.ID,
		ConnectionID: row.ConnectionID,
		DomainID:     row.DomainID,
		Channel:      model.ConnectionType(row.Channel),
		SchemaID:     int(row.SchemaID),
		AppID:        row.AppID,
		Variables:    vars,
		Status:       session.Status(row.Status),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	if row.ClosedAt.Valid {
		cp.ClosedAt = &row.ClosedAt.Time
	}
	return cp, nil
}

func marshalVars(vars map[string]string) (pqtype.NullRawMessage, error) {
	if len(vars) == 0 {
		return pqtype.NullRawMessage{}, nil
	}
	b, err := json.Marshal(vars)
	if err != nil {
		return pqtype.NullRawMessage{}, fmt.Errorf("marshal vars: %w", err)
	}
	return pqtype.NullRawMessage{RawMessage: b, Valid: true}, nil
}

func unmarshalVars(v pqtype.NullRawMessage) (map[string]string, error) {
	if !v.Valid || len(v.RawMessage) == 0 {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(v.RawMessage, &m); err != nil {
		return nil, fmt.Errorf("unmarshal vars: %w", err)
	}
	return m, nil
}
