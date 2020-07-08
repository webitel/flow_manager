package client

import (
	"context"
	"github.com/webitel/flow_manager/providers/grpc/flow"
)

type queueApi struct {
	cli *flowManager
}

func NewQueueApi(m *flowManager) QueueApi {
	return &queueApi{
		cli: m,
	}
}

func (api *queueApi) DoDistributeAttempt(in *flow.DistributeAttemptRequest) (*flow.DistributeAttemptResponse, error) {
	cli, err := api.cli.getRandomClient()
	if err != nil {
		return nil, err
	}

	return cli.queue.DistributeAttempt(context.Background(), in)
}
