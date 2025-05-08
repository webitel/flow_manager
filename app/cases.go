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
) (*casespb.Case, error) {
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
