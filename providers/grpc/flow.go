package grpc

import (
	"context"
	"errors"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/workflow"
	"time"
)

const (
	timeoutFlowSchema = 30 * time.Second //todo config ?
)

func (s *server) DistributeAttempt(ctx context.Context, in *workflow.DistributeAttemptRequest) (*workflow.DistributeAttemptResponse, error) {
	var vars = in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	conn := newConnection(ctx, vars)

	var result *workflow.DistributeAttemptResponse

	conn.schemaId = int(in.SchemaId)
	conn.domainId = in.DomainId
	conn.id = model.NewId()

	s.consume <- conn

	select {
	case <-ctx.Done():
		return nil, errors.New("ctx done")
	case r := <-conn.result:
		result, _ = r.(*workflow.DistributeAttemptResponse)
	}

	return result, nil
}

func (s *server) StartFlow(_ context.Context, in *workflow.StartFlowRequest) (*workflow.StartFlowResponse, error) {
	var vars = in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}
	c, _ := context.WithTimeout(context.Background(), timeoutFlowSchema)
	conn := newConnection(c, vars)
	id := model.NewId()
	conn.id = id

	conn.schemaId = int(in.SchemaId)
	conn.domainId = in.DomainId

	s.consume <- conn
	return &workflow.StartFlowResponse{
		Id: id,
	}, nil
}
