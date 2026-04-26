package interpreter

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

const maxGotoDepth = 100

// Step advances the execution state by one op and returns the resulting Action
// and the updated ExecState. The caller (Driver) loops Step until the action
// is not ActionContinue.
//
// Step is a pure function — it does not touch the database or any I/O.
func Step(ctx context.Context, es state.ExecState, tr *tree.Tree, reg *ops.Registry) (Action, state.ExecState, error) {
	for {
		if len(es.Stack) == 0 {
			return Action{Kind: ActionDone}, es, nil
		}

		top := &es.Stack[len(es.Stack)-1]

		container, ok := tr.ByID[top.NodeID]
		if !ok {
			err := fmt.Errorf("interpreter: unknown node %q in stack", top.NodeID)
			return Action{Kind: ActionFail, FailReason: err.Error()}, es, err
		}

		if top.Position >= len(container.Children) {
			// Container exhausted — pop frame and keep looping.
			es.Stack = es.Stack[:len(es.Stack)-1]
			continue
		}

		child := container.Children[top.Position]
		top.Position++

		op := reg.Get(child.OpName)
		if op == nil {
			// Unknown op — skip silently (forward-compatibility).
			return Action{Kind: ActionContinue}, es, nil
		}

		out, err := op.Execute(ctx, ops.OpInput{
			Node:      child,
			Variables: es.Variables,
		})
		if err != nil {
			reason := err.Error()
			return Action{Kind: ActionFail, FailReason: reason}, es, err
		}

		// Merge new variables.
		if len(out.SetVars) > 0 {
			if es.Variables == nil {
				es.Variables = make(map[string]string, len(out.SetVars))
			}
			for k, v := range out.SetVars {
				es.Variables[k] = v
			}
		}

		// Goto: reset stack so the tagged node executes next.
		if out.Goto != "" {
			target, ok := tr.ByTag[out.Goto]
			if !ok {
				err := fmt.Errorf("interpreter: goto: unknown tag %q", out.Goto)
				return Action{Kind: ActionFail, FailReason: err.Error()}, es, err
			}
			es.GotoCounter++
			if es.GotoCounter > maxGotoDepth {
				err := fmt.Errorf("interpreter: goto depth limit %d exceeded", maxGotoDepth)
				return Action{Kind: ActionFail, FailReason: err.Error()}, es, err
			}
			// The tagged node lives at target.SiblingIndex inside target.ParentID.
			// Set Position to SiblingIndex so it is the next node executed.
			es.Stack = []state.Frame{
				{NodeID: target.ParentID, Position: target.SiblingIndex},
			}
			return Action{Kind: ActionContinue}, es, nil
		}

		// Break: stop execution (matches legacy flow.SetCancel behaviour).
		if out.Break {
			es.Stack = nil
			return Action{Kind: ActionDone}, es, nil
		}

		// Branch: enter a sub-tree (if/while/switch).
		if out.Branch != nil {
			if out.Repeat {
				// While loop: undo the position increment so the while node
				// is re-evaluated after the branch body completes.
				es.Stack[len(es.Stack)-1].Position--
			}
			es.Stack = append(es.Stack, state.Frame{NodeID: out.Branch.ID, Position: 0})
			return Action{Kind: ActionContinue}, es, nil
		}

		// Suspend: pause and wait for an external event.
		if out.SuspendKey != "" {
			if out.Pending != nil {
				es.Pending = out.Pending
			}
			return Action{Kind: ActionSuspend, SuspendKey: out.SuspendKey}, es, nil
		}

		return Action{Kind: ActionContinue}, es, nil
	}
}
