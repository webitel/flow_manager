package app

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"

	eng "buf.build/gen/go/webitel/engine/protocolbuffers/go"
)

func (fm *FlowManager) MakeCall(ctx context.Context, req *eng.CreateCallRequest) *model.AppError {
	if fm.engineCallCli == nil {
		return model.NewAppError("App", "MakeCall", nil, "engine client not initialized to make a call", http.StatusInternalServerError)
	}
	_, err := fm.engineCallCli.CreateCallNA(ctx, req)
	if err != nil {
		return model.NewAppError("App", "MakeCall", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func NewEngineConnection(consulAddress string) (*grpc.ClientConn, error) {
	return grpc.NewClient(fmt.Sprintf("consul://%s/engine?wait=14s", consulAddress),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

}
