package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"golang.org/x/oauth2"

	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/calendar"
	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/email"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/list"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/domain/routing"
	"github.com/webitel/flow_manager/internal/domain/session"
	"github.com/webitel/flow_manager/internal/domain/user"
	"github.com/webitel/flow_manager/internal/domain/webhook"
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
	WebHook() WebHookStore
	SystemcSettings() SystemcSettings
	SocketSession() SocketSessionStore
	Session() SessionStore
}

type SocketSessionStore interface {
	Get(userID, domainID int64, appName string) (*session.SocketSession, error)
}

type SessionStore interface {
	Touch(id, appId string) (*int, error)
	Remove(id, appId string) error
	RemoveAll(appId string) error
}

type CacheStore interface {
	Set(key, value any) error
}

type UserStore interface {
	GetProperties(domainId int64, search *user.SearchUser, mapRes flow.Variables) (flow.Variables, error)
	GetAgentIdByExtension(domainId int64, extension string) (*int32, error)
}

type EmailStore interface {
	Save(domainId int64, m *email.Email) error
	ProfileTaskFetch(node string) ([]*email.EmailProfileTask, error)
	GetProfile(id int) (*email.EmailProfile, error)
	GetProfileUpdatedAt(domainId int64, id int) (int64, error)
	SetError(profileId int, appErr error) error

	GerProperties(domainId int64, id *int64, messageId *string, mapRes flow.Variables) (flow.Variables, error)
	SetToken(id int, token *oauth2.Token) error
	SmtpSettings(domainId int64, search *queue.SearchEntity) (*email.SmtSettings, error)
	SetContact(ctx context.Context, domainId int64, id string, contactIds []int64) error
}

type CallStore interface {
	Save(c *call.CallActionRinging) error
	SetState(c *call.CallAction) error
	SetBridged(c *call.CallActionBridge) error
	SetHangup(c *call.CallActionHangup) error
	MoveToHistory() ([]call.MissedCall, error)
	Delete(id string) error
	UpdateFrom(id string, name, number, destination *string) error
	SaveTranscript(transcribe *call.CallActionTranscript) error
	SetHeartbeat(id string) error

	LastBridged(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes flow.Variables) (flow.Variables, error)
	SetGranteeId(domainId int64, id string, granteeId int64) error
	SetUserId(domainId int64, id string, userId int64) error
	SetBlindTransfer(domainId int64, id, destination string) error
	SetContactId(domainId int64, id string, contactId int64) error
	SetVariables(id string, vars *call.CallVariables) error
	SaveMediaStats(stats *call.CallActionMediaStats) error
}

type SchemaStore interface {
	Get(domainId int64, id int) (*routing.Schema, error)
	GetUpdatedAt(domainId int64, id int) (int64, error)
	GetTransferredRouting(domainId int64, schemaId int) (*routing.Routing, error)
	GetVariable(domainId int64, name string) (*routing.SchemaVariable, error)
	SetVariable(domainId int64, name string, val *routing.SchemaVariable) error
}

type CallRoutingStore interface {
	FromGateway(domainId int64, gatewayId int) (*routing.Routing, error)
	SearchToDestination(domainId int64, destination string) (*routing.Routing, error)
	FromQueue(domainId int64, queueId int) (*routing.Routing, error)
}

type EndpointStore interface {
	Get(domainId int64, callerName, callerNumber string, endpoints flow.Applications) ([]*call.Endpoint, error)
}

type MediaStore interface {
	GetFiles(domainId int64, req *[]*call.PlaybackFile) ([]*call.PlaybackFile, error)
	Get(domainId int64, id int) (*files.File, error)
	SearchOne(domainId int64, search *files.SearchFile) (*files.File, error)
	GetPlaybackFile(domainId int64, req *call.PlaybackFile) (*call.PlaybackFile, error)
}

type CalendarStore interface {
	Check(domainId int64, id *int, name *string) (*calendar.Calendar, error)
	GetTimezones() ([]*calendar.Timezone, error)
}

type ListStore interface {
	CheckNumber(domainId int64, number string, listId *int, listName *string) (bool, error)
	AddDestination(domainId int64, entry *queue.SearchEntity, comm *list.ListCommunication) error
	CleanExpired() (int64, error)
}

type ChatStore interface {
	RoutingFromProfile(domainId, profileId int64) (*routing.Routing, error)
	RoutingFromSchemaId(domainId int64, schemaId int32) (*routing.Routing, error)
	GetMessagesByConversation(ctx context.Context, domainId int64, conversationId string, limit int64) ([]chatdomain.ChatMessage, error)
	LastBridged(domainId int64, number, hours string, queueIds []int, mapRes flow.Variables) (flow.Variables, error)
	ProfileType(domainId int64, profileId int) (string, error)
}

type QueueStore interface {
	HistoryStatistics(domainId int64, search *queue.SearchQueueCompleteStatistics) (float64, error)
	GetQueueData(domainId int64, search *queue.SearchEntity, mapRes flow.Variables) (flow.Variables, error)
	GetQueueAgents(domainId int64, queueId int, channel string, mapRes flow.Variables) (flow.Variables, error)
	FindQueueByName(domainId int64, name string) (int32, error)
}

type MemberStore interface {
	CreateMember(domainId int64, queueId, holdSec int, member *queue.CallbackMember) error
	CallPosition(callId string) (int64, error)
	EWTPuzzle(domainId int64, callId string, min int, queueIds, bucketIds []int) (float64, error)
	GetProperties(domainId int64, req *queue.SearchMember, mapRes flow.Variables) (flow.Variables, error)
	PatchMembers(domainId int64, req *queue.SearchMember, patch *queue.PatchMember) (int, error)
}

type LogStore interface {
	Save(schemaId int, connId string, log any) error
}

type FileStore interface {
	GetMetadata(domainId int64, ids []int64) ([]files.File, error)
}

type WebHookStore interface {
	Get(id string) (webhook.WebHook, error)
}

type SystemcSettings interface {
	Get(ctx context.Context, domainId int64, name string) (json.RawMessage, error)
}
