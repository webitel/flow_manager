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
}

type CallStore interface {
	Save(call *model.CallActionRinging) *model.AppError
	SetState(call *model.CallAction) *model.AppError
}

type SchemaStore interface {
	Get(domainId, id int) (*model.Schema, *model.AppError)
}

type CallRoutingStore interface {
	FromGateway(domainId, gatewayId int) (*model.Routing, *model.AppError)
	SearchToDestination(domainId int, destination string) (*model.Routing, *model.AppError)
}

type EndpointStore interface {
	Get(domainId int64, callerName, callerNumber string, endpoints model.Applications) ([]*model.Endpoint, *model.AppError)
}
