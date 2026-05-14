package interpreter

import (
	"context"
	"fmt"

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

	runBranch := func(bCtx context.Context, node *tree.Node, vars map[string]string) {
		go d.runBranchAsync(bCtx, node, vars, tr, domainID, rec.ConnectionID, globalVar)
	}

	l.Debug("run flow")

	// maxSyncSteps limits synchronous steps per Run invocation (i.e. between
	// two suspend/resume boundaries). It guards against infinite loops caused
	// by schemas that cycle through known ops (e.g. switch with no match)
	// which reset GotoCounter, preventing the consecutive-goto limit from
	// firing. Resets naturally on every resume because Run() is re-entered.
	const maxSyncSteps = 10_000
	var syncSteps int

	// persist uses a background context for all state-saving calls so that
	// DB writes complete even when the flow context has been cancelled or has
	// timed out. The flow context (ctx) is only used for op execution logic.
	persist := context.Background()

	stepPayload := payload
	for {
		if ctx.Err() != nil {
			// Flow context cancelled (connection dropped, gRPC deadline, etc.).
			// Mark the record as failed so it is not left stuck as "running".
			rec.State = es
			rec.Status = state.StatusFailed
			_ = d.repo.Fail(persist, rec.ID, ctx.Err().Error())
			return ctx.Err()
		}

		syncSteps++
		if syncSteps > maxSyncSteps {
			err := fmt.Errorf("flow exceeded maximum synchronous step limit (%d) — possible infinite loop", maxSyncSteps)
			rec.State = es
			rec.Status = state.StatusFailed
			return d.repo.Fail(persist, rec.ID, err.Error())
		}

		action, next, err := Step(ctx, l, es, tr, d.reg, domainID, rec.ConnectionID, globalVar, stepPayload, runBranch)
		stepPayload = nil // consumed by first Step; nil for all subsequent
		es = next

		switch action.Kind {
		case ActionContinue:
			l.Debug("flow continue")
			rec.State = es
			if err2 := d.repo.Update(persist, rec); err2 != nil {
				return err2
			}

		case ActionDone:
			l.Debug("flow completed")
			rec.State = es
			rec.Status = state.StatusCompleted
			return d.repo.Complete(persist, rec.ID)

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
			if err2 := d.repo.Update(persist, rec); err2 != nil {
				return err2
			}
			return d.repo.Suspend(persist, rec.ID, action.SuspendKey)

		case ActionBranchAsync:
			// Fire-and-forget trigger sub-flow: run branch in a goroutine
			// sharing a snapshot of current variables. Main flow then re-suspends.
			l.Debug("branch async",
				wlog.String("branch", action.AsyncBranch.ID),
				wlog.String("key", action.SuspendKey),
			)
			varSnap := make(map[string]string, len(es.Variables))
			for k, v := range es.Variables {
				varSnap[k] = v
			}
			branch := action.AsyncBranch
			go d.runBranchAsync(ctx, branch, varSnap, tr, domainID, rec.ConnectionID, globalVar)
			// Re-suspend the main flow on the same key.
			rec.State = es
			rec.Status = state.StatusSuspended
			if err2 := d.repo.Update(persist, rec); err2 != nil {
				return err2
			}
			return d.repo.Suspend(persist, rec.ID, action.SuspendKey)

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
			return d.repo.Fail(persist, rec.ID, reason)
		}
	}
}

// RunTrigger runs a named trigger sub-tree synchronously without persisting state.
// ctx should already carry any channel-specific values the ops need (e.g. the
// connection reference injected by the channel decorator). Returns nil when the
// named trigger is absent in tr. Suspend actions are treated as done — triggers
// are not resumable.
func (d *Driver) RunTrigger(ctx context.Context, tr *tree.Tree, name string, vars map[string]string, domainID int64, connID string) error {
	branch, ok := tr.Triggers[name]
	if !ok {
		return nil
	}

	var globalVar func(string) string
	if d.globals != nil {
		globalVar = func(gname string) string {
			return d.globals(ctx, domainID, gname)
		}
	}

	es := state.ExecState{
		Variables: vars,
		Stack:     []state.Frame{{NodeID: branch.ID, Position: 0}},
	}
	l := d.log.With(
		wlog.String("conn", connID),
		wlog.String("trigger", name),
	)

	runBranch := func(bCtx context.Context, node *tree.Node, bVars map[string]string) {
		go d.runBranchAsync(bCtx, node, bVars, tr, domainID, connID, globalVar)
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		action, next, err := Step(ctx, l, es, tr, d.reg, domainID, connID, globalVar, nil, runBranch)
		es = next
		switch action.Kind {
		case ActionDone:
			return nil
		case ActionFail:
			if err != nil {
				return err
			}
			return fmt.Errorf("trigger %s failed: %s", name, action.FailReason)
		case ActionSuspend:
			return nil // triggers are not resumable
		case ActionContinue, ActionBranchAsync:
			// continue loop
		}
	}
}

// runBranchAsync executes a trigger sub-tree in a goroutine without persisting
// state. Variables are a snapshot — writes do not affect the main flow.
// Errors are logged and swallowed; the goroutine is fire-and-forget.
func (d *Driver) runBranchAsync(ctx context.Context, branch *tree.Node, vars map[string]string, tr *tree.Tree, domainID int64, connID string, globalVar func(string) string) {
	es := state.ExecState{
		Variables: vars,
		Stack:     []state.Frame{{NodeID: branch.ID, Position: 0}},
	}
	l := d.log.With(wlog.String("async_branch", branch.ID))
	for {
		if ctx.Err() != nil {
			return
		}
		action, next, _ := Step(ctx, l, es, tr, d.reg, domainID, connID, globalVar, nil, nil)
		es = next
		switch action.Kind {
		case ActionDone, ActionFail, ActionSuspend:
			return
		case ActionContinue, ActionBranchAsync:
			// ActionBranchAsync inside a trigger sub-flow is not supported;
			// treat it as continue to avoid recursion.
		}
	}
}
