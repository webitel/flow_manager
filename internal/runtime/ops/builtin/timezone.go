package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type timezoneArgs struct {
	Name *string `json:"name"`
	Id   *int    `json:"id"`
}

type timezoneOp struct {
	getLocation func(id int) *time.Location
}

// TimezoneOp returns an op that sets the active timezone for the flow.
// getLocation resolves a timezone by DB id; pass nil to disable id-based lookup.
func TimezoneOp(getLocation func(id int) *time.Location) ops.Op {
	return &timezoneOp{getLocation: getLocation}
}

func (o *timezoneOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *timezoneOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv timezoneArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	var loc *time.Location

	if argv.Id != nil && o.getLocation != nil {
		loc = o.getLocation(*argv.Id)
	} else if argv.Name != nil {
		var err error
		loc, err = time.LoadLocation(*argv.Name)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("timezone: load %q: %w", *argv.Name, err)
		}
	}

	if loc == nil {
		return ops.OpOutput{}, fmt.Errorf("timezone: location not found")
	}

	return ops.OpOutput{SetTimezone: loc.String()}, nil
}
