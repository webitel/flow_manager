package cases

import (
	"context"
	"fmt"

	gogrpc "buf.build/gen/go/webitel/cases/grpc/go/_gogrpc"
	cases "buf.build/gen/go/webitel/cases/protocolbuffers/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Api struct {
	cases gogrpc.CasesClient
}

type SearchCasesRequest struct {
	Token   string
	Limit   int64
	Offset  int64
	Fields  []string
	Filters map[string]string
}

func NewClient(consulTarget string) (*Api, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("consul://%s/webitel.cases?wait=14s", consulTarget),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	cases := gogrpc.NewCasesClient(conn)

	return &Api{
		cases: cases,
	}, nil
}

func (api *Api) SearchCases(ctx context.Context, req *SearchCasesRequest) (*cases.CaseList, error) {
	// Extract existing metadata from context
	existingMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		existingMD = metadata.New(nil)
	}

	// Add authorization token to metadata
	md := metadata.Join(existingMD, metadata.Pairs("X-Webitel-Access", req.Token))

	// Create a new outgoing context with the updated metadata
	newCtx := metadata.NewOutgoingContext(ctx, md)

	// Make the gRPC request
	cases, err := api.cases.SearchCases(newCtx, &cases.SearchCasesRequest{
		Size:    int32(req.Limit),
		Page:    int32(req.Offset),
		Fields:  req.Fields,
		Filters: req.Filters,
	})
	if err != nil {
		return nil, err
	}

	return cases, nil
}
