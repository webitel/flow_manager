package grpc_route

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	flow2 "github.com/webitel/protos/workflow"
)

type DoDistributeCancelArgs struct {
	Description        string `json:"description"`
	WaitBetweenRetries int    `json:"waitBetweenRetries"`
	Stop               bool   `json:"stop"`

	Export                      []string `json:"export"`
	ExcludeCurrentCommunication bool     `json:"excludeCurrentCommunication"`
	MinOfferingAt               *int64   `json:"minOfferingAt"`
}

type DoDistributeConfirmArgs struct {
	Display     string `json:"display"`
	Destination string `json:"destination"`

	Export []string `json:"export"`
}

type AfterAttemptSuccess struct {
	Export []string `json:"export"`
}

type AfterAttemptAbandoned struct {
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

type AfterAttemptRetry struct {
	NextResource bool                `json:"nextResource"`
	Sleep        int32               `json:"sleep"`
	Resource     *model.SearchEntity `json:"resource"`
	Export       []string            `json:"export"`
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
				NextDistributeSec: uint32(argv.WaitBetweenRetries),
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

func (r *Router) success(ctx context.Context, scope *flow.Flow, conn model.GRPCConnection, args interface{}) (model.Response, *model.AppError) {
	var argv AfterAttemptSuccess

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	conn.Result(&flow2.ResultAttemptResponse{
		Result: &flow2.ResultAttemptResponse_Success_{
			Success: &flow2.ResultAttemptResponse_Success{},
		},
		Variables: exportVars(conn, argv.Export),
	})

	scope.SetCancel()

	return model.CallResponseOK, nil
}

func (r *Router) abandoned(ctx context.Context, scope *flow.Flow, conn model.GRPCConnection, args interface{}) (model.Response, *model.AppError) {
	var argv = AfterAttemptAbandoned{
		Status: "abandoned",
	}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	abandoned := &flow2.ResultAttemptResponse_Abandoned{
		Status:                      argv.Status,
		MaxAttempts:                 argv.MaxAttempts,
		WaitBetweenRetries:          argv.WaitBetweenRetries,
		ExcludeCurrentCommunication: argv.ExcludeCurrentCommunication,
		Redial:                      argv.Redial,
		AgentId:                     0,
		Display:                     argv.Display,
		Description:                 argv.Description,
	}

	if argv.AgentId != nil {
		abandoned.AgentId = *argv.AgentId
	}

	conn.Result(&flow2.ResultAttemptResponse{
		Result: &flow2.ResultAttemptResponse_Abandoned_{
			Abandoned: abandoned,
		},
		Variables: exportVars(conn, argv.Export),
	})

	scope.SetCancel()

	return model.CallResponseOK, nil
}

func (r *Router) retry(ctx context.Context, scope *flow.Flow, conn model.GRPCConnection, args interface{}) (model.Response, *model.AppError) {
	var argv = AfterAttemptRetry{}

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	var resourceId int32
	if argv.Resource != nil && argv.Resource.Id != nil {
		resourceId = int32(*argv.Resource.Id)
	}

	retry := &flow2.ResultAttemptResponse{
		Result: &flow2.ResultAttemptResponse_Retry_{
			Retry: &flow2.ResultAttemptResponse_Retry{
				NextResource: argv.NextResource,
				Sleep:        argv.Sleep,
				ResourceId:   resourceId,
			},
		},
		Variables: exportVars(conn, argv.Export),
	}

	conn.Result(retry)

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
