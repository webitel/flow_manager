package session_recovery

import (
	"context"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/session"
)

const (
	defaultInterval = 30 * time.Second
	defaultStale    = 90 * time.Second
)

// Worker periodically heartbeats active checkpoints and claims orphaned ones
// left behind by crashed or restarted instances.
type Worker struct {
	repo     session.Repository
	appID    string
	interval time.Duration
	stale    time.Duration
	log      *wlog.Logger
}

func New(repo session.Repository, appID string, log *wlog.Logger) *Worker {
	return &Worker{
		repo:     repo,
		appID:    appID,
		interval: defaultInterval,
		stale:    defaultStale,
		log:      log,
	}
}

// Run blocks until ctx is cancelled. Call in a goroutine.
func (w *Worker) Run(ctx context.Context) {
	w.log.Info("session recovery worker started", wlog.String("app_id", w.appID))

	w.claimAndClose(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("session recovery worker stopped")
			return
		case <-ticker.C:
			w.heartbeat(ctx)
			w.claimAndClose(ctx)
		}
	}
}

func (w *Worker) heartbeat(ctx context.Context) {
	tCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := w.repo.Touch(tCtx, w.appID); err != nil {
		w.log.Warn("session recovery: heartbeat failed", wlog.Err(err))
	}
}

func (w *Worker) claimAndClose(ctx context.Context) {
	tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	orphans, err := w.repo.ClaimOrphaned(tCtx, w.appID, w.stale)
	if err != nil {
		w.log.Warn("session recovery: claim orphaned failed", wlog.Err(err))
		return
	}

	for _, cp := range orphans {
		w.log.Warn("session recovery: closing orphaned checkpoint",
			wlog.String("connection_id", cp.ConnectionID),
			wlog.Int64("domain_id", cp.DomainID),
			wlog.Int("channel", int(cp.Channel)),
		)
		closeCtx, closeCancel := context.WithTimeout(ctx, 5*time.Second)
		if err := w.repo.Close(closeCtx, cp.ConnectionID); err != nil {
			w.log.Warn("session recovery: close orphaned checkpoint failed",
				wlog.String("connection_id", cp.ConnectionID),
				wlog.Err(err),
			)
		}
		closeCancel()
	}
}
