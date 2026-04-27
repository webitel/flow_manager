package runtime_recovery

import (
	"context"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/runtime/persistence"
)

const (
	defaultInterval = 30 * time.Second
	defaultStale    = 90 * time.Second
)

// Worker periodically heartbeats running runtime records and claims orphaned
// ones left behind by crashed or restarted instances.
type Worker struct {
	repo     persistence.Repository
	appID    string
	interval time.Duration
	stale    time.Duration
	log      *wlog.Logger
}

func New(repo persistence.Repository, appID string, log *wlog.Logger) *Worker {
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
	w.log.Info("runtime recovery worker started", wlog.String("app_id", w.appID))

	w.claimAndFail(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("runtime recovery worker stopped")
			return
		case <-ticker.C:
			w.heartbeat(ctx)
			w.claimAndFail(ctx)
		}
	}
}

func (w *Worker) heartbeat(ctx context.Context) {
	tCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := w.repo.Touch(tCtx, w.appID); err != nil {
		w.log.Warn("runtime recovery: heartbeat failed", wlog.Err(err))
	}
}

func (w *Worker) claimAndFail(ctx context.Context) {
	tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	orphans, err := w.repo.ClaimOrphaned(tCtx, w.appID, w.stale)
	if err != nil {
		w.log.Warn("runtime recovery: claim orphaned failed", wlog.Err(err))
		return
	}

	for _, rec := range orphans {
		w.log.Warn("runtime recovery: failing orphaned record",
			wlog.String("id", rec.ID.String()),
			wlog.String("connection_id", rec.ConnectionID),
			wlog.Int64("domain_id", rec.DomainID),
			wlog.Int("channel", int(rec.Channel)),
			wlog.String("status", string(rec.Status)),
		)
		failCtx, failCancel := context.WithTimeout(ctx, 5*time.Second)
		if err := w.repo.Fail(failCtx, rec.ID, "orphaned by app restart"); err != nil {
			w.log.Warn("runtime recovery: fail orphaned record failed",
				wlog.String("id", rec.ID.String()),
				wlog.Err(err),
			)
		}
		failCancel()
	}
}
