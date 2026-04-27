package interpreter

import (
	"context"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// Driver loops Step until the flow finishes or suspends, persisting lifecycle
// transitions via the Repository.
type Driver struct {
	repo persistence.Repository
	reg  *ops.Registry
	log  *wlog.Logger
}

func NewDriver(repo persistence.Repository, reg *ops.Registry, log *wlog.Logger) *Driver {
	return &Driver{repo: repo, reg: reg, log: log}
}

// Run executes the flow described by rec and tr until the flow completes,
// suspends, fails, or ctx is cancelled.
//
// It writes state transitions to the repository but does NOT call Create —
// the caller is responsible for creating the record before calling Run.
func (d *Driver) Run(ctx context.Context, rec *persistence.Record, tr *tree.Tree) error {
	es := rec.State

	l := d.log.With(
		wlog.String("conn", rec.ConnectionID),
		wlog.Int("schema_id", rec.SchemaID),
	)

	l.Debug("run flow")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		action, next, err := Step(ctx, l, es, tr, d.reg)
		es = next

		switch action.Kind {
		case ActionContinue:
			rec.State = es
			if err2 := d.repo.Update(ctx, rec); err2 != nil {
				return err2
			}

		case ActionDone:
			l.Debug("flow completed")
			rec.State = es
			rec.Status = state.StatusCompleted
			return d.repo.Complete(ctx, rec.ID)

		case ActionSuspend:
			l.Debug("flow suspended",
				wlog.String("key", action.SuspendKey),
			)
			rec.State = es
			rec.Status = state.StatusSuspended
			// Update persisted state BEFORE suspending so that the external
			// event handler can safely load and resume.
			if err2 := d.repo.Update(ctx, rec); err2 != nil {
				return err2
			}
			return d.repo.Suspend(ctx, rec.ID, action.SuspendKey)

		case ActionFail:
			reason := action.FailReason
			if err != nil {
				reason = err.Error()
			}
			l.Debug("flow failed",
				wlog.String("reason", reason),
			)
			rec.State = es
			rec.Status = state.StatusFailed
			return d.repo.Fail(ctx, rec.ID, reason)
		}
	}
}
