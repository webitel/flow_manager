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

	// LoadByConnectionID returns the active (running or suspended) Record for
	// the given connection, or (nil, nil) if none exists.
	LoadByConnectionID(ctx context.Context, connectionID string) (*Record, error)

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

	// Touch updates updated_at for all running records owned by appID.
	// Called periodically so healthy records are not mistaken for orphans.
	Touch(ctx context.Context, appID string) error

	// ClaimOrphaned reassigns to appID all running/suspended records whose
	// updated_at is older than staleDuration and whose owner differs from
	// appID. Returns the claimed records so the caller can act on them.
	ClaimOrphaned(ctx context.Context, appID string, staleDuration time.Duration) ([]*Record, error)

	// ClaimTimerExpired claims suspended soft_sleep records for the given
	// channel whose wake_at has passed. Claimed records are transitioned to
	// running so concurrent workers cannot double-resume them.
	ClaimTimerExpired(ctx context.Context, channel int16, appID string) ([]*Record, error)
}
