package sqlstore

import (
	infraSql "github.com/webitel/flow_manager/infra/sql"
	postgresStorage "github.com/webitel/flow_manager/internal/storage/postgres"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

type SqlSupplierOldStores struct {
	call          store.CallStore
	schema        store.SchemaStore
	callRouting   store.CallRoutingStore
	endpoint      store.EndpointStore
	email         store.EmailStore
	media         store.MediaStore
	calendar      store.CalendarStore
	list          store.ListStore
	chat          store.ChatStore
	queue         store.QueueStore
	member        store.MemberStore
	user          store.UserStore
	log           store.LogStore
	file          store.FileStore
	webHook       store.WebHookStore
	sysSettings   store.SystemcSettings
	socketSession store.SocketSessionStore
	session       store.SessionStore
}

type SqlSupplier struct {
	oldStores SqlSupplierOldStores
	settings  *model.SqlSettings
}

func NewSqlSupplier(settings model.SqlSettings, db infraSql.Store) *SqlSupplier {
	supplier := &SqlSupplier{
		settings: &settings,
	}

	supplier.oldStores.schema = postgresStorage.NewSchemaRepository(db)
	supplier.oldStores.webHook = postgresStorage.NewWebHookRepository(db)
	supplier.oldStores.media = postgresStorage.NewMediaRepository(db)
	supplier.oldStores.call = postgresStorage.NewCallRepository(db)
	supplier.oldStores.callRouting = postgresStorage.NewCallRoutingRepository(db)
	supplier.oldStores.endpoint = postgresStorage.NewEndpointRepository(db)
	supplier.oldStores.email = postgresStorage.NewEmailRepository(db)
	supplier.oldStores.calendar = postgresStorage.NewCalendarRepository(db)
	supplier.oldStores.list = postgresStorage.NewListRepository(db)
	supplier.oldStores.chat = postgresStorage.NewChatRepository(db)
	supplier.oldStores.queue = postgresStorage.NewQueueRepository(db)
	supplier.oldStores.member = postgresStorage.NewMemberRepository(db)
	supplier.oldStores.user = postgresStorage.NewUserRepository(db)
	supplier.oldStores.log = postgresStorage.NewLogRepository(db)
	supplier.oldStores.file = postgresStorage.NewFileRepository(db)
	supplier.oldStores.sysSettings = postgresStorage.NewSysSettingsRepository(db)
	supplier.oldStores.socketSession = postgresStorage.NewSocketSessionRepository(db)
	supplier.oldStores.session = postgresStorage.NewSessionRepository(db)

	return supplier
}

func (ss *SqlSupplier) Call() store.CallStore {
	return ss.oldStores.call
}

func (ss *SqlSupplier) Schema() store.SchemaStore {
	return ss.oldStores.schema
}

func (ss *SqlSupplier) CallRouting() store.CallRoutingStore {
	return ss.oldStores.callRouting
}

func (ss *SqlSupplier) Endpoint() store.EndpointStore {
	return ss.oldStores.endpoint
}

func (ss *SqlSupplier) Email() store.EmailStore {
	return ss.oldStores.email
}

func (ss *SqlSupplier) Media() store.MediaStore {
	return ss.oldStores.media
}

func (ss *SqlSupplier) Calendar() store.CalendarStore {
	return ss.oldStores.calendar
}

func (ss *SqlSupplier) List() store.ListStore {
	return ss.oldStores.list
}

func (ss *SqlSupplier) Chat() store.ChatStore {
	return ss.oldStores.chat
}

func (ss *SqlSupplier) Queue() store.QueueStore {
	return ss.oldStores.queue
}

func (ss *SqlSupplier) Member() store.MemberStore {
	return ss.oldStores.member
}

func (ss *SqlSupplier) User() store.UserStore {
	return ss.oldStores.user
}

func (ss *SqlSupplier) Log() store.LogStore {
	return ss.oldStores.log
}

func (ss *SqlSupplier) File() store.FileStore {
	return ss.oldStores.file
}

func (ss *SqlSupplier) WebHook() store.WebHookStore {
	return ss.oldStores.webHook
}

func (ss *SqlSupplier) SystemcSettings() store.SystemcSettings {
	return ss.oldStores.sysSettings
}

func (ss *SqlSupplier) SocketSession() store.SocketSessionStore {
	return ss.oldStores.socketSession
}

func (ss *SqlSupplier) Session() store.SessionStore {
	return ss.oldStores.session
}
