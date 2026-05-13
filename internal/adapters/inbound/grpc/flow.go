package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/webitel/flow_manager/api/gen/workflow"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

const (
	timeoutFlowSchema = 30 * time.Second // todo config ?
)

func (s *Server) DistributeAttempt(ctx context.Context, in *workflow.DistributeAttemptRequest) (*workflow.DistributeAttemptResponse, error) {
	vars := in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	conn := newConnection(utils.NewId(), in.DomainId, int(in.SchemaId), ctx, vars, 0)

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

func (s *Server) ResultAttempt(ctx context.Context, in *workflow.ResultAttemptRequest) (*workflow.ResultAttemptResponse, error) {
	vars := in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	conn := newConnection(utils.NewId(), in.DomainId, int(in.SchemaId), ctx, vars, 0)

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

func (s *Server) StartFlow(_ context.Context, in *workflow.StartFlowRequest) (*workflow.StartFlowResponse, error) {
	vars := in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}
	id := utils.NewId()

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

func (s *Server) StartSyncFlow(ctx context.Context, in *workflow.StartSyncFlowRequest) (*workflow.StartSyncFlowResponse, error) {
	vars := in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	id := utils.NewId()
	conn := newConnection(id, in.DomainId, int(in.SchemaId), ctx, vars, time.Duration(in.TimeoutSec)*time.Second)

	s.consume <- conn

	<-conn.Context().Done()

	return &workflow.StartSyncFlowResponse{
		Id: id,
	}, nil
}

func (s *Server) BotExecute(ctx context.Context, in *workflow.BotExecuteRequest) (*workflow.BotExecuteResponse, error) {
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
