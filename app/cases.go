package app

import (
	"context"

	cases_pb "buf.build/gen/go/webitel/cases/protocolbuffers/go"
)

func (fm *FlowManager) SearchCases(
	ctx context.Context,
	req *cases_pb.SearchCasesRequest,
	token string,
) (*cases_pb.CaseList, error) {
	return fm.cases.SearchCases(ctx, req, token)
}

func (fm *FlowManager) LocateCase(
	ctx context.Context,
	req *cases_pb.LocateCaseRequest,
	token string,
) (*cases_pb.Case, error) {
	return fm.cases.LocateCase(ctx, req, token)
}

func (fm *FlowManager) CreateCase(
	ctx context.Context,
	req *cases_pb.CreateCaseRequest,
	token string,
) (*cases_pb.Case, error) {
	return fm.cases.CreateCase(ctx, req, token)
}

func (fm *FlowManager) UpdateCase(
	ctx context.Context,
	req *cases_pb.UpdateCaseRequest,
	token string,
) (*cases_pb.Case, error) {
	return fm.cases.UpdateCase(ctx, req, token)
}

func (fm *FlowManager) LinkCommunication(
	ctx context.Context,
	req *cases_pb.LinkCommunicationRequest,
	token string,
) (*cases_pb.LinkCommunicationResponse, error) {
	return fm.cases.LinkCommunication(ctx, req, token)
}
