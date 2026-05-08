package ports

import (
	"context"
	"io"
	"time"

	"github.com/webitel/wlog"

	casespb "github.com/webitel/flow_manager/gen/cases"
	genpb "github.com/webitel/flow_manager/gen/cc"
	aibridge "github.com/webitel/flow_manager/internal/adapters/outbound/aibridge"
	domcases "github.com/webitel/flow_manager/internal/domain/cases"
	domcc "github.com/webitel/flow_manager/internal/domain/cc"
	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
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
	Meeting() domainmeeting.Client
	Cases() domcases.Client

	// schema / variable
	GetLocation(id int) *time.Location
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
	ContactLinkToChat(ctx context.Context, conversationId string, contactId string) *model.AppError
	JoinChatToInboundQueue(ctx context.Context, in *genpb.ChatJoinToQueueRequest) (genpb.MemberService_ChatJoinToQueueClient, error)
	JoinIMToInboundQueue(ctx context.Context, in *genpb.IMJoinToQueueRequest) (int64, <-chan domcc.QueueEvent, error)
	LeavingIMToInboundQueue(attId int64)
	SenChatAction(ctx context.Context, channelId string, action model.ChatAction) *model.AppError
	SearchMediaFile(domainId int64, search *model.SearchFile) (*model.File, *model.AppError)
	SetupPublicFileUrl(file *model.File, domainId int64, server, source string, expire int64) (*model.File, *model.AppError)
	GetFileTranscription(ctx context.Context, fileId, domainId int64, profileId int64, language string) (string, *model.AppError)
	ChatProfileType(domainId int64, profileId int) (string, error)
	BroadcastChatMessage(ctx context.Context, domainId int64, req model.BroadcastChat, peers []model.BroadcastPeer) (*model.BroadcastChatResponse, error)
	GetChatMessagesByConversationId(ctx context.Context, domainId int64, conversationId string, limit int64) (*[]model.ChatMessage, error)
	ParseChatMessages(messages *[]model.ChatMessage, format string) (string, error)

	// http / cookie cache
	GetCookieCache(ctx context.Context, domainID int64, key string) (string, error)
	SetCookieCache(ctx context.Context, domainID int64, key string, value string, ttlSecs int64) error

	// global schema variables
	SetGlobalVar(ctx context.Context, domainId int64, name string, value string, encrypt bool) error

	// external sql
	SqlQuery(ctx context.Context, driver, dns, query string, params []interface{}) (map[string]interface{}, error)

	// file links
	GeneratePreSignedLink(ctx context.Context, action, source string, fileId, domainId int64, query map[string]string) (string, error)

	// open link (send URL to agent browser via WebSocket)
	PushOpenLink(domainId int64, sockId string, userId int64, message, url string) error

	// list
	CheckList(domainId int64, number string, listId *int, listName *string) (bool, error)
	AddToList(ctx context.Context, domainId int64, listId *int, listName *string, destination string, description *string, expireAtMS int64) error

	// cache
	CacheGet(ctx context.Context, cacheType string, domainID int64, key string) (string, error)
	CacheSet(ctx context.Context, cacheType string, domainID int64, key string, value string, ttlSecs int64) error
	CacheDelete(ctx context.Context, cacheType string, domainID int64, key string) error

	// email
	MailSetContacts(ctx context.Context, domainId int64, id string, contactIds []int64) *model.AppError
	ReplyEmail(conn model.EmailConnection, text string) *model.AppError
	SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, error)
	SmtpSettingsOAuthToken(settings *model.SmtSettings) (string, error)
	GetFileMetadata(domainId int64, ids []int64) ([]model.File, error)
	DownloadFile(domainId int64, id int64) (io.ReadCloser, error)
	SaveEmail(domainId int64, email *model.Email) error

	// processing / cases
	LocateService(ctx context.Context, req *casespb.LocateServiceRequest, token string) (*casespb.LocateServiceResponse, error)
	LocateCatalog(ctx context.Context, req *casespb.LocateCatalogRequest, token string) (*casespb.LocateCatalogResponse, error)
	ListStatusConditions(ctx context.Context, req *casespb.ListStatusConditionRequest, token string) (*casespb.StatusConditionList, error)
}
