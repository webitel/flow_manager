// Package persistence defines the port for storing and retrieving resumable
// flow execution state. The implementation lives in
// internal/storage/postgres/runtime_state_repository.go.
package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/webitel/flow_manager/internal/runtime/state"
)

// Record is one row of flow.runtime_state.
type Record struct {
	ID           uuid.UUID
	ConnectionID string
	DomainID     int64
	Channel      int16
	SchemaID     int
	AppID        string
	State        state.ExecState // schema_version lives inside State
	Status       state.Status
	ResumeKey    string
	FailReason   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	SuspendedAt  *time.Time
	CompletedAt  *time.Time
}

// Repository is the port for persistence of runtime execution state.
// Implementations must be safe for concurrent use.
type Repository interface {
	// Create persists a new Record. rec.ID is set by the implementation.
	Create(ctx context.Context, rec *Record) error

	// Load returns the Record with the given primary key.
	Load(ctx context.Context, id uuid.UUID) (*Record, error)

	// LoadByResumeKey returns the single suspended Record whose resume_key
	// matches key, or an error if none is found.
	LoadByResumeKey(ctx context.Context, key string) (*Record, error)

	// Update persists the current State and Status of rec, refreshing updated_at.
	// Use Suspend/Complete/Fail for lifecycle transitions.
	Update(ctx context.Context, rec *Record) error

	// Suspend marks the record as suspended and records the resume_key that an
	// external event must supply to resume execution.
	Suspend(ctx context.Context, id uuid.UUID, resumeKey string) error

	// Complete marks the record as completed.
	Complete(ctx context.Context, id uuid.UUID) error

	// Fail marks the record as failed and stores a human-readable reason.
	Fail(ctx context.Context, id uuid.UUID, reason string) error
}
