package store

import "context"

type LayeredStoreDatabaseLayer interface {
	Store
}

type LayeredStore struct {
	TmpContext    context.Context
	DatabaseLayer LayeredStoreDatabaseLayer
}

func NewLayeredStore(db LayeredStoreDatabaseLayer) Store {
	store := &LayeredStore{
		TmpContext:    context.TODO(),
		DatabaseLayer: db,
	}

	return store
}

func (s *LayeredStore) Call() CallStore {
	return s.DatabaseLayer.Call()
}

func (s *LayeredStore) Schema() SchemaStore {
	return s.DatabaseLayer.Schema()
}

func (s *LayeredStore) CallRouting() CallRoutingStore {
	return s.DatabaseLayer.CallRouting()
}

func (s *LayeredStore) Endpoint() EndpointStore {
	return s.DatabaseLayer.Endpoint()
}

func (s *LayeredStore) Email() EmailStore {
	return s.DatabaseLayer.Email()
}

func (s *LayeredStore) Media() MediaStore {
	return s.DatabaseLayer.Media()
}

func (s *LayeredStore) Calendar() CalendarStore {
	return s.DatabaseLayer.Calendar()
}

func (s *LayeredStore) List() ListStore {
	return s.DatabaseLayer.List()
}

func (s *LayeredStore) Chat() ChatStore {
	return s.DatabaseLayer.Chat()
}

func (s *LayeredStore) Queue() QueueStore {
	return s.DatabaseLayer.Queue()
}

func (s *LayeredStore) Member() MemberStore {
	return s.DatabaseLayer.Member()
}

func (s *LayeredStore) User() UserStore {
	return s.DatabaseLayer.User()
}

func (s *LayeredStore) Log() LogStore {
	return s.DatabaseLayer.Log()
}

func (s *LayeredStore) File() FileStore {
	return s.DatabaseLayer.File()
}

func (s *LayeredStore) WebHook() WebHookStore {
	return s.DatabaseLayer.WebHook()
}

func (s *LayeredStore) SystemcSettings() SystemcSettings {
	return s.DatabaseLayer.SystemcSettings()
}

func (s *LayeredStore) SocketSession() SocketSessionStore {
	return s.DatabaseLayer.SocketSession()
}

func (s *LayeredStore) Session() SessionStore {
	return s.DatabaseLayer.Session()
}
