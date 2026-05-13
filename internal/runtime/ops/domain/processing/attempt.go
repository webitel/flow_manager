package processing

import (
	"context"
	"fmt"
	"strconv"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// AttemptDeps is the subset of  attempt ops need.
type AttemptDeps interface {
	AttemptResult(result *model.AttemptResult) error
	ResumeAttempt(ctx context.Context, attemptId, domainId int64) error
}

// RegisterAttempt adds attemptResult and resumeAttempt to reg.
func RegisterAttempt(reg *ops.Registry, deps AttemptDeps) {
	reg.Register("attemptResult", &attemptResultOp{deps: deps})
	reg.Register("resumeAttempt", &resumeAttemptOp{deps: deps})
}

// ── attemptResult ─────────────────────────────────────────────────────────────

type attemptResultOp struct{ deps AttemptDeps }

func (o *attemptResultOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *attemptResultOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := connFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("attemptResult: no processing connection in context")
	}

	var argv model.AttemptResult
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	attIDStr := in.Variables["attempt_id"]
	attID, _ := strconv.ParseInt(attIDStr, 10, 64)
	argv.Id = attID

	if exportVars := conn.DumpExportVariables(); len(exportVars) > 0 {
		if argv.Variables == nil {
			argv.Variables = make(map[string]string)
		}
		for k, v := range exportVars {
			argv.Variables[k] = v
		}
	}

	if appErr := o.deps.AttemptResult(&argv); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("attemptResult: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}

// ── resumeAttempt ─────────────────────────────────────────────────────────────

type resumeAttemptOp struct{ deps AttemptDeps }

func (o *resumeAttemptOp) Kind() ops.OpKind { return ops.OpKindSync }

type resumeAttemptArgs struct {
	Id int `json:"id"`
}

func (o *resumeAttemptOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv resumeAttemptArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}

	id := int64(argv.Id)
	if id == 0 {
		attIDStr := in.Variables["attempt_id"]
		id, _ = strconv.ParseInt(attIDStr, 10, 64)
	}

	if err := o.deps.ResumeAttempt(ctx, id, in.DomainID); err != nil {
		return ops.OpOutput{}, fmt.Errorf("resumeAttempt: %w", err)
	}
	return ops.OpOutput{}, nil
}
