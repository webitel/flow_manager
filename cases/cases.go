package cases

import (
	"context"
	"fmt"

	"github.com/webitel/flow_manager/gen/cases"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Api struct {
	cases             cases.CasesClient
	caseCommunication cases.CaseCommunicationsClient
	serviceCatalogs   cases.CatalogsClient
	comments          cases.CaseCommentsClient
	links             cases.CaseLinksClient
	services          cases.ServicesClient
	relatedCases      cases.RelatedCasesClient
}

func NewClient(consulTarget string) (*Api, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("consul://%s/webitel.cases?wait=14s", consulTarget),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	casesClient := cases.NewCasesClient(conn)
	caseCommunication := cases.NewCaseCommunicationsClient(conn)
	serviceCatalogs := cases.NewCatalogsClient(conn)
	comments := cases.NewCaseCommentsClient(conn)
	links := cases.NewCaseLinksClient(conn)
	services := cases.NewServicesClient(conn)
	relatedCases := cases.NewRelatedCasesClient(conn)

	return &Api{
		cases:             casesClient,
		caseCommunication: caseCommunication,
		serviceCatalogs:   serviceCatalogs,
		comments:          comments,
		links:             links,
		services:          services,
		relatedCases:      relatedCases,
	}, nil
}

func (api *Api) SearchCases(ctx context.Context, req *cases.SearchCasesRequest, token string) (*cases.CaseList, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)

	// Make the gRPC request
	c, err := api.cases.SearchCases(newCtx, req)
	if err != nil {
		return nil, err
	}

	return c, nil
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

func (api *Api) UpdateCase(ctx context.Context, req *cases.UpdateCaseRequest, token string) (*cases.UpdateCaseResponse, error) {
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

func (api *Api) GetServiceCatalogs(ctx context.Context, req *cases.ListCatalogRequest, token string) (*cases.CatalogList, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)
	// Make the gRPC request
	s, err := api.serviceCatalogs.ListCatalogs(newCtx, req)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (api *Api) PublishComment(ctx context.Context, req *cases.PublishCommentRequest, token string) (*cases.CaseComment, error) {
	// Create a new outgoing context with the updated metadata
	newCtx := attachToken(ctx, token)
	// Make the gRPC request
	c, err := api.comments.PublishComment(newCtx, req)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (api *Api) CreateLink(ctx context.Context, req *cases.CreateLinkRequest, token string) (*cases.CaseLink, error) {
	newCtx := attachToken(ctx, token)
	c, err := api.links.CreateLink(newCtx, req)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (api *Api) DeleteLink(ctx context.Context, req *cases.DeleteLinkRequest, token string) (*cases.CaseLink, error) {
	newCtx := attachToken(ctx, token)
	c, err := api.links.DeleteLink(newCtx, req)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (api *Api) LocateService(ctx context.Context, req *cases.LocateServiceRequest, token string) (*cases.LocateServiceResponse, error) {
	newCtx := attachToken(ctx, token)
	s, err := api.services.LocateService(newCtx, req)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (api *Api) CreateRelatedCase(ctx context.Context, req *cases.CreateRelatedCaseRequest, token string) (*cases.RelatedCase, error) {
	newCtx := attachToken(ctx, token)
	c, err := api.relatedCases.CreateRelatedCase(newCtx, req)
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
