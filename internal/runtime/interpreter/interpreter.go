package interpreter

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/wlog"
)

const maxGotoDepth = 100

// Step advances the execution state by one op and returns the resulting Action
// and the updated ExecState. The caller (Driver) loops Step until the action
// is not ActionContinue.
//
// Step is a pure function — it does not touch the database or any I/O.
func Step(ctx context.Context, log *wlog.Logger, es state.ExecState, tr *tree.Tree, reg *ops.Registry) (Action, state.ExecState, error) {
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
			if log != nil {
				log.Debug("pop frame", wlog.String("id", top.NodeID))
			}
			es.Stack = es.Stack[:len(es.Stack)-1]
			continue
		}

		child := container.Children[top.Position]
		top.Position++

		if log != nil {
			log.Debug(fmt.Sprintf("%s> %s (%s)", indent(len(es.Stack)), child.ID, child.OpName))
		}

		op := reg.Get(child.OpName)
		if op == nil {
			// Unknown op — skip silently (forward-compatibility).
			if log != nil {
				log.Debug("unknown op skipped", wlog.String("op", child.OpName))
			}
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

		// Goto: rebuild ancestor stack so the tagged node executes next and
		// execution continues past all enclosing composite ops afterwards.
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

			if log != nil {
				log.Debug("goto", wlog.String("tag", out.Goto), wlog.String("target", target.ID))
			}

			stack, buildErr := buildGotoStack(tr, target)
			if buildErr != nil {
				return Action{Kind: ActionFail, FailReason: buildErr.Error()}, es, buildErr
			}
			es.Stack = stack
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

			if log != nil {
				log.Debug("enter branch", wlog.String("id", out.Branch.ID))
			}

			es.Stack = append(es.Stack, state.Frame{NodeID: out.Branch.ID, Position: 0})
			return Action{Kind: ActionContinue}, es, nil
		}

		// Suspend: pause and wait for an external event.
		if out.SuspendKey != "" {
			es.GotoCounter = 0
			if out.Pending != nil {
				es.Pending = out.Pending
			}
			return Action{Kind: ActionSuspend, SuspendKey: out.SuspendKey}, es, nil
		}

		// Any non-goto op resets the consecutive-goto counter.
		es.GotoCounter = 0
		return Action{Kind: ActionContinue}, es, nil
	}
}

// buildGotoStack constructs the execution stack for a goto jump. It walks up
// the ancestor chain from target to the root, inserting a frame for each
// container node (OpName=="") so execution continues past enclosing composite
// ops once the target's branch is exhausted.
func buildGotoStack(tr *tree.Tree, target *tree.Node) ([]state.Frame, error) {
	var frames []state.Frame
	prev := target
	first := true

	for prev.ParentID != "" {
		parent, ok := tr.ByID[prev.ParentID]
		if !ok {
			return nil, fmt.Errorf("goto: broken tree: unknown parent %q of %q", prev.ParentID, prev.ID)
		}

		if parent.OpName == "" { // container or root
			pos := prev.SiblingIndex
			if !first {
				// Resume AFTER the intermediate composite op, not inside it.
				pos = prev.SiblingIndex + 1
			}
			frames = append([]state.Frame{{NodeID: prev.ParentID, Position: pos}}, frames...)
		}

		first = false
		prev = parent
	}

	if len(frames) == 0 {
		return nil, fmt.Errorf("goto: could not build stack for target %q", target.ID)
	}
	return frames, nil
}

func indent(depth int) string {
	if depth <= 1 {
		return ""
	}
	res := ""
	for i := 0; i < depth-1; i++ {
		res += "  "
	}
	return res
}
