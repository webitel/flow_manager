package sqlstore

import (
	"github.com/go-gorp/gorp"
	_ "github.com/lib/pq"
	"github.com/webitel/flow_manager/store"
)

type SqlStore interface {
	GetMaster() *gorp.DbMap
	GetReplica() *gorp.DbMap
	GetAllConns() []*gorp.DbMap

	Schema() store.SchemaStore
	CallRouting() store.CallRoutingStore
	Endpoint() store.EndpointStore
}
