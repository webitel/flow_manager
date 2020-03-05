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

func (s *LayeredStore) Schema() SchemaStore {
	return s.DatabaseLayer.Schema()
}

func (s *LayeredStore) CallRouting() CallRoutingStore {
	return s.DatabaseLayer.CallRouting()
}

func (s *LayeredStore) Endpoint() EndpointStore {
	return s.DatabaseLayer.Endpoint()
}
