package cases

import (
	"context"

	cases2 "github.com/webitel/flow_manager/api/gen/cases"
)

// Client is the narrow interface that the cases native ops depend on.
// *outboundcases.Api satisfies this interface.
type Client interface {
	SearchCases(ctx context.Context, req *cases2.SearchCasesRequest, token string) (*cases2.CaseList, error)
	LocateCase(ctx context.Context, req *cases2.LocateCaseRequest, token string) (*cases2.Case, error)
	CreateCase(ctx context.Context, req *cases2.CreateCaseRequest, token string) (*cases2.Case, error)
	UpdateCase(ctx context.Context, req *cases2.UpdateCaseRequest, token string) (*cases2.UpdateCaseResponse, error)
	LinkCommunication(ctx context.Context, req *cases2.LinkCommunicationRequest, token string) (*cases2.LinkCommunicationResponse, error)
	GetServiceCatalogs(ctx context.Context, req *cases2.ListCatalogRequest, token string) (*cases2.CatalogList, error)
	PublishComment(ctx context.Context, req *cases2.PublishCommentRequest, token string) (*cases2.CaseComment, error)
	CreateLink(ctx context.Context, req *cases2.CreateLinkRequest, token string) (*cases2.CaseLink, error)
	DeleteLink(ctx context.Context, req *cases2.DeleteLinkRequest, token string) (*cases2.CaseLink, error)
	LocateService(ctx context.Context, req *cases2.LocateServiceRequest, token string) (*cases2.LocateServiceResponse, error)
	CreateRelatedCase(ctx context.Context, req *cases2.CreateRelatedCaseRequest, token string) (*cases2.RelatedCase, error)
	ListCaseFiles(ctx context.Context, req *cases2.ListFilesRequest, token string) (*cases2.CaseFileList, error)
	LocateCatalog(ctx context.Context, req *cases2.LocateCatalogRequest, token string) (*cases2.LocateCatalogResponse, error)
	ListStatusConditions(ctx context.Context, req *cases2.ListStatusConditionRequest, token string) (*cases2.StatusConditionList, error)
}
