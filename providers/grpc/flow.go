package grpc

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
	"github.com/webitel/protos/workflow"
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

	conn := newConnection(ctx, vars)

	var result *workflow.ResultAttemptResponse

	conn.schemaId = int(in.SchemaId)
	conn.domainId = in.DomainId
	conn.id = model.NewId()

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

func (s *server) StartSyncFlow(ctx context.Context, in *workflow.StartSyncFlowRequest) (*workflow.StartSyncFlowResponse, error) {
	var vars = in.Variables

	if vars == nil {
		vars = make(map[string]string)
	}

	var timeout = 30

	if in.TimeoutSec != 0 {
		timeout = int(in.TimeoutSec)
	}
	c, _ := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	conn := newConnection(c, vars)
	id := model.NewId()
	conn.id = id

	conn.schemaId = int(in.SchemaId)
	conn.domainId = in.DomainId

	s.consume <- conn

	<-conn.Context().Done()

	return &workflow.StartSyncFlowResponse{
		Id: id,
	}, nil
}

func (s *server) TestMem() {
	wg := sync.WaitGroup{}

	h := func(wg *sync.WaitGroup, s *server) {
		wg.Add(1)
		s.StartFlow(context.Background(), &workflow.StartFlowRequest{
			SchemaId:  635,
			DomainId:  1,
			Variables: map[string]string{"A": "A"},
		})
		wg.Done()
	}

	for i := 0; i < 1000; i++ {
		for y := 0; y < 50; y++ {
			go h(&wg, s)
		}
		time.Sleep(time.Millisecond * 500)
	}

	wg.Wait()
	wlog.Error("STOP TEST")
}
