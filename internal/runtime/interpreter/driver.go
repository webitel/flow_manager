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
//
// payload is the event data from the external trigger (e.g. inbound message
// body, queue event fields). It is passed as OpInput.ResumePayload to the
// first op that executes after the resume.
func (d *Driver) Resume(ctx context.Context, rec *persistence.Record, tr *tree.Tree, payload map[string]string) error {
	rec.Status = state.StatusRunning
	// Apply payload→variable mappings declared by the suspending op before
	// clearing Pending, so the next op sees the resolved variables immediately.
	if len(payload) > 0 && rec.State.Pending != nil {
		for payloadKey, varName := range rec.State.Pending.VarFromPayload {
			if varName == "" {
				continue
			}
			if val, ok := payload[payloadKey]; ok {
				if rec.State.Variables == nil {
					rec.State.Variables = make(map[string]string)
				}
				rec.State.Variables[varName] = val
			}
		}
	}
	rec.State.Pending = nil
	return d.Run(ctx, rec, tr, payload)
}

// Run executes the flow described by rec and tr until the flow completes,
// suspends, fails, or ctx is cancelled.
//
// payload is passed as OpInput.ResumePayload on the first Step call only;
// subsequent iterations receive nil. Pass nil for a fresh (non-resumed) run.
//
// It writes state transitions to the repository but does NOT call Create —
// the caller is responsible for creating the record before calling Run.
func (d *Driver) Run(ctx context.Context, rec *persistence.Record, tr *tree.Tree, payload map[string]string) error {
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

	stepPayload := payload
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		action, next, err := Step(ctx, l, es, tr, d.reg, domainID, globalVar, stepPayload)
		stepPayload = nil // consumed by first Step; nil for all subsequent
		es = next

		switch action.Kind {
		case ActionContinue:
			l.Debug("flow continue")
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
			if action.ReSuspend {
				l.Debug("flow re-suspended",
					wlog.String("key", action.SuspendKey),
				)
			} else {
				l.Debug("flow suspended",
					wlog.String("key", action.SuspendKey),
				)
			}
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
