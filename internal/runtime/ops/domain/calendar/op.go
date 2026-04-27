// Package calendar provides the "calendar" op for the new interpreter.
package calendar

import (
	"context"
	"encoding/json"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

// Result mirrors the relevant fields of model.Calendar without importing model/.
type Result struct {
	Accept   bool
	Expire   bool
	Excepted *string
}

// CheckFn is the function signature injected at registration time.
type CheckFn func(ctx context.Context, domainID int64, id *int, name *string) (*Result, error)

type calendarOp struct{ check CheckFn }

// New returns an Op that evaluates a calendar and sets a flow variable.
func New(fn CheckFn) ops.Op { return calendarOp{check: fn} }

func (calendarOp) Kind() ops.OpKind { return ops.OpKindSync }

func (c calendarOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	setVar, _ := in.Node.Args["setVar"].(string)
	if setVar == "" {
		return ops.OpOutput{}, nil
	}

	extended, _ := in.Node.Args["extended"].(bool)

	var id *int
	var name *string

	if v, ok := in.Node.Args["id"]; ok && v != nil {
		switch val := v.(type) {
		case float64:
			i := int(val)
			id = &i
		case json.Number:
			if i64, err := val.Int64(); err == nil {
				i := int(i64)
				id = &i
			}
		}
	}
	if v, ok := in.Node.Args["name"].(string); ok && v != "" {
		name = &v
	}

	res, err := c.check(ctx, in.DomainID, id, name)
	if err != nil {
		return ops.OpOutput{}, err
	}

	value := "false"
	if res.Accept && !res.Expire && res.Excepted == nil {
		value = "true"
	} else if extended {
		if res.Expire {
			value = "expire"
		} else if res.Excepted != nil && *res.Excepted != "" {
			value = *res.Excepted
		}
	}

	return ops.OpOutput{SetVars: map[string]string{setVar: value}}, nil
}
