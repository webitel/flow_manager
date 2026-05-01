// Package interpreter contains the Step function and the Driver that loops it.
package interpreter

import "github.com/webitel/flow_manager/internal/runtime/tree"

// ActionKind describes what the interpreter should do after a Step.
type ActionKind int

const (
	// ActionContinue means the step executed successfully; call Step again.
	ActionContinue ActionKind = iota
	// ActionSuspend means execution paused waiting for an external event.
	ActionSuspend
	// ActionDone means the flow completed normally.
	ActionDone
	// ActionFail means the flow terminated with an error.
	ActionFail
	// ActionBranchAsync means the op spawned an async trigger sub-flow; the
	// Driver must launch it in a goroutine while the main flow continues or
	// re-suspends on the same key.
	ActionBranchAsync
)

// Action is the result of one Step call.
type Action struct {
	Kind        ActionKind
	SuspendKey  string     // non-empty when Kind == ActionSuspend
	ReSuspend   bool       // true when the op re-suspends on the same key after consuming an event
	FailReason  string     // non-empty when Kind == ActionFail
	AsyncBranch *tree.Node // non-nil when Kind == ActionBranchAsync
}
