package postgres

import (
	infraSql "github.com/webitel/flow_manager/internal/infrastructure/sql"
	"github.com/webitel/flow_manager/internal/storage"
)

// pgStore implements storage.Store by aggregating all pgx-backed repositories.
type pgStore struct {
	call          storage.CallStore
	schema        storage.SchemaStore
	callRouting   storage.CallRoutingStore
	endpoint      storage.EndpointStore
	email         storage.EmailStore
	media         storage.MediaStore
	calendar      storage.CalendarStore
	list          storage.ListStore
	chat          storage.ChatStore
	queue         storage.QueueStore
	member        storage.MemberStore
	user          storage.UserStore
	log           storage.LogStore
	file          storage.FileStore
	webHook       storage.WebHookStore
	sysSettings   storage.SystemcSettings
	socketSession storage.SocketSessionStore
	session       storage.SessionStore
}

func NewStore(db infraSql.Store) storage.Store {
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

func (s *pgStore) Call() storage.CallStore                   { return s.call }
func (s *pgStore) Schema() storage.SchemaStore               { return s.schema }
func (s *pgStore) CallRouting() storage.CallRoutingStore     { return s.callRouting }
func (s *pgStore) Endpoint() storage.EndpointStore           { return s.endpoint }
func (s *pgStore) Email() storage.EmailStore                 { return s.email }
func (s *pgStore) Media() storage.MediaStore                 { return s.media }
func (s *pgStore) Calendar() storage.CalendarStore           { return s.calendar }
func (s *pgStore) List() storage.ListStore                   { return s.list }
func (s *pgStore) Chat() storage.ChatStore                   { return s.chat }
func (s *pgStore) Queue() storage.QueueStore                 { return s.queue }
func (s *pgStore) Member() storage.MemberStore               { return s.member }
func (s *pgStore) User() storage.UserStore                   { return s.user }
func (s *pgStore) Log() storage.LogStore                     { return s.log }
func (s *pgStore) File() storage.FileStore                   { return s.file }
func (s *pgStore) WebHook() storage.WebHookStore             { return s.webHook }
func (s *pgStore) SystemcSettings() storage.SystemcSettings  { return s.sysSettings }
func (s *pgStore) SocketSession() storage.SocketSessionStore { return s.socketSession }
func (s *pgStore) Session() storage.SessionStore             { return s.session }
