package store

import (
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
}

type EmailStore interface {
	Save(domainId int64, m *model.Email) *model.AppError
	ProfileTaskFetch(node string) ([]*model.EmailProfileTask, *model.AppError)
	GetProfile(id int) (*model.EmailProfile, *model.AppError)
}

type CallStore interface {
	Save(call *model.CallActionRinging) *model.AppError
	SetState(call *model.CallAction) *model.AppError
	SetBridged(call *model.CallActionBridge) *model.AppError
	SetHangup(call *model.CallActionHangup) *model.AppError
	MoveToHistory() *model.AppError

	AddMemberToQueueQueue(domainId int64, queueId int, number, name string, typeId, holdSec int, variables map[string]string) *model.AppError
}

type SchemaStore interface {
	Get(domainId int64, id int) (*model.Schema, *model.AppError)
	GetUpdatedAt(id int) (int64, *model.AppError)
}

type CallRoutingStore interface {
	FromGateway(domainId int64, gatewayId int) (*model.Routing, *model.AppError)
	SearchToDestination(domainId int64, destination string) (*model.Routing, *model.AppError)
}

type EndpointStore interface {
	Get(domainId int64, callerName, callerNumber string, endpoints model.Applications) ([]*model.Endpoint, *model.AppError)
}

type MediaStore interface {
	GetFiles(domainId int64, req *[]*model.PlaybackFile) ([]*model.PlaybackFile, *model.AppError)
}

type CalendarStore interface {
	Check(domainId int64, id *int, name *string) (*model.Calendar, *model.AppError)
	GetTimezones() ([]*model.Timezone, *model.AppError)
}

type ListStore interface {
	CheckNumber(domainId int64, number string, listId *int, listName *string) (bool, *model.AppError)
}

type ChatStore interface {
	Get(channelId string) (*model.ConversationInfo, *model.AppError)
	CreateConversation(secretKey string, title string, name string, body model.PostBody) (model.ConversationInfo, *model.AppError)
	ConversationUnreadMessages(channelId string, limit int) ([]*model.ConversationMessage, *model.AppError)
	ConversationPostMessage(channelId string, body model.PostBody) ([]*model.ConversationMessage, *model.AppError)
	ConversationHistory(channelId string, limit, offset int) ([]*model.ConversationMessage, *model.AppError)
	Join(parentChannelId string, name string) ([]*model.ConversationMessageJoined, *model.AppError)
	Close(channelId string) *model.AppError
}
