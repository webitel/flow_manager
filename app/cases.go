package app

import (
	"context"

	grpc_cases "buf.build/gen/go/webitel/cases/protocolbuffers/go"
	"github.com/webitel/flow_manager/cases"
)

func (fm *FlowManager) SearchCases(ctx context.Context, req *cases.SearchCasesRequest) (*grpc_cases.CaseList, error) {
	return fm.cases.SearchCases(ctx, req)
}
