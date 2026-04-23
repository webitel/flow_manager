package cases

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/webitel/flow_manager/gen/cases"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const ServiceName = "webitel.cases"

type Api struct {
	cases             *grpcdial.Client[cases.CasesClient]
	caseCommunication *grpcdial.Client[cases.CaseCommunicationsClient]
	serviceCatalogs   *grpcdial.Client[cases.CatalogsClient]
	comments          *grpcdial.Client[cases.CaseCommentsClient]
	links             *grpcdial.Client[cases.CaseLinksClient]
	services          *grpcdial.Client[cases.ServicesClient]
	relatedCases      *grpcdial.Client[cases.RelatedCasesClient]
	caseFiles         *grpcdial.Client[cases.CaseFilesClient]
	statusConditions  *grpcdial.Client[cases.StatusConditionsClient]
}

func NewClient(consulTarget string) (*Api, error) {
	var api Api
	var err error

	if api.cases, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewCasesClient); err != nil {
		return nil, err
	}

	if api.caseCommunication, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewCaseCommunicationsClient); err != nil {
		return nil, err
	}

	if api.serviceCatalogs, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewCatalogsClient); err != nil {
		return nil, err
	}

	if api.comments, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewCaseCommentsClient); err != nil {
		return nil, err
	}

	if api.links, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewCaseLinksClient); err != nil {
		return nil, err
	}

	if api.services, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewServicesClient); err != nil {
		return nil, err
	}

	if api.relatedCases, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewRelatedCasesClient); err != nil {
		return nil, err
	}

	if api.caseFiles, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewCaseFilesClient); err != nil {
		return nil, err
	}

	if api.statusConditions, err = grpcdial.NewClient(consulTarget, ServiceName, cases.NewStatusConditionsClient); err != nil {
		return nil, err
	}

	return &api, nil
}

func (api *Api) SearchCases(ctx context.Context, req *cases.SearchCasesRequest, token string) (*cases.CaseList, error) {
	return api.cases.API.SearchCases(attachToken(ctx, token), req)
}

func (api *Api) LocateCase(ctx context.Context, req *cases.LocateCaseRequest, token string) (*cases.Case, error) {
	return api.cases.API.LocateCase(attachToken(ctx, token), req)
}

func (api *Api) CreateCase(ctx context.Context, req *cases.CreateCaseRequest, token string) (*cases.Case, error) {
	return api.cases.API.CreateCase(attachToken(ctx, token), req)
}

func (api *Api) UpdateCase(ctx context.Context, req *cases.UpdateCaseRequest, token string) (*cases.UpdateCaseResponse, error) {
	return api.cases.API.UpdateCase(attachToken(ctx, token), req)
}

func (api *Api) LinkCommunication(ctx context.Context, req *cases.LinkCommunicationRequest, token string) (*cases.LinkCommunicationResponse, error) {
	return api.caseCommunication.API.LinkCommunication(attachToken(ctx, token), req)
}

func (api *Api) GetServiceCatalogs(ctx context.Context, req *cases.ListCatalogRequest, token string) (*cases.CatalogList, error) {
	return api.serviceCatalogs.API.ListCatalogs(attachToken(ctx, token), req)
}

func (api *Api) PublishComment(ctx context.Context, req *cases.PublishCommentRequest, token string) (*cases.CaseComment, error) {
	return api.comments.API.PublishComment(attachToken(ctx, token), req)
}

func (api *Api) CreateLink(ctx context.Context, req *cases.CreateLinkRequest, token string) (*cases.CaseLink, error) {
	return api.links.API.CreateLink(attachToken(ctx, token), req)
}

func (api *Api) DeleteLink(ctx context.Context, req *cases.DeleteLinkRequest, token string) (*cases.CaseLink, error) {
	return api.links.API.DeleteLink(attachToken(ctx, token), req)
}

func (api *Api) LocateService(ctx context.Context, req *cases.LocateServiceRequest, token string) (*cases.LocateServiceResponse, error) {
	return api.services.API.LocateService(attachToken(ctx, token), req)
}

func (api *Api) CreateRelatedCase(ctx context.Context, req *cases.CreateRelatedCaseRequest, token string) (*cases.RelatedCase, error) {
	return api.relatedCases.API.CreateRelatedCase(attachToken(ctx, token), req)
}

func (api *Api) ListCaseFiles(ctx context.Context, req *cases.ListFilesRequest, token string) (*cases.CaseFileList, error) {
	return api.caseFiles.API.ListFiles(attachToken(ctx, token), req)
}

func (api *Api) LocateCatalog(ctx context.Context, req *cases.LocateCatalogRequest, token string) (*cases.LocateCatalogResponse, error) {
	return api.serviceCatalogs.API.LocateCatalog(attachToken(ctx, token), req)
}

func (api *Api) ListStatusConditions(ctx context.Context, req *cases.ListStatusConditionRequest, token string) (*cases.StatusConditionList, error) {
	return api.statusConditions.API.ListStatusConditions(attachToken(ctx, token), req)
}

func attachToken(ctx context.Context, token string) context.Context {
	existingMD, _ := metadata.FromIncomingContext(ctx)
	md := metadata.Join(existingMD, metadata.Pairs("X-Webitel-Access", token))
	return metadata.NewOutgoingContext(ctx, md)
}
