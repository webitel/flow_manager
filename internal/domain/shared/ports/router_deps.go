package ports

import (
	"context"

	"github.com/webitel/wlog"

	casespb "github.com/webitel/flow_manager/gen/cases"
	genpb "github.com/webitel/flow_manager/gen/cc"
	aibridge "github.com/webitel/flow_manager/internal/adapters/outbound/aibridge"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	"github.com/webitel/flow_manager/internal/runtime/persistence"
	"github.com/webitel/flow_manager/internal/session"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

// RouterDeps is the narrow interface that all route Init functions depend on.
// *app.FlowManager satisfies this interface.
type RouterDeps interface {
	AppID() string
	Log() *wlog.Logger
	Config() *model.Config
	CheckpointRepo() session.Repository
	RuntimeStateRepo() persistence.Repository
	GetStore() store.Store
	GetAiBots() *aibridge.Client

	// schema / variable
	SchemaVariable(ctx context.Context, domainId int64, name string) string
	GetSchemaById(domainId int64, id int) (*model.Schema, *model.AppError)
	GetSystemSettings(ctx context.Context, domainId int64, name string) (model.SysValue, *model.AppError)

	// logging / variables
	StoreLog(schemaId int, connId string, log []*model.StepLog) *model.AppError
	StoreCallVariables(id string, vars map[string]string) *model.AppError

	// call routing
	SearchTransferredRouting(domainId int64, schemaId int) (*model.Routing, *model.AppError)
	SearchOutboundToDestinationRouting(domainId int64, dest string) (*model.Routing, *model.AppError)
	SearchOutboundFromQueueRouting(domainId int64, queueId int) (*model.Routing, *model.AppError)
	TransferQueueRouting(domainId int64, queueId int) (*model.Routing, *model.AppError)
	TransferAgentRouting(domainId int64, agentId int) (*model.Routing, *model.AppError)
	GetRoutingFromDestToGateway(domainId int64, gatewayId int) (*model.Routing, *model.AppError)
	SetBlindTransferNumber(domainId int64, callId, destination string) *model.AppError

	// call operations
	UpdateCallFrom(id string, name, number, destination *string) *model.AppError
	SetCallUserId(domainId int64, id string, userId int64) *model.AppError
	SetCallGranteeId(domainId int64, id string, granteeId int64) *model.AppError
	CallSetContactId(domainId int64, callId string, contactId int64) *model.AppError
	UserNotification(n model.Notification)
	GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError)
	GetMediaFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError)
	GetPlaybackFile(domainId int64, search *model.PlaybackFile) (*model.PlaybackFile, *model.AppError)
	GenerateTTSLink(ctx context.Context, text string, domainId int64, profileId int, textType string, voice string, language string) (string, *model.AppError)

	// CC / queue
	JoinToInboundQueue(ctx context.Context, in *genpb.CallJoinToQueueRequest) (genpb.MemberService_CallJoinToQueueClient, error)
	JoinToAgent(ctx context.Context, in *genpb.CallJoinToAgentRequest) (genpb.MemberService_CallJoinToAgentClient, error)
	CallOutboundQueue(ctx context.Context, in *genpb.OutboundCallRequest) (*genpb.OutboundCallResponse, error)
	CancelAttempt(ctx context.Context, att model.InQueueKey, result string) *model.AppError
	AttemptResult(result *model.AttemptResult) *model.AppError
	ResumeAttempt(ctx context.Context, attemptId, domainId int64) error
	FindQueueByName(domainId int64, name string) (int32, *model.AppError)

	// chat / IM routing
	GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*model.Routing, *model.AppError)
	GetChatRouteFromUserId(domainId int64, userId int64) (*model.Routing, *model.AppError)
	GetChatRouteFromProfile(domainId, profileId int64) (*model.Routing, *model.AppError)

	// chat operations
	JoinChatToInboundQueue(ctx context.Context, in *genpb.ChatJoinToQueueRequest) (genpb.MemberService_ChatJoinToQueueClient, error)
	JoinIMToInboundQueue(ctx context.Context, in *genpb.IMJoinToQueueRequest) (int64, <-chan domcc.QueueEvent, error)
	LeavingIMToInboundQueue(attId int64)
	SenChatAction(ctx context.Context, channelId string, action model.ChatAction) *model.AppError
	SearchMediaFile(domainId int64, search *model.SearchFile) (*model.File, *model.AppError)
	SetupPublicFileUrl(file *model.File, domainId int64, server, source string, expire int64) (*model.File, *model.AppError)
	GetFileTranscription(ctx context.Context, fileId, domainId int64, profileId int64, language string) (string, *model.AppError)

	// email
	MailSetContacts(ctx context.Context, domainId int64, id string, contactIds []int64) *model.AppError
	ReplyEmail(conn model.EmailConnection, text string) *model.AppError

	// processing / cases
	LocateService(ctx context.Context, req *casespb.LocateServiceRequest, token string) (*casespb.LocateServiceResponse, error)
	LocateCatalog(ctx context.Context, req *casespb.LocateCatalogRequest, token string) (*casespb.LocateCatalogResponse, error)
	ListStatusConditions(ctx context.Context, req *casespb.ListStatusConditionRequest, token string) (*casespb.StatusConditionList, error)
}
