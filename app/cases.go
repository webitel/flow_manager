package app

import (
	"context"

	casespb "github.com/webitel/flow_manager/gen/cases"
)

func (fm *FlowManager) SearchCases(
	ctx context.Context,
	req *casespb.SearchCasesRequest,
	token string,
) (*casespb.CaseList, error) {
	return fm.cases.SearchCases(ctx, req, token)
}

func (fm *FlowManager) LocateCase(
	ctx context.Context,
	req *casespb.LocateCaseRequest,
	token string,
) (*casespb.Case, error) {
	return fm.cases.LocateCase(ctx, req, token)
}

func (fm *FlowManager) CreateCase(
	ctx context.Context,
	req *casespb.CreateCaseRequest,
	token string,
) (*casespb.Case, error) {
	return fm.cases.CreateCase(ctx, req, token)
}

func (fm *FlowManager) UpdateCase(
	ctx context.Context,
	req *casespb.UpdateCaseRequest,
	token string,
) (*casespb.UpdateCaseResponse, error) {
	return fm.cases.UpdateCase(ctx, req, token)
}

func (fm *FlowManager) LinkCommunication(
	ctx context.Context,
	req *casespb.LinkCommunicationRequest,
	token string,
) (*casespb.LinkCommunicationResponse, error) {
	return fm.cases.LinkCommunication(ctx, req, token)
}

func (fm *FlowManager) GetServiceCatalogs(
	ctx context.Context,
	req *casespb.ListCatalogRequest,
	token string,
) (*casespb.CatalogList, error) {
	return fm.cases.GetServiceCatalogs(ctx, req, token)
}

func (fm *FlowManager) PublishComment(
	ctx context.Context,
	req *casespb.PublishCommentRequest,
	token string,
) (*casespb.CaseComment, error) {
	return fm.cases.PublishComment(ctx, req, token)
}

func (fm *FlowManager) CreateLink(
	ctx context.Context,
	req *casespb.CreateLinkRequest,
	token string,
) (*casespb.CaseLink, error) {
	return fm.cases.CreateLink(ctx, req, token)
}

func (fm *FlowManager) DeleteLink(
	ctx context.Context,
	req *casespb.DeleteLinkRequest,
	token string,
) (*casespb.CaseLink, error) {
	return fm.cases.DeleteLink(ctx, req, token)
}

func (fm *FlowManager) LocateService(
	ctx context.Context,
	req *casespb.LocateServiceRequest,
	token string,
) (*casespb.LocateServiceResponse, error) {
	return fm.cases.LocateService(ctx, req, token)
}

func (fm *FlowManager) CreateRelatedCase(
	ctx context.Context,
	req *casespb.CreateRelatedCaseRequest,
	token string,
) (*casespb.RelatedCase, error) {
	return fm.cases.CreateRelatedCase(ctx, req, token)
}

func (fm *FlowManager) ListCaseFiles(
	ctx context.Context,
	req *casespb.ListFilesRequest,
	token string,
) (*casespb.CaseFileList, error) {
	return fm.cases.ListCaseFiles(ctx, req, token)
}

func (fm *FlowManager) LocateCatalog(
	ctx context.Context,
	req *casespb.LocateCatalogRequest,
	token string,
) (*casespb.LocateCatalogResponse, error) {
	return fm.cases.LocateCatalog(ctx, req, token)
}

func (fm *FlowManager) ListStatusConditions(
	ctx context.Context,
	req *casespb.ListStatusConditionRequest,
	token string,
) (*casespb.StatusConditionList, error) {
	return fm.cases.ListStatusConditions(ctx, req, token)
}
