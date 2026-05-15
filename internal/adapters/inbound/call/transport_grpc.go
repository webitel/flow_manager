package call

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	workflow2 "github.com/webitel/flow_manager/api/gen/workflow"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/infrastructure/utils"
)

const timeoutGrpcSchema = 30 * time.Second

// CallbackResolver handles bot-execute callbacks over gRPC.
type CallbackResolver interface {
	Callback(ctx context.Context, id string, data any) (any, error)
}

type grpcRegistrar interface {
	Register(desc *grpc.ServiceDesc, impl any)
}

// CallGrpcTransport implements FlowServiceServer and routes inbound gRPC
// call-flow requests to the Dispatcher via Consume().
type CallGrpcTransport struct {
	consume chan flow.Connection
	cb      CallbackResolver
	workflow2.UnsafeFlowServiceServer
}

// NewCallGrpcTransport creates the transport and registers it on grpcSrv.
func NewCallGrpcTransport(grpcSrv grpcRegistrar, cb CallbackResolver) *CallGrpcTransport {
	t := &CallGrpcTransport{
		consume: make(chan flow.Connection),
		cb:      cb,
	}
	grpcSrv.Register(&workflow2.FlowService_ServiceDesc, t)
	return t
}

func (t *CallGrpcTransport) Consume() <-chan flow.Connection {
	return t.consume
}

func (t *CallGrpcTransport) DistributeAttempt(ctx context.Context, in *workflow2.DistributeAttemptRequest) (*workflow2.DistributeAttemptResponse, error) {
	vars := in.Variables
	if vars == nil {
		vars = make(map[string]string)
	}

	conn := newGrpcConnection(utils.NewId(), in.DomainId, int(in.SchemaId), ctx, vars, 0)

	var result *workflow2.DistributeAttemptResponse

	t.consume <- conn

	select {
	case <-conn.ctx.Done():
		return nil, errors.New("error: server close connection")
	case <-ctx.Done():
		return nil, errors.New("ctx done")
	case r := <-conn.result:
		switch v := r.(type) {
		case *workflow2.DistributeAttemptResponse:
			result = v
		default:
			result = &workflow2.DistributeAttemptResponse{}
		}
	}

	if result != nil {
		result.Id = conn.id
	}

	return result, nil
}

func (t *CallGrpcTransport) ResultAttempt(ctx context.Context, in *workflow2.ResultAttemptRequest) (*workflow2.ResultAttemptResponse, error) {
	vars := in.Variables
	if vars == nil {
		vars = make(map[string]string)
	}

	conn := newGrpcConnection(utils.NewId(), in.DomainId, int(in.SchemaId), ctx, vars, 0)

	var result *workflow2.ResultAttemptResponse
	sc := in.GetScope()
	if sc != nil {
		conn.scope.Id = sc.Id
		conn.scope.Channel = sc.Channel
	}

	t.consume <- conn

	select {
	case <-ctx.Done():
		return nil, errors.New("ctx done")
	case <-conn.ctx.Done():
		return nil, errors.New("error: server close connection")
	case r := <-conn.result:
		switch v := r.(type) {
		case *workflow2.ResultAttemptResponse:
			result = v
		case *workflow2.DistributeAttemptResponse:
			result = &workflow2.ResultAttemptResponse{}
		default:
			result = &workflow2.ResultAttemptResponse{}
		}
	}

	if result != nil {
		result.Id = conn.id
	}

	return result, nil
}

func (t *CallGrpcTransport) StartFlow(_ context.Context, in *workflow2.StartFlowRequest) (*workflow2.StartFlowResponse, error) {
	vars := in.Variables
	if vars == nil {
		vars = make(map[string]string)
	}
	id := utils.NewId()

	conn := newGrpcConnection(id, in.DomainId, int(in.SchemaId), context.Background(), vars, 0)
	conn.id = id

	sc := in.GetScope()
	if sc != nil {
		conn.scope.Id = sc.Id
		conn.scope.Channel = sc.Channel
	}

	t.consume <- conn
	return &workflow2.StartFlowResponse{
		Id: id,
	}, nil
}

func (t *CallGrpcTransport) StartSyncFlow(ctx context.Context, in *workflow2.StartSyncFlowRequest) (*workflow2.StartSyncFlowResponse, error) {
	vars := in.Variables
	if vars == nil {
		vars = make(map[string]string)
	}

	id := utils.NewId()
	conn := newGrpcConnection(id, in.DomainId, int(in.SchemaId), ctx, vars, time.Duration(in.TimeoutSec)*time.Second)

	t.consume <- conn

	<-conn.Context().Done()

	return &workflow2.StartSyncFlowResponse{
		Id: id,
	}, nil
}

func (t *CallGrpcTransport) BotExecute(ctx context.Context, in *workflow2.BotExecuteRequest) (*workflow2.BotExecuteResponse, error) {
	res, err := t.cb.Callback(ctx, in.DialogId, in)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *workflow2.BotExecuteResponse:
		return r, nil
	default:
		return nil, errors.New("callback error")
	}
}
