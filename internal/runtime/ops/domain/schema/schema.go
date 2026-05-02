// Package schema provides the "schema" op — loads and executes a flow schema
// by ID as an inline or asynchronous sub-flow.
package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/webitel/flow_manager/internal/runtime/interpreter"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/state"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// LoadTreeFn loads and parses a flow schema by (domainID, schemaID).
type LoadTreeFn func(ctx context.Context, domainID int64, schemaID int) (*tree.Tree, error)

type schemaOp struct {
	loadTree LoadTreeFn
	reg      *ops.Registry
}

// New returns an Op that loads a schema by ID and executes it as a sub-flow.
// Sync mode (default): suspends/resumes together with the parent flow so
// suspendable ops inside the sub-schema (e.g. recvMessage) work correctly.
// Async mode (async: true): fire-and-forget goroutine; suspension is not
// supported — the goroutine exits on the first ActionSuspend.
func New(load LoadTreeFn, reg *ops.Registry) ops.Op {
	return &schemaOp{loadTree: load, reg: reg}
}

func (s *schemaOp) Kind() ops.OpKind { return ops.OpKindSuspendable }

type schemaArgs struct {
	Id    any  `json:"id"`
	Async bool `json:"async"`
}

func (s *schemaOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv schemaArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	schemaID, err := toInt(argv.Id)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("schema: id: %w", err)
	}
	if schemaID == 0 {
		return ops.OpOutput{}, fmt.Errorf("schema: id is required")
	}

	subTr, loadErr := s.loadTree(ctx, in.DomainID, schemaID)
	if loadErr != nil {
		return ops.OpOutput{}, fmt.Errorf("schema %d: %w", schemaID, loadErr)
	}
	// Inherit parent triggers: ops inside the sub-schema (e.g. recvMessage)
	// must be able to fire trigger commands declared in the calling schema.
	// Sub-schema triggers take priority if the name collides.
	tr := withParentTriggers(subTr, in.Triggers)

	if argv.Async {
		varSnap := maps.Clone(in.Variables)
		gv := in.GlobalVar
		reg := s.reg
		domainID := in.DomainID
		connID := in.ConnID
		go func() {
			subES := state.ExecState{
				SchemaID:  tr.SchemaID,
				Variables: varSnap,
				Stack:     []state.Frame{{NodeID: tr.Root.ID, Position: 0}},
			}
			for ctx.Err() == nil {
				action, next, _ := interpreter.Step(ctx, nil, subES, tr, reg, domainID, connID, gv, nil)
				subES = next
				switch action.Kind {
				case interpreter.ActionDone, interpreter.ActionFail, interpreter.ActionSuspend, interpreter.ActionBranchAsync:
					return
				}
			}
		}()
		return ops.OpOutput{}, nil
	}

	// subStateKey holds the JSON-encoded sub-schema ExecState in the outer
	// variables while the sub-schema is suspended.
	subStateKey := "__sub_" + in.Node.ID

	var subES state.ExecState
	var stepPayload map[string]string

	if in.ResumePayload != nil && in.Variables[subStateKey] != "" {
		// Resume path: decode persisted sub-schema state.
		if jsonErr := json.Unmarshal([]byte(in.Variables[subStateKey]), &subES); jsonErr != nil {
			return ops.OpOutput{}, fmt.Errorf("schema %d: decode sub-state: %w", schemaID, jsonErr)
		}
		// Apply VarFromPayload into sub-schema variables (mirrors driver.Resume logic).
		if subES.Pending != nil {
			for payloadKey, varName := range subES.Pending.VarFromPayload {
				if varName != "" {
					if val, ok := in.ResumePayload[payloadKey]; ok {
						if subES.Variables == nil {
							subES.Variables = make(map[string]string)
						}
						subES.Variables[varName] = val
					}
				}
			}
			subES.Pending = nil
		}
		stepPayload = in.ResumePayload
	} else {
		// Fresh start: clone parent variables into sub-schema.
		subES = state.ExecState{
			SchemaID:  tr.SchemaID,
			Variables: maps.Clone(in.Variables),
			Stack:     []state.Frame{{NodeID: tr.Root.ID, Position: 0}},
		}
	}

	for ctx.Err() == nil {
		action, next, stepErr := interpreter.Step(ctx, nil, subES, tr, s.reg, in.DomainID, in.ConnID, in.GlobalVar, stepPayload)
		stepPayload = nil
		subES = next

		switch action.Kind {
		case interpreter.ActionContinue:
			// keep looping

		case interpreter.ActionDone:
			outVars := subES.Variables
			outVars[subStateKey] = "" // clear sub-state from outer variables
			return ops.OpOutput{SetVars: outVars}, nil

		case interpreter.ActionFail:
			if stepErr != nil {
				return ops.OpOutput{}, fmt.Errorf("schema %d: %w", schemaID, stepErr)
			}
			return ops.OpOutput{}, fmt.Errorf("schema %d: %s", schemaID, action.FailReason)

		case interpreter.ActionSuspend:
			return s.suspendOutput(subStateKey, subES, action.SuspendKey, action.ReSuspend)

		case interpreter.ActionBranchAsync:
			// Fire trigger branch as goroutine, then suspend on the same key.
			varSnap := maps.Clone(subES.Variables)
			branch := action.AsyncBranch
			reg := s.reg
			gv := in.GlobalVar
			domainID := in.DomainID
			connID := in.ConnID
			go func() {
				brES := state.ExecState{
					Variables: varSnap,
					Stack:     []state.Frame{{NodeID: branch.ID, Position: 0}},
				}
				for ctx.Err() == nil {
					a, nb, _ := interpreter.Step(ctx, nil, brES, tr, reg, domainID, connID, gv, nil)
					brES = nb
					switch a.Kind {
					case interpreter.ActionDone, interpreter.ActionFail, interpreter.ActionSuspend, interpreter.ActionBranchAsync:
						return
					}
				}
			}()
			// subES already has position backed up (Step handled ReenterOnResume).
			return s.suspendOutput(subStateKey, subES, action.SuspendKey, action.ReSuspend)
		}
	}

	return ops.OpOutput{}, ctx.Err()
}

func (s *schemaOp) suspendOutput(subStateKey string, subES state.ExecState, suspendKey string, reSuspend bool) (ops.OpOutput, error) {
	subJSON, jsonErr := json.Marshal(subES)
	if jsonErr != nil {
		return ops.OpOutput{}, fmt.Errorf("schema: encode sub-state: %w", jsonErr)
	}
	return ops.OpOutput{
		SetVars:         map[string]string{subStateKey: string(subJSON)},
		SuspendKey:      suspendKey,
		ReSuspend:       reSuspend,
		ReenterOnResume: true,
		Pending:         &state.PendingIntent{OpName: "schema"},
	}, nil
}

// withParentTriggers returns a shallow copy of subTr whose Triggers and ByID
// maps include the parent schema's triggers. Sub-schema triggers take priority
// over parent triggers on name collision. All nodes reachable from parent
// trigger roots are registered in ByID so Step can find them when executing
// trigger branches inside the sub-schema context.
func withParentTriggers(subTr *tree.Tree, parentTriggers map[string]*tree.Node) *tree.Tree {
	if len(parentTriggers) == 0 {
		return subTr
	}
	merged := *subTr // shallow copy — safe to mutate maps
	merged.ByID = make(map[tree.NodeID]*tree.Node, len(subTr.ByID)+len(parentTriggers)*8)
	maps.Copy(merged.ByID, subTr.ByID)

	merged.Triggers = make(map[string]*tree.Node, len(parentTriggers)+len(subTr.Triggers))
	maps.Copy(merged.Triggers, parentTriggers)
	maps.Copy(merged.Triggers, subTr.Triggers) // sub overrides parent

	for _, node := range parentTriggers {
		registerNodes(merged.ByID, node)
	}
	return &merged
}

// registerNodes walks the node tree and adds every node to byID.
func registerNodes(byID map[tree.NodeID]*tree.Node, node *tree.Node) {
	if node == nil {
		return
	}
	byID[node.ID] = node
	for _, child := range node.Children {
		registerNodes(byID, child)
	}
}

func toInt(v any) (int, error) {
	switch x := v.(type) {
	case int:
		return x, nil
	case int64:
		return int(x), nil
	case float64:
		return int(x), nil
	case string:
		var n int
		if _, err := fmt.Sscanf(x, "%d", &n); err != nil {
			return 0, fmt.Errorf("not a number: %q", x)
		}
		return n, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}
