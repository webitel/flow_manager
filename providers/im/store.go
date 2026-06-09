package im

import (
	"errors"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/webitel/wlog"
)

const (
	cacheClientsSize = 10000
	cacheExpire      = 24 * 10 * time.Hour
)

var ErrClientNotFound = errors.New("connection not found in cache")

type ConnectionStore struct {
	log   *wlog.Logger
	conns *expirable.LRU[string, *Connection]
}

func NewConnectionStore(log *wlog.Logger) *ConnectionStore {
	conns := expirable.NewLRU[string, *Connection](cacheClientsSize, nil, cacheExpire)

	return &ConnectionStore{
		log:   log,
		conns: conns,
	}
}

func (s *ConnectionStore) Get(id string) (*Connection, bool) {
	if conn, ok := s.conns.Get(id); ok {
		s.log.Debug("connection cache hit", wlog.String("id", id))
		return conn, true
	}

	s.log.Debug("connection cache miss", wlog.String("id", id))

	return nil, false
}

func (s *ConnectionStore) Add(conn *Connection) {
	s.log.Debug("adding new connection to cache", wlog.String("id", conn.Id()))
	s.conns.Add(conn.Id(), conn)
}

func (s *ConnectionStore) Delete(conn *Connection) {
	s.log.Debug("delete connection from cache", wlog.String("id", conn.Id()))
	s.conns.Remove(conn.Id())
}
