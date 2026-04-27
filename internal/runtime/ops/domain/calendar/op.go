// Package calendar provides the "calendar" op for the new interpreter.
package calendar

import (
	"context"

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

type calendarArgs struct {
	Name     *string `json:"name"`
	Id       *int    `json:"id"`
	SetVar   string  `json:"setVar"`
	Extended bool    `json:"extended"`
}

func (calendarOp) Doc() ops.OpDoc {
	return ops.OpDoc{
		Description: "Checks business hours using a Webitel calendar. Writes result to a session variable.",
		AvailableIn: []string{"voice", "chat", "form", "service"},
		Visual:      true,
		Args: map[string]ops.ArgDoc{
			"name":     {Type: "string", Description: "Calendar name (mutually exclusive with id)."},
			"id":       {Type: "integer", Description: "Calendar ID in Webitel."},
			"setVar":   {Type: "string", Required: true, Description: "Variable to store result."},
			"extended": {Type: "boolean", Default: false, Description: "Extended mode: returns 'expire' or exception name instead of plain 'false'."},
		},
		Notes: []string{
			"setVar stores 'true' or 'false' — compare explicitly: ${is_work_time} == 'true'.",
		},
		Examples: map[string]ops.Example{
			"basic": {
				Description: "Check business hours and branch",
				Schema: `{"calendar": {"name": "My Calendar", "setVar": "is_work_time"}},` +
					`{"if": {"expression": "${is_work_time} == 'true'", "then": [...], "else": [...]}}`,
			},
		},
	}
}

func (c calendarOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv calendarArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	if argv.SetVar == "" {
		return ops.OpOutput{}, nil
	}

	res, err := c.check(ctx, in.DomainID, argv.Id, argv.Name)
	if err != nil {
		return ops.OpOutput{}, err
	}

	value := "false"
	if res.Accept && !res.Expire && res.Excepted == nil {
		value = "true"
	} else if argv.Extended {
		if res.Expire {
			value = "expire"
		} else if res.Excepted != nil && *res.Excepted != "" {
			value = *res.Excepted
		}
	}

	return ops.OpOutput{SetVars: map[string]string{argv.SetVar: value}}, nil
}
