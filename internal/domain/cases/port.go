package cases

import (
	"context"

	pb "github.com/webitel/flow_manager/gen/cases"
)

// Client is the narrow interface that the cases native ops depend on.
// *outboundcases.Api satisfies this interface.
type Client interface {
	SearchCases(ctx context.Context, req *pb.SearchCasesRequest, token string) (*pb.CaseList, error)
	LocateCase(ctx context.Context, req *pb.LocateCaseRequest, token string) (*pb.Case, error)
	CreateCase(ctx context.Context, req *pb.CreateCaseRequest, token string) (*pb.Case, error)
	UpdateCase(ctx context.Context, req *pb.UpdateCaseRequest, token string) (*pb.UpdateCaseResponse, error)
	LinkCommunication(ctx context.Context, req *pb.LinkCommunicationRequest, token string) (*pb.LinkCommunicationResponse, error)
	GetServiceCatalogs(ctx context.Context, req *pb.ListCatalogRequest, token string) (*pb.CatalogList, error)
	PublishComment(ctx context.Context, req *pb.PublishCommentRequest, token string) (*pb.CaseComment, error)
	CreateLink(ctx context.Context, req *pb.CreateLinkRequest, token string) (*pb.CaseLink, error)
	DeleteLink(ctx context.Context, req *pb.DeleteLinkRequest, token string) (*pb.CaseLink, error)
	LocateService(ctx context.Context, req *pb.LocateServiceRequest, token string) (*pb.LocateServiceResponse, error)
	CreateRelatedCase(ctx context.Context, req *pb.CreateRelatedCaseRequest, token string) (*pb.RelatedCase, error)
	ListCaseFiles(ctx context.Context, req *pb.ListFilesRequest, token string) (*pb.CaseFileList, error)
	LocateCatalog(ctx context.Context, req *pb.LocateCatalogRequest, token string) (*pb.LocateCatalogResponse, error)
	ListStatusConditions(ctx context.Context, req *pb.ListStatusConditionRequest, token string) (*pb.StatusConditionList, error)
}
