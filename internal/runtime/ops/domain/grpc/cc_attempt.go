// Package grpc provides native ops for the gRPC (CC attempt) channel.
package grpc

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/api/gen/workflow"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/model"
)

// Register adds all gRPC channel ops to reg.
func Register(reg *ops.Registry) {
	reg.Register("cancel", &cancelOp{})
	reg.Register("confirm", &confirmOp{})
	reg.Register("abandoned", &abandonedOp{})
	reg.Register("success", &successOp{})
	reg.Register("retry", &retryOp{})
	reg.Register("export", &exportOp{})
}

// grpcConnFromContext retrieves model.GRPCConnection from the context stored
// by connctx.WithConnection in the decorator.
func grpcConnFromContext(ctx context.Context) (model.GRPCConnection, bool) {
	conn := connctx.ConnectionFromContext(ctx)
	if conn == nil {
		return nil, false
	}
	gc, ok := conn.(model.GRPCConnection)
	return gc, ok
}

// exportVars returns a map of the requested variable names to their values
// from the flow state.
func exportVars(vars []string, variables map[string]string) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	res := make(map[string]string, len(vars))
	for _, v := range vars {
		if val, ok := variables[v]; ok {
			res[v] = val
		}
	}
	return res
}

// ── cancel ────────────────────────────────────────────────────────────────────

type cancelOp struct{}

func (cancelOp) Kind() ops.OpKind { return ops.OpKindSync }

type cancelArgs struct {
	Description                 string   `json:"description"`
	WaitBetweenRetries          int      `json:"waitBetweenRetries"`
	Stop                        bool     `json:"stop"`
	Export                      []string `json:"export"`
	ExcludeCurrentCommunication bool     `json:"excludeCurrentCommunication"`
	MinOfferingAt               *int64   `json:"minOfferingAt"`
}

func (cancelOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := grpcConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("cancel: no grpc connection in context")
	}
	var argv cancelArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	conn.Result(&workflow.DistributeAttemptResponse{
		Result: &workflow.DistributeAttemptResponse_Cancel_{
			Cancel: &workflow.DistributeAttemptResponse_Cancel{
				Description:       argv.Description,
				NextDistributeSec: uint32(argv.WaitBetweenRetries),
				Stop:              argv.Stop,
			},
		},
		Variables: exportVars(argv.Export, in.Variables),
	})
	return ops.OpOutput{}, nil
}

// ── confirm ───────────────────────────────────────────────────────────────────

type confirmOp struct{}

func (confirmOp) Kind() ops.OpKind { return ops.OpKindSync }

type confirmArgs struct {
	Display     string   `json:"display"`
	Destination string   `json:"destination"`
	Export      []string `json:"export"`
}

func (confirmOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := grpcConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("confirm: no grpc connection in context")
	}
	var argv confirmArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	conn.Result(&workflow.DistributeAttemptResponse{
		Result: &workflow.DistributeAttemptResponse_Confirm_{
			Confirm: &workflow.DistributeAttemptResponse_Confirm{
				Destination: argv.Destination,
				Display:     argv.Display,
			},
		},
		Variables: exportVars(argv.Export, in.Variables),
	})
	return ops.OpOutput{}, nil
}

// ── abandoned ─────────────────────────────────────────────────────────────────

type abandonedOp struct{}

func (abandonedOp) Kind() ops.OpKind { return ops.OpKindSync }

type abandonedArgs struct {
	Status                      string   `json:"status"`
	MaxAttempts                 uint32   `json:"maxAttempts"`
	WaitBetweenRetries          uint32   `json:"waitBetweenRetries"`
	Export                      []string `json:"export"`
	ExcludeCurrentCommunication bool     `json:"excludeCurrentCommunication"`
	Redial                      bool     `json:"redial"`
	Display                     bool     `json:"display"`
	Description                 string   `json:"description"`
	AgentId                     *int32   `json:"agentId"`
}

func (abandonedOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := grpcConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("abandoned: no grpc connection in context")
	}
	argv := abandonedArgs{Status: "abandoned"}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	ab := &workflow.ResultAttemptResponse_Abandoned{
		Status:                      argv.Status,
		MaxAttempts:                 argv.MaxAttempts,
		WaitBetweenRetries:          argv.WaitBetweenRetries,
		ExcludeCurrentCommunication: argv.ExcludeCurrentCommunication,
		Redial:                      argv.Redial,
		Display:                     argv.Display,
		Description:                 argv.Description,
	}
	if argv.AgentId != nil {
		ab.AgentId = *argv.AgentId
	}
	conn.Result(&workflow.ResultAttemptResponse{
		Result:    &workflow.ResultAttemptResponse_Abandoned_{Abandoned: ab},
		Variables: exportVars(argv.Export, in.Variables),
	})
	return ops.OpOutput{}, nil
}

// ── success ───────────────────────────────────────────────────────────────────

type successOp struct{}

func (successOp) Kind() ops.OpKind { return ops.OpKindSync }

type successArgs struct {
	Export []string `json:"export"`
}

func (successOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := grpcConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("success: no grpc connection in context")
	}
	var argv successArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	conn.Result(&workflow.ResultAttemptResponse{
		Result:    &workflow.ResultAttemptResponse_Success_{Success: &workflow.ResultAttemptResponse_Success{}},
		Variables: exportVars(argv.Export, in.Variables),
	})
	return ops.OpOutput{}, nil
}

// ── retry ─────────────────────────────────────────────────────────────────────

type retryOp struct{}

func (retryOp) Kind() ops.OpKind { return ops.OpKindSync }

type retryArgs struct {
	NextResource bool                `json:"nextResource"`
	Sleep        int32               `json:"sleep"`
	Resource     *model.SearchEntity `json:"resource"`
	Export       []string            `json:"export"`
}

func (retryOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := grpcConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("retry: no grpc connection in context")
	}
	var argv retryArgs
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, err
	}
	var resourceId int32
	if argv.Resource != nil && argv.Resource.Id != nil {
		resourceId = int32(*argv.Resource.Id)
	}
	conn.Result(&workflow.ResultAttemptResponse{
		Result: &workflow.ResultAttemptResponse_Retry_{
			Retry: &workflow.ResultAttemptResponse_Retry{
				NextResource: argv.NextResource,
				Sleep:        argv.Sleep,
				ResourceId:   resourceId,
			},
		},
		Variables: exportVars(argv.Export, in.Variables),
	})
	return ops.OpOutput{}, nil
}

// ── export ────────────────────────────────────────────────────────────────────

type exportOp struct{}

func (exportOp) Kind() ops.OpKind { return ops.OpKindSync }

func (exportOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	conn, ok := grpcConnFromContext(ctx)
	if !ok {
		return ops.OpOutput{}, fmt.Errorf("export: no grpc connection in context")
	}
	var vars []string
	switch v := in.Node.RawArgs.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				vars = append(vars, ops.ExpandStr(s, in.Variables, in.GlobalVar))
			}
		}
	}
	if _, appErr := conn.Export(ctx, vars); appErr != nil {
		return ops.OpOutput{}, fmt.Errorf("export: %s", appErr.Error())
	}
	return ops.OpOutput{}, nil
}
