package cases

import (
	"context"

	"google.golang.org/grpc/metadata"

	cases2 "github.com/webitel/flow_manager/api/gen/cases"
	"github.com/webitel/flow_manager/infra/grpcdial"
)

const ServiceName = "webitel.cases"

type Api struct {
	cases             *grpcdial.Client[cases2.CasesClient]
	caseCommunication *grpcdial.Client[cases2.CaseCommunicationsClient]
	serviceCatalogs   *grpcdial.Client[cases2.CatalogsClient]
	comments          *grpcdial.Client[cases2.CaseCommentsClient]
	links             *grpcdial.Client[cases2.CaseLinksClient]
	services          *grpcdial.Client[cases2.ServicesClient]
	relatedCases      *grpcdial.Client[cases2.RelatedCasesClient]
	caseFiles         *grpcdial.Client[cases2.CaseFilesClient]
	statusConditions  *grpcdial.Client[cases2.StatusConditionsClient]
}

func NewClient(consulTarget string) (*Api, error) {
	var api Api
	var err error

	if api.cases, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewCasesClient); err != nil {
		return nil, err
	}

	if api.caseCommunication, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewCaseCommunicationsClient); err != nil {
		return nil, err
	}

	if api.serviceCatalogs, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewCatalogsClient); err != nil {
		return nil, err
	}

	if api.comments, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewCaseCommentsClient); err != nil {
		return nil, err
	}

	if api.links, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewCaseLinksClient); err != nil {
		return nil, err
	}

	if api.services, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewServicesClient); err != nil {
		return nil, err
	}

	if api.relatedCases, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewRelatedCasesClient); err != nil {
		return nil, err
	}

	if api.caseFiles, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewCaseFilesClient); err != nil {
		return nil, err
	}

	if api.statusConditions, err = grpcdial.NewClient(consulTarget, ServiceName, cases2.NewStatusConditionsClient); err != nil {
		return nil, err
	}

	return &api, nil
}

func (api *Api) SearchCases(ctx context.Context, req *cases2.SearchCasesRequest, token string) (*cases2.CaseList, error) {
	return api.cases.API.SearchCases(attachToken(ctx, token), req)
}

func (api *Api) LocateCase(ctx context.Context, req *cases2.LocateCaseRequest, token string) (*cases2.Case, error) {
	return api.cases.API.LocateCase(attachToken(ctx, token), req)
}

func (api *Api) CreateCase(ctx context.Context, req *cases2.CreateCaseRequest, token string) (*cases2.Case, error) {
	return api.cases.API.CreateCase(attachToken(ctx, token), req)
}

func (api *Api) UpdateCase(ctx context.Context, req *cases2.UpdateCaseRequest, token string) (*cases2.UpdateCaseResponse, error) {
	return api.cases.API.UpdateCase(attachToken(ctx, token), req)
}

func (api *Api) LinkCommunication(ctx context.Context, req *cases2.LinkCommunicationRequest, token string) (*cases2.LinkCommunicationResponse, error) {
	return api.caseCommunication.API.LinkCommunication(attachToken(ctx, token), req)
}

func (api *Api) GetServiceCatalogs(ctx context.Context, req *cases2.ListCatalogRequest, token string) (*cases2.CatalogList, error) {
	return api.serviceCatalogs.API.ListCatalogs(attachToken(ctx, token), req)
}

func (api *Api) PublishComment(ctx context.Context, req *cases2.PublishCommentRequest, token string) (*cases2.CaseComment, error) {
	return api.comments.API.PublishComment(attachToken(ctx, token), req)
}

func (api *Api) CreateLink(ctx context.Context, req *cases2.CreateLinkRequest, token string) (*cases2.CaseLink, error) {
	return api.links.API.CreateLink(attachToken(ctx, token), req)
}

func (api *Api) DeleteLink(ctx context.Context, req *cases2.DeleteLinkRequest, token string) (*cases2.CaseLink, error) {
	return api.links.API.DeleteLink(attachToken(ctx, token), req)
}

func (api *Api) LocateService(ctx context.Context, req *cases2.LocateServiceRequest, token string) (*cases2.LocateServiceResponse, error) {
	return api.services.API.LocateService(attachToken(ctx, token), req)
}

func (api *Api) CreateRelatedCase(ctx context.Context, req *cases2.CreateRelatedCaseRequest, token string) (*cases2.RelatedCase, error) {
	return api.relatedCases.API.CreateRelatedCase(attachToken(ctx, token), req)
}

func (api *Api) ListCaseFiles(ctx context.Context, req *cases2.ListFilesRequest, token string) (*cases2.CaseFileList, error) {
	return api.caseFiles.API.ListFiles(attachToken(ctx, token), req)
}

func (api *Api) LocateCatalog(ctx context.Context, req *cases2.LocateCatalogRequest, token string) (*cases2.LocateCatalogResponse, error) {
	return api.serviceCatalogs.API.LocateCatalog(attachToken(ctx, token), req)
}

func (api *Api) ListStatusConditions(ctx context.Context, req *cases2.ListStatusConditionRequest, token string) (*cases2.StatusConditionList, error) {
	return api.statusConditions.API.ListStatusConditions(attachToken(ctx, token), req)
}

func attachToken(ctx context.Context, token string) context.Context {
	existingMD, _ := metadata.FromIncomingContext(ctx)
	md := metadata.Join(existingMD, metadata.Pairs("X-Webitel-Access", token))
	return metadata.NewOutgoingContext(ctx, md)
}
