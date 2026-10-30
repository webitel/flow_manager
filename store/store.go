package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"golang.org/x/oauth2"

	"github.com/webitel/flow_manager/model"
)

var ErrNoRows = sql.ErrNoRows

type Store interface {
	Call() CallStore
	Schema() SchemaStore
	CallRouting() CallRoutingStore
	Endpoint() EndpointStore
	Email() EmailStore
	EmailEvent() EmailEventStore
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
	Get(userID, domainID int64, appName string) (*model.SocketSession, *model.AppError)
}

type SessionStore interface {
	Touch(id, appId string) (*int, error)
	Remove(id, appId string) error
	RemoveByThread(threadID string) error
	RemoveAll(appId string) error
}

type CacheStore interface {
	Set(key, value any) *model.AppError
}

type UserStore interface {
	GetProperties(domainId int64, search *model.SearchUser, mapRes model.Variables) (model.Variables, *model.AppError)
	GetAgentIdByExtension(domainId int64, extension string) (*int32, *model.AppError)
}

type EmailStore interface {
	LastBridged(ctx context.Context, domainId int64, number, hours string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError)
	Save(domainId int64, m *model.Email) *model.AppError
	ProfileTaskFetch(node string) ([]*model.EmailProfileTask, *model.AppError)
	GetProfile(id int) (*model.EmailProfile, *model.AppError)
	GetProfileUpdatedAt(domainId int64, id int) (int64, *model.AppError)
	SetError(profileId int, appErr *model.AppError) *model.AppError

	GerProperties(domainId int64, id *int64, messageId *string, mapRes model.Variables) (model.Variables, *model.AppError)
	SetToken(id int, token *oauth2.Token) *model.AppError
	SmtpSettings(domainId int64, search *model.SearchEntity) (*model.SmtSettings, *model.AppError)
	SetContact(ctx context.Context, domainId int64, id string, contactIds []int64) *model.AppError
}

// EmailEventStore is the Inbox of email-service events. Every write to a claimed
// row is conditional on the lease of owner, so a worker never overwrites another.
type EmailEventStore interface {
	// Insert stores a pending event; false means the event_id is already stored.
	Insert(ctx context.Context, e *model.EmailEvent) (bool, *model.AppError)
	// RecoverExpired returns rows with an expired lease to pending.
	RecoverExpired(ctx context.Context) (int64, *model.AppError)
	// Claim leases the next due row to owner; nil when none is due.
	Claim(ctx context.Context, owner string, lease time.Duration) (*model.EmailEvent, *model.AppError)
	MarkAccepted(ctx context.Context, owner, eventId string) (model.EmailEventAcceptResult, *model.AppError)
	// Release records a failed launch and returns the row to pending at next.
	Release(ctx context.Context, owner, eventId, reason string, next time.Time) (bool, *model.AppError)
	// MarkFailed records a failed launch and gives up on the row.
	MarkFailed(ctx context.Context, owner, eventId, reason string) (bool, *model.AppError)
	// DeleteAcceptedBefore removes up to limit rows accepted before the time; unfinished
	// routes are kept.
	DeleteAcceptedBefore(ctx context.Context, before time.Time, limit int) (int64, *model.AppError)

	// InitRoute starts the route of a row owner holds, once: pending with the deadline when needed,
	// not_needed otherwise. EmailRouteUnset means owner lost the row before the route was written.
	InitRoute(ctx context.Context, owner, eventId string, needed bool, fallbackQueueId *int32, fromAddress string, due time.Duration) (*model.EmailRoute, *model.AppError)
	// BeginFlowRoute records the Flow intent and leases the row: pending before the deadline, or joining
	// the same queue, which keeps the first intent; false returns the route that stopped it, nil if gone.
	BeginFlowRoute(ctx context.Context, owner, eventId string, intent model.EmailRouteIntent, lease time.Duration) (bool, *model.EmailRoute, *model.AppError)
	// Route reads the route of an event; nil when the row is gone.
	Route(ctx context.Context, eventId string) (*model.EmailRoute, *model.AppError)
	// ClaimRoutes leases up to limit routes due for the watcher; a pending one past the
	// deadline becomes resolving.
	ClaimRoutes(ctx context.Context, owner string, limit int, lease time.Duration) ([]*model.EmailRoute, *model.AppError)
	// FinishRoute moves a route owner holds in state from to the result and frees it.
	FinishRoute(ctx context.Context, owner, eventId string, from model.EmailRouteState, res model.EmailRouteResult) (bool, *model.AppError)
	// RetryRoute frees a route owner holds in state from for another join at next;
	// counted spends an attempt.
	RetryRoute(ctx context.Context, owner, eventId string, from model.EmailRouteState, reason string, next time.Time, counted bool) (bool, *model.AppError)
}

type CallStore interface {
	Save(call *model.CallActionRinging) *model.AppError
	SetState(call *model.CallAction) *model.AppError
	SetBridged(call *model.CallActionBridge) *model.AppError
	SetHangup(call *model.CallActionHangup) *model.AppError
	MoveToHistory() ([]model.MissedCall, *model.AppError)
	Delete(id string) *model.AppError
	UpdateFrom(id string, name, number, destination *string) *model.AppError
	SaveTranscript(transcribe *model.CallActionTranscript) *model.AppError
	SetHeartbeat(id string) *model.AppError

	LastBridged(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError)
	SetGranteeId(domainId int64, id string, granteeId int64) *model.AppError
	SetUserId(domainId int64, id string, userId int64) *model.AppError
	SetBlindTransfer(domainId int64, id, destination string) *model.AppError
	SetContactId(domainId int64, id string, contactId int64) *model.AppError
	SetVariables(id string, vars *model.CallVariables) *model.AppError
	SaveMediaStats(stats *model.CallActionMediaStats) *model.AppError
	SaveProgress(ctx context.Context, p *model.CallActionProgress) *model.AppError
}

type SchemaStore interface {
	Get(domainId int64, id int) (*model.Schema, *model.AppError)
	GetUpdatedAt(domainId int64, id int) (int64, *model.AppError)
	GetTransferredRouting(domainId int64, schemaId int) (*model.Routing, *model.AppError)
	GetVariable(domainId int64, name string) (*model.SchemaVariable, *model.AppError)
	SetVariable(domainId int64, name string, val *model.SchemaVariable) *model.AppError
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
	GetPlaybackFile(domainId int64, req *model.PlaybackFile) (*model.PlaybackFile, *model.AppError)
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
	ProfileType(domainId int64, profileId int) (string, *model.AppError)
}

type QueueStore interface {
	HistoryStatistics(domainId int64, search *model.SearchQueueCompleteStatistics) (float64, *model.AppError)
	GetQueueData(domainId int64, search *model.SearchEntity, mapRes model.Variables) (model.Variables, *model.AppError)
	GetQueueAgents(domainId int64, queueId int, channel string, mapRes model.Variables) (model.Variables, *model.AppError)
	FindQueueByName(domainId int64, name string) (int32, *model.AppError)
}

type MemberStore interface {
	CreateMember(domainId int64, queueId, holdSec int, member *model.CallbackMember) *model.AppError
	CallPosition(callId string) (int64, *model.AppError)
	EWTPuzzle(domainId int64, callId string, min int, queueIds, bucketIds []int) (float64, *model.AppError)
	GetProperties(domainId int64, req *model.SearchMember, mapRes model.Variables) (model.Variables, *model.AppError)
	PatchMembers(domainId int64, req *model.SearchMember, patch *model.PatchMember) (int, *model.AppError)
}

type LogStore interface {
	Save(schemaId int, connId string, log any) *model.AppError
}

type FileStore interface {
	GetMetadata(domainId int64, ids []int64) ([]model.File, *model.AppError)
}

type WebHookStore interface {
	Get(id string) (model.WebHook, *model.AppError)
}

type SystemcSettings interface {
	Get(ctx context.Context, domainId int64, name string) (json.RawMessage, *model.AppError)
}
