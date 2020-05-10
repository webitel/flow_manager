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
}
