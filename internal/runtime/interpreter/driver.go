package interpreter

import (
	"context"

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
}

func NewDriver(repo persistence.Repository, reg *ops.Registry) *Driver {
	return &Driver{repo: repo, reg: reg}
}

// Run executes the flow described by rec and tr until the flow completes,
// suspends, fails, or ctx is cancelled.
//
// It writes state transitions to the repository but does NOT call Create —
// the caller is responsible for creating the record before calling Run.
func (d *Driver) Run(ctx context.Context, rec *persistence.Record, tr *tree.Tree) error {
	es := rec.State

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		action, next, err := Step(ctx, es, tr, d.reg)
		es = next

		switch action.Kind {
		case ActionContinue:
			// nothing — loop

		case ActionDone:
			rec.State = es
			rec.Status = state.StatusCompleted
			return d.repo.Complete(ctx, rec.ID)

		case ActionSuspend:
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
			rec.State = es
			rec.Status = state.StatusFailed
			return d.repo.Fail(ctx, rec.ID, reason)
		}
	}
}
