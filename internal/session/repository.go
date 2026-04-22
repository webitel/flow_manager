package session

import (
	"context"
	"time"
)

// Repository is the port for persisting session checkpoints.
// Implemented by the storage layer (postgres, etc.).
type Repository interface {
	// Save persists a new checkpoint and sets its ID.
	Save(ctx context.Context, cp *Checkpoint) error

	// Update overwrites variables and updated_at for an existing checkpoint.
	Update(ctx context.Context, cp *Checkpoint) error

	// Close marks a checkpoint as closed (flow finished or connection gone).
	Close(ctx context.Context, connectionID string) error

	// ActiveByApp returns all active checkpoints owned by the given appID.
	// Used on graceful shutdown to decide what needs flushing.
	ActiveByApp(ctx context.Context, appID string) ([]*Checkpoint, error)

	// ClaimOrphaned returns active checkpoints not updated since staleDuration
	// and reassigns them to appID. Used by the recovery worker on startup.
	ClaimOrphaned(ctx context.Context, appID string, staleDuration time.Duration) ([]*Checkpoint, error)

	// Touch updates updated_at for all active checkpoints owned by appID.
	// Called periodically to prevent healthy sessions from appearing stale.
	Touch(ctx context.Context, appID string) error
}
