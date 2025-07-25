package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/webitel/flow_manager/gen/workflow"
	"github.com/webitel/flow_manager/model"
)

const (
	timeoutFlowSchema = 30 * time.Second //todo config ?
)

func (s *server) DistributeAttempt(ctx context.Context, in *workflow.DistributeAttemptRequest) (*workflow.DistributeAttemptResponse, error) {
	var vars = in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	conn := newConnection(model.NewId(), in.DomainId, int(in.SchemaId), ctx, vars, 0)

	var result *workflow.DistributeAttemptResponse

	s.consume <- conn

	select {
	case <-conn.ctx.Done():
		return nil, errors.New("error: server close connection")
	case <-ctx.Done():
		return nil, errors.New("ctx done")
	case r := <-conn.result:
		switch v := r.(type) {
		case *workflow.DistributeAttemptResponse:
			result = v
		default:
			result = &workflow.DistributeAttemptResponse{}
		}
	}

	if result != nil {
		result.Id = conn.id
	}

	return result, nil
}

func (s *server) ResultAttempt(ctx context.Context, in *workflow.ResultAttemptRequest) (*workflow.ResultAttemptResponse, error) {
	var vars = in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	conn := newConnection(model.NewId(), in.DomainId, int(in.SchemaId), ctx, vars, 0)

	var result *workflow.ResultAttemptResponse
	sc := in.GetScope()
	if sc != nil {
		conn.scope.Id = sc.Id
		conn.scope.Channel = sc.Channel
	}

	s.consume <- conn

	select {
	case <-ctx.Done():
		return nil, errors.New("ctx done")
	case <-conn.ctx.Done():
		return nil, errors.New("error: server close connection")
	case r := <-conn.result:
		switch v := r.(type) {
		case *workflow.ResultAttemptResponse:
			result = v
		case *workflow.DistributeAttemptResponse:
			result = &workflow.ResultAttemptResponse{}
		default:
			result = &workflow.ResultAttemptResponse{}
		}
	}

	if result != nil {
		result.Id = conn.id
	}

	return result, nil
}

func (s *server) StartFlow(_ context.Context, in *workflow.StartFlowRequest) (*workflow.StartFlowResponse, error) {
	var vars = in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}
	id := model.NewId()

	conn := newConnection(id, in.DomainId, int(in.SchemaId), context.Background(), vars, 0)

	conn.id = id

	sc := in.GetScope()
	if sc != nil {
		conn.scope.Id = sc.Id
		conn.scope.Channel = sc.Channel
	}

	s.consume <- conn
	return &workflow.StartFlowResponse{
		Id: id,
	}, nil
}

func (s *server) StartSyncFlow(ctx context.Context, in *workflow.StartSyncFlowRequest) (*workflow.StartSyncFlowResponse, error) {
	var vars = in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	id := model.NewId()
	conn := newConnection(id, in.DomainId, int(in.SchemaId), ctx, vars, time.Duration(in.TimeoutSec)*time.Second)

	s.consume <- conn

	<-conn.Context().Done()

	return &workflow.StartSyncFlowResponse{
		Id: id,
	}, nil
}

func (s *server) BotExecute(ctx context.Context, in *workflow.BotExecuteRequest) (*workflow.BotExecuteResponse, error) {
	res, err := s.cb.Callback(ctx, in.DialogId, in)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *workflow.BotExecuteResponse:
		return r, nil
	default:
		return nil, errors.New("callback error")
	}
}
