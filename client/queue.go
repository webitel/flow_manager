package client

import (
	"context"
	"github.com/webitel/flow_manager/providers/grpc/workflow"
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
