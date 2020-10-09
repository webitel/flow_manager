package grpc_route

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	flow2 "github.com/webitel/protos/workflow"
)

type DoDistributeCancelArgs struct {
	Description       string `json:"description"`
	NextDistributeSec uint32 `json:"nextDistributeSec"`
	Stop              bool   `json:"stop"`

	Export []string `json:"export"`
}

type DoDistributeConfirmArgs struct {
	Display     string `json:"display"`
	Destination string `json:"destination"`

	Export []string `json:"export"`
}

func (r *Router) cancel(ctx context.Context, scope *flow.Flow, conn model.GRPCConnection, args interface{}) (model.Response, *model.AppError) {
	var argv DoDistributeCancelArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	conn.Result(&flow2.DistributeAttemptResponse{
		Result: &flow2.DistributeAttemptResponse_Cancel_{
			Cancel: &flow2.DistributeAttemptResponse_Cancel{
				Description:       argv.Description,
				NextDistributeSec: argv.NextDistributeSec,
				Stop:              argv.Stop,
			},
		},
		Variables: exportVars(conn, argv.Export),
	})

	scope.SetCancel()

	return model.CallResponseOK, nil
}

func (r *Router) confirm(ctx context.Context, scope *flow.Flow, conn model.GRPCConnection, args interface{}) (model.Response, *model.AppError) {
	var argv DoDistributeConfirmArgs

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	conn.Result(&flow2.DistributeAttemptResponse{
		Result: &flow2.DistributeAttemptResponse_Confirm_{
			Confirm: &flow2.DistributeAttemptResponse_Confirm{
				Destination: argv.Destination,
				Display:     argv.Display,
			},
		},
		Variables: exportVars(conn, argv.Export),
	})

	scope.SetCancel()

	return model.CallResponseOK, nil
}

func exportVars(conn model.GRPCConnection, vars []string) map[string]string {
	if len(vars) == 0 {

		return nil
	}

	res := make(map[string]string)
	for _, varName := range vars {
		if val, ok := conn.Get(varName); ok {
			res[varName] = val
		}
	}

	return res
}
