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
