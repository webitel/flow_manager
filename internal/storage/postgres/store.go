package postgres

import (
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/store"
)

// pgStore implements store.Store by aggregating all pgx-backed repositories.
type pgStore struct {
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

func NewStore(db infraSql.Store) store.Store {
	return &pgStore{
		call:          NewCallRepository(db),
		schema:        NewSchemaRepository(db),
		callRouting:   NewCallRoutingRepository(db),
		endpoint:      NewEndpointRepository(db),
		email:         NewEmailRepository(db),
		media:         NewMediaRepository(db),
		calendar:      NewCalendarRepository(db),
		list:          NewListRepository(db),
		chat:          NewChatRepository(db),
		queue:         NewQueueRepository(db),
		member:        NewMemberRepository(db),
		user:          NewUserRepository(db),
		log:           NewLogRepository(db),
		file:          NewFileRepository(db),
		webHook:       NewWebHookRepository(db),
		sysSettings:   NewSysSettingsRepository(db),
		socketSession: NewSocketSessionRepository(db),
		session:       NewSessionRepository(db),
	}
}

func (s *pgStore) Call() store.CallStore                   { return s.call }
func (s *pgStore) Schema() store.SchemaStore               { return s.schema }
func (s *pgStore) CallRouting() store.CallRoutingStore     { return s.callRouting }
func (s *pgStore) Endpoint() store.EndpointStore           { return s.endpoint }
func (s *pgStore) Email() store.EmailStore                 { return s.email }
func (s *pgStore) Media() store.MediaStore                 { return s.media }
func (s *pgStore) Calendar() store.CalendarStore           { return s.calendar }
func (s *pgStore) List() store.ListStore                   { return s.list }
func (s *pgStore) Chat() store.ChatStore                   { return s.chat }
func (s *pgStore) Queue() store.QueueStore                 { return s.queue }
func (s *pgStore) Member() store.MemberStore               { return s.member }
func (s *pgStore) User() store.UserStore                   { return s.user }
func (s *pgStore) Log() store.LogStore                     { return s.log }
func (s *pgStore) File() store.FileStore                   { return s.file }
func (s *pgStore) WebHook() store.WebHookStore             { return s.webHook }
func (s *pgStore) SystemcSettings() store.SystemcSettings  { return s.sysSettings }
func (s *pgStore) SocketSession() store.SocketSessionStore { return s.socketSession }
func (s *pgStore) Session() store.SessionStore             { return s.session }
