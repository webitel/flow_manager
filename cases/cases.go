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
	cases             gogrpc.CasesClient
	caseCommunication gogrpc.CaseCommunicationsClient
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
	caseCommunication := gogrpc.NewCaseCommunicationsClient(conn)

	return &Api{
		cases:             cases,
		caseCommunication: caseCommunication,
	}, nil
}

func (api *Api) SearchCases(ctx context.Context, req *cases.SearchCasesRequest, token string) (*cases.CaseList, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)

	// Make the gRPC request
	cases, err := api.cases.SearchCases(newCtx, req)
	if err != nil {
		return nil, err
	}

	return cases, nil
}

func (api *Api) LocateCase(ctx context.Context, req *cases.LocateCaseRequest, token string) (*cases.Case, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)

	// Make the gRPC request
	c, err := api.cases.LocateCase(newCtx, req)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (api *Api) CreateCase(ctx context.Context, req *cases.CreateCaseRequest, token string) (*cases.Case, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)

	// Make the gRPC request
	c, err := api.cases.CreateCase(newCtx, req)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (api *Api) UpdateCase(ctx context.Context, req *cases.UpdateCaseRequest, token string) (*cases.Case, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)

	// Make the gRPC request
	c, err := api.cases.UpdateCase(newCtx, req)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (api *Api) LinkCommunication(ctx context.Context, req *cases.LinkCommunicationRequest, token string) (*cases.LinkCommunicationResponse, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)

	// Make the gRPC request
	c, err := api.caseCommunication.LinkCommunication(newCtx, req)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// attachToken adds the authentication token to the gRPC metadata.
func attachToken(ctx context.Context, token string) context.Context {
	existingMD, _ := metadata.FromIncomingContext(ctx)
	md := metadata.Join(existingMD, metadata.Pairs("X-Webitel-Access", token))
	return metadata.NewOutgoingContext(ctx, md)
}
