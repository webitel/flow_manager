// Package ops defines the Op interface and the input/output types used by the
// interpreter. Builtin ops live in ops/builtin; legacy adapters in ops/legacy.
package ops

import (
	"context"

	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// OpKind describes how the interpreter should handle an op's lifecycle.
type OpKind int

const (
	// OpKindSync completes in one Step call.
	OpKindSync OpKind = iota
	// OpKindSuspendable may return a non-empty SuspendKey to pause execution.
	OpKindSuspendable
)

// OpInput is the read-only context passed to every Op.Execute call.
type OpInput struct {
	Node      *tree.Node
	Variables map[string]string
	DomainID  int64
	// ConnID is the identifier of the active connection (call UUID, conversation ID, etc.).
	// Empty for service/processing flows that have no associated connection.
	ConnID string
	// GlobalVar returns the domain-scoped schema variable for name.
	// Pre-bound to DomainID by the Driver; nil-safe (returns "" when nil).
	GlobalVar func(name string) string

	// ResumePayload is non-nil when this Execute is triggered by an external
	// resume event (e.g. inbound message, queue event, timer expiry).
	// Sync ops always see nil.
	ResumePayload map[string]string

	// Triggers maps trigger names (e.g. "disconnected", "commands-/cancel") to
	// their sub-tree root. Populated by the interpreter from tr.Triggers.
	// Nil when the schema declares no triggers.
	Triggers map[string]*tree.Node

	// Timezone is the IANA timezone name currently active for this flow
	// (set by the "timezone" op). Empty means UTC / system default.
	// Passed to date/time helpers in expression evaluation.
	Timezone string
}

// OpOutput carries the interpreter directives produced by one op execution.
// Only one of Branch/Goto/Break/SuspendKey should be set per call.
type OpOutput struct {
	// SetVars is merged into ExecState.Variables after execution.
	SetVars map[string]string

	// Branch is a container node to enter (used by if/while/switch).
	Branch *tree.Node

	// Repeat, when true together with Branch, causes the interpreter to
	// re-execute the current node after the branch body completes (while loop).
	Repeat bool

	// BranchAsync, when true together with Branch, forks the branch into a
	// separate goroutine instead of entering it inline. Used for trigger
	// sub-flows that must not block the main flow.
	BranchAsync bool

	// Goto is a tag name; the interpreter resets the stack to execute from
	// the tagged node on the next step.
	Goto string

	// Break exits the current execution block (maps to ActionDone for MVP).
	Break bool

	// SuspendKey is non-empty when the op needs to suspend and wait for an
	// external event identified by this key.
	SuspendKey string

	// ReSuspend, when true together with SuspendKey, signals that the op
	// consumed a resume event but still needs more events on the same key.
	// The Driver keeps the record suspended with the refreshed Pending.
	ReSuspend bool

	// Pending is a write-ahead idempotency record for suspendable ops.
	Pending *state.PendingIntent

	// ReenterOnResume, when true, causes the interpreter to back up the
	// execution position before persisting the suspend state, so this op is
	// called again on resume with OpInput.ResumePayload populated.
	// Set by suspendable ops that need to inspect the resume event themselves
	// (e.g. recvMessage for TriggerCommands). Sync ops must not set this.
	ReenterOnResume bool

	// SetTimezone, when non-empty, updates ExecState.Timezone after this op
	// executes. Use the IANA timezone name (e.g. "Europe/Kyiv").
	SetTimezone string
}

// Op is the interface every flow application must implement to run inside the
// new interpreter. Use ops/legacy to bridge existing flow.ApplicationHandler
// implementations.
type Op interface {
	Kind() OpKind
	Execute(ctx context.Context, in OpInput) (OpOutput, error)
}
