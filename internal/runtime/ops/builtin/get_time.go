package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/internal/runtime/ops"
)

type getTimeArgs struct {
	// Set is the variable name that will receive the formatted timestamp.
	Set string `json:"set"`
	// Timezone is an IANA timezone name (e.g. "Europe/Kyiv"). Supports
	// ${var} expansion. Falls back to the flow-level timezone, then UTC.
	Timezone string `json:"timezone"`
}

type getTimeOp struct{}

// GetTime returns an op that writes the current time in the requested timezone
// to a flow variable. Format: RFC3339 (e.g. "2006-01-02T15:04:05+03:00").
func GetTime() ops.Op { return getTimeOp{} }

func (getTimeOp) Kind() ops.OpKind { return ops.OpKindSync }

func (getTimeOp) Execute(_ context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv getTimeArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	if argv.Set == "" {
		return ops.OpOutput{}, nil
	}

	// Priority: op-level timezone → flow-level timezone → UTC.
	tzName := argv.Timezone
	if tzName == "" {
		tzName = in.Timezone
	}

	loc := time.UTC
	if tzName != "" {
		var err error
		loc, err = time.LoadLocation(tzName)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("getTime: unknown timezone %q: %w", tzName, err)
		}
	}

	ts := time.Now().In(loc).Format(time.RFC3339)
	return ops.OpOutput{
		SetVars: map[string]string{argv.Set: ts},
	}, nil
}
