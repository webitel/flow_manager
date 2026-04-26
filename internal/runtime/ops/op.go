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

	// Goto is a tag name; the interpreter resets the stack to execute from
	// the tagged node on the next step.
	Goto string

	// Break exits the current execution block (maps to ActionDone for MVP).
	Break bool

	// SuspendKey is non-empty when the op needs to suspend and wait for an
	// external event identified by this key.
	SuspendKey string

	// Pending is a write-ahead idempotency record for suspendable ops.
	Pending *state.PendingIntent
}

// Op is the interface every flow application must implement to run inside the
// new interpreter. Use ops/legacy to bridge existing flow.ApplicationHandler
// implementations.
type Op interface {
	Kind() OpKind
	Execute(ctx context.Context, in OpInput) (OpOutput, error)
}
