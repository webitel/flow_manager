package store

import (
	"database/sql"
	"github.com/webitel/flow_manager/model"
)

var ErrNoRows = sql.ErrNoRows

type Store interface {
	Schema() SchemaStore
	CallRouting() CallRoutingStore
}

type SchemaStore interface {
	Get(domainId, id int) (*model.Schema, *model.AppError)
}

type CallRoutingStore interface {
	FromGateway(domainId, gatewayId int) (*model.Routing, *model.AppError)
}
