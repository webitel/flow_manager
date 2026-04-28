package im

import (
	"context"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/runtime/ops/legacy"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

const timerInterval = 5 * time.Second

// StartBackground starts goroutines that are owned by this router's lifecycle.
// ctx is cancelled when the application shuts down.
func (r *Router) StartBackground(ctx context.Context) {
	go r.runTimerWakeup(ctx)
}

func (r *Router) runTimerWakeup(ctx context.Context) {
	r.fm.Log().Info("im timer wakeup worker started")
	r.wakeExpiredTimers(ctx)

	ticker := time.NewTicker(timerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.fm.Log().Info("im timer wakeup worker stopped")
			return
		case <-ticker.C:
			r.wakeExpiredTimers(ctx)
		}
	}
}

func (r *Router) wakeExpiredTimers(ctx context.Context) {
	tCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	records, err := r.fm.RuntimeStateRepo().ClaimTimerExpired(tCtx, imChannel, r.fm.AppID())
	if err != nil {
		r.fm.Log().Warn("im timer wakeup: claim failed", wlog.Err(err))
		return
	}

	for _, rec := range records {
		r.fm.Log().Info("im timer wakeup: resuming flow",
			wlog.String("id", rec.ID.String()),
			wlog.String("connection_id", rec.ConnectionID),
			wlog.Int("schema_id", rec.SchemaID),
		)
		go r.resumeRecord(ctx, rec)
	}
}

func (r *Router) resumeRecord(ctx context.Context, rec *persistence.Record) {
	routing, appErr := r.fm.GetChatRouteFromSchemaId(rec.DomainID, int32(rec.SchemaID))
	if appErr != nil || routing == nil {
		r.fm.Log().Warn("im timer wakeup: schema not found",
			wlog.String("id", rec.ID.String()),
			wlog.Int("schema_id", rec.SchemaID),
			wlog.Int64("domain_id", rec.DomainID),
		)
		return
	}

	rawSchema := make([]map[string]any, len(routing.Schema.Schema))
	for i, app := range routing.Schema.Schema {
		rawSchema[i] = map[string]any(app)
	}
	tr, parseErr := tree.Parse(routing.SchemaId, rawSchema)
	if parseErr != nil {
		r.fm.Log().Warn("im timer wakeup: schema parse failed",
			wlog.String("id", rec.ID.String()),
			wlog.Err(parseErr),
		)
		return
	}

	// No live IM connection is available for timer wakeup. Legacy ops that
	// require a connection (e.g. sending a message) will receive nil from
	// ConnectionFromContext and must guard against it. Pure ops (log, set,
	// soft_sleep, if, etc.) work correctly without a connection.
	runCtx := legacy.WithConnection(ctx, nil)

	// For recv_message timeout, inject the timeout payload so VarFromPayload
	// can map "timeout" → the variable named in timeoutSet.
	var payload map[string]string
	if rec.State.Pending != nil && rec.State.Pending.OpName == "recv_message" {
		payload = map[string]string{"timeout": "true"}
	}

	if runErr := r.driver.Resume(runCtx, rec, tr, payload); runErr != nil {
		r.fm.Log().Warn("im timer wakeup: resume failed",
			wlog.String("id", rec.ID.String()),
			wlog.Err(runErr),
		)
	}
}
