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
	// globals resolves domain-scoped schema variables; may be nil.
	globals func(ctx context.Context, domainID int64, name string) string
}

func NewDriver(repo persistence.Repository, reg *ops.Registry, log *wlog.Logger, globals func(ctx context.Context, domainID int64, name string) string) *Driver {
	return &Driver{repo: repo, reg: reg, log: log, globals: globals}
}

// Resume transitions a suspended record back to running and continues
// execution via Run. The caller must ensure rec was loaded from the DB and
// has Status == StatusSuspended. Pending is cleared so the next Update
// persists a clean state.
func (d *Driver) Resume(ctx context.Context, rec *persistence.Record, tr *tree.Tree) error {
	rec.Status = state.StatusRunning
	rec.State.Pending = nil
	return d.Run(ctx, rec, tr)
}

// Run executes the flow described by rec and tr until the flow completes,
// suspends, fails, or ctx is cancelled.
//
// It writes state transitions to the repository but does NOT call Create —
// the caller is responsible for creating the record before calling Run.
func (d *Driver) Run(ctx context.Context, rec *persistence.Record, tr *tree.Tree) error {
	es := rec.State
	domainID := rec.DomainID

	l := d.log.With(
		wlog.String("conn", rec.ConnectionID),
		wlog.Int("schema_id", rec.SchemaID),
	)

	var globalVar func(string) string
	if d.globals != nil {
		globalVar = func(name string) string {
			return d.globals(ctx, domainID, name)
		}
	}

	l.Debug("run flow")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		action, next, err := Step(ctx, l, es, tr, d.reg, domainID, globalVar)
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
