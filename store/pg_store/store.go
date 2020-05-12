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

	Call() store.CallStore
	Schema() store.SchemaStore
	CallRouting() store.CallRoutingStore
	Endpoint() store.EndpointStore
	Email() store.EmailStore
	Media() store.MediaStore
	Calendar() store.CalendarStore
	List() store.ListStore
	Chat() store.ChatStore
}
