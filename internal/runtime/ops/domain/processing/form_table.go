package processing

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
	procpkg "github.com/webitel/flow_manager/pkg/processing"
)

// RegisterFormTable adds the formTable op to reg.
func RegisterFormTable(reg *ops.Registry) {
	reg.Register("formTable", &formTableOp{})
}

type formTableOp struct{}

func (formTableOp) Kind() ops.OpKind { return ops.OpKindSync }

func (formTableOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := connFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("formTable: no processing connection in context")
	}

	var argv procpkg.FormTable
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Id == "" {
		return ops.OpOutput{}, fmt.Errorf("formTable: id is required")
	}

	// _outputs_index maps output name → index in node.Children.
	// Populated by the tree parser (parseFormTableOutputs).
	outputsIndex, _ := in.Node.Args["_outputs_index"].(map[string]int)

	// Capture RunBranch and a variable snapshot at Execute time.
	// Both are safe to use from the gRPC callback goroutine: RunBranch
	// executes sub-trees via the same driver, and varSnap is read-only.
	runBranch := in.RunBranch
	varSnap := make(map[string]string, len(in.Variables))
	for k, v := range in.Variables {
		varSnap[k] = v
	}

	argv.OutputsFn = make(map[string]procpkg.FormTableActionFn, len(outputsIndex))
	for name, idx := range outputsIndex {
		name, idx := name, idx
		if idx < 0 || idx >= len(in.Node.Children) {
			continue
		}
		branch := in.Node.Children[idx]

		argv.OutputsFn[name] = func(cbCtx context.Context, sync bool, cbVars map[string]any) error {
			// Merge callback vars on top of flow snapshot.
			merged := make(map[string]string, len(varSnap)+len(cbVars))
			for k, v := range varSnap {
				merged[k] = v
			}
			for k, v := range cbVars {
				merged[k] = fmt.Sprintf("%v", v)
			}
			// Sync connection-level variables so subsequent legacy ops see them.
			if len(cbVars) > 0 {
				connVars := make(model.Variables, len(cbVars))
				for k, v := range cbVars {
					connVars[k] = v
				}
				_, _ = conn.Set(cbCtx, connVars)
			}
			if runBranch == nil {
				return nil
			}
			if sync {
				runBranch(cbCtx, branch, merged)
			} else {
				go runBranch(cbCtx, branch, merged)
			}
			return nil
		}
	}

	conn.SetComponent(argv.Id, argv)
	return ops.OpOutput{}, nil
}
