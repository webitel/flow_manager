package builtin

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// GlobalDeps is the narrow interface required by the global op.
type GlobalDeps interface {
	SetGlobalVar(ctx context.Context, domainId int64, name string, value string, encrypt bool) error
}

type globalOp struct{ deps GlobalDeps }

// GlobalOp returns the native global op: writes one or more domain-scoped schema
// variables. Each key in args is a variable name mapping to {value, encrypt}.
func GlobalOp(deps GlobalDeps) ops.Op { return globalOp{deps: deps} }

func (globalOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o globalOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	for name, raw := range in.Node.Args {
		m, ok := raw.(map[string]any)
		if !ok {
			return ops.OpOutput{}, fmt.Errorf("global: variable %q must be an object {value, encrypt}", name)
		}
		value, _ := m["value"].(string)
		value = ops.ExpandStr(value, in.Variables, in.GlobalVar)
		encrypt, _ := m["encrypt"].(bool)

		if err := o.deps.SetGlobalVar(ctx, in.DomainID, name, value, encrypt); err != nil {
			return ops.OpOutput{}, fmt.Errorf("global: set %q: %w", name, err)
		}
	}
	return ops.OpOutput{}, nil
}
