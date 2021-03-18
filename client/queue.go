package client

import (
	"context"
	// "github.com/webitel/flow_manager/providers/grpc/workflow"
	"github.com/webitel/protos/workflow"
)

type queueApi struct {
	cli *flowManager
}

func NewQueueApi(m *flowManager) QueueApi {
	return &queueApi{
		cli: m,
	}
}

func (api *queueApi) DoDistributeAttempt(in *workflow.DistributeAttemptRequest) (*workflow.DistributeAttemptResponse, error) {
	cli, err := api.cli.getRandomClient()
	if err != nil {
		return nil, err
	}

	return cli.queue.DistributeAttempt(context.Background(), in)
}

func (api *queueApi) StartFlow(in *workflow.StartFlowRequest) (string, error) {
	var res *workflow.StartFlowResponse
	cli, err := api.cli.getRandomClient()
	if err != nil {
		return "", err
	}

	res, err = cli.queue.StartFlow(context.Background(), in)
	if err != nil {

		return "", err
	}

	return res.Id, nil
}
