package client

import (
	workflow "buf.build/gen/go/webitel/workflow/protocolbuffers/go"
	"context"
)

type QueueApi interface {
	DoDistributeAttempt(in *workflow.DistributeAttemptRequest) (*workflow.DistributeAttemptResponse, error)
	ResultAttempt(in *workflow.ResultAttemptRequest) (*workflow.ResultAttemptResponse, error)
	StartFlow(in *workflow.StartFlowRequest) (string, error)
	StartSyncFlow(in *workflow.StartSyncFlowRequest) (string, error)
	NewProcessing(ctx context.Context, domainId int64, schemaId int, vars map[string]string) (*QueueProcessing, error)
}

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

func (api *queueApi) ResultAttempt(in *workflow.ResultAttemptRequest) (*workflow.ResultAttemptResponse, error) {
	cli, err := api.cli.getRandomClient()
	if err != nil {
		return nil, err
	}

	return cli.queue.ResultAttempt(context.Background(), in)
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

func (api *queueApi) StartSyncFlow(in *workflow.StartSyncFlowRequest) (string, error) {
	var res *workflow.StartSyncFlowResponse
	cli, err := api.cli.getRandomClient()
	if err != nil {
		return "", err
	}

	res, err = cli.queue.StartSyncFlow(context.Background(), in)
	if err != nil {

		return "", err
	}

	return res.Id, nil
}
