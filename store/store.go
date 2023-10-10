package store

import (
	"context"
	"database/sql"

	"github.com/webitel/flow_manager/model"
)

var ErrNoRows = sql.ErrNoRows

type Store interface {
	Call() CallStore
	Schema() SchemaStore
	CallRouting() CallRoutingStore
	Endpoint() EndpointStore
	Email() EmailStore
	Media() MediaStore
	Calendar() CalendarStore
	List() ListStore
	Chat() ChatStore
	Queue() QueueStore
	Member() MemberStore
	User() UserStore
	Log() LogStore
	File() FileStore
}

type CacheStore interface {
	Set(key interface{}, value interface{}) *model.AppError
}

type UserStore interface {
	GetProperties(domainId int64, search *model.SearchUser, mapRes model.Variables) (model.Variables, *model.AppError)
	GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError)
}

type EmailStore interface {
	Save(domainId int64, m *model.Email) *model.AppError
	ProfileTaskFetch(node string) ([]*model.EmailProfileTask, *model.AppError)
	GetProfile(id int) (*model.EmailProfile, *model.AppError)
	GetProfileUpdatedAt(domainId int64, id int) (int64, *model.AppError)
	SetError(profileId int, appErr *model.AppError) *model.AppError

	GerProperties(domainId int64, id *int64, messageId *string, mapRes model.Variables) (model.Variables, *model.AppError)
	SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, *model.AppError)
}

type CallStore interface {
	Save(call *model.CallActionRinging) *model.AppError
	SetState(call *model.CallAction) *model.AppError
	SetBridged(call *model.CallActionBridge) *model.AppError
	SetHangup(call *model.CallActionHangup) *model.AppError
	MoveToHistory() *model.AppError
	UpdateFrom(id string, name, number *string) *model.AppError
	SaveTranscribe(callId, transcribe string) *model.AppError

	LastBridged(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError)
	SetGranteeId(domainId int64, id string, granteeId int64) *model.AppError
	SetUserId(domainId int64, id string, userId int64) *model.AppError
	SetBlindTransfer(domainId int64, id string, destination string) *model.AppError
	SetContactId(domainId int64, id string, contactId int64) *model.AppError
}

type SchemaStore interface {
	Get(domainId int64, id int) (*model.Schema, *model.AppError)
	GetUpdatedAt(domainId int64, id int) (int64, *model.AppError)
	GetTransferredRouting(domainId int64, schemaId int) (*model.Routing, *model.AppError)
}

type CallRoutingStore interface {
	FromGateway(domainId int64, gatewayId int) (*model.Routing, *model.AppError)
	SearchToDestination(domainId int64, destination string) (*model.Routing, *model.AppError)
	FromQueue(domainId int64, queueId int) (*model.Routing, *model.AppError)
}

type EndpointStore interface {
	Get(domainId int64, callerName, callerNumber string, endpoints model.Applications) ([]*model.Endpoint, *model.AppError)
}

type MediaStore interface {
	GetFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError)
	Get(domainId int64, id int) (*model.File, *model.AppError)
	SearchOne(domainId int64, search *model.SearchFile) (*model.File, *model.AppError)
}

type CalendarStore interface {
	Check(domainId int64, id *int, name *string) (*model.Calendar, *model.AppError)
	GetTimezones() ([]*model.Timezone, *model.AppError)
}

type ListStore interface {
	CheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError)
	AddDestination(domainId int64, entry *model.SearchEntity, comm *model.ListCommunication) *model.AppError
	CleanExpired() (int64, *model.AppError)
}

type ChatStore interface {
	RoutingFromProfile(domainId, profileId int64) (*model.Routing, *model.AppError)
	RoutingFromSchemaId(domainId int64, schemaId int32) (*model.Routing, *model.AppError)
	GetMessagesByConversation(ctx context.Context, domainId int64, conversationId string, limit int64) (*[]model.ChatMessage, *model.AppError)
	LastBridged(domainId int64, number, hours string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError)
}

type QueueStore interface {
	HistoryStatistics(domainId int64, search *model.SearchQueueCompleteStatistics) (float64, *model.AppError)
	GetQueueData(domainId int64, search *model.SearchEntity, mapRes model.Variables) (model.Variables, *model.AppError)
	GetQueueAgents(domainId int64, queueId int, channel string, mapRes model.Variables) (model.Variables, *model.AppError)
}

type MemberStore interface {
	CreateMember(domainId int64, queueId int, holdSec int, member *model.CallbackMember) *model.AppError
	CallPosition(callId string) (int64, *model.AppError)
	EWTPuzzle(callId string, min int, queueIds []int, bucketIds []int) (float64, *model.AppError)
	GetProperties(domainId int64, req *model.SearchMember, mapRes model.Variables) (model.Variables, *model.AppError)
	PatchMembers(domainId int64, req *model.SearchMember, patch *model.PatchMember) (int, *model.AppError)
}

type LogStore interface {
	Save(schemaId int, connId string, log interface{}) *model.AppError
}

type FileStore interface {
	GetMetadata(domainId int64, ids []int64) ([]model.File, *model.AppError)
}
