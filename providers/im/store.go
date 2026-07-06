package im

import (
	"errors"
	"strings"
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

// BreakByThread cancels every running connection belonging to the given thread and
// returns how many were broken. Connection ids are "<threadID>.<schemaSub>", so the
// thread is matched by the "<threadID>." prefix. The schema goroutine then unwinds and
// removes itself from the cache via Stop().
func (s *ConnectionStore) BreakByThread(threadID string) int {
	if threadID == "" {
		return 0
	}

	prefix := threadID + "."

	var broken int
	for _, key := range s.conns.Keys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		if conn, ok := s.conns.Get(key); ok {
			conn.Break()
			broken++
		}
	}

	return broken
}
