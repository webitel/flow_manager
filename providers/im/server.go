package im

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/infra/discovery"
	outboundim "github.com/webitel/flow_manager/internal/adapters/outbound/im"
	"github.com/webitel/flow_manager/model"
)

// SessionStore is the distributed ownership-claim store for IM connections.
// A DB-backed implementation prevents two nodes from processing the same
// conversation simultaneously.
type SessionStore interface {
	Touch(id, appId string) (*int, error)
	Remove(id, appId string) error
	RemoveAll(appId string) error
}

type server struct {
	id              string
	receiver        <-chan model.MessageWrapper
	consume         chan model.Connection
	didFinishListen chan struct{}
	stopped         chan struct{}
	startOnce       sync.Once
	client          *outboundim.Client
	log             *wlog.Logger
	connectionStore *ConnectionStore
	dispatcher      *Dispatcher
}

func NewServer(id, consulAddr string, receiver <-chan model.MessageWrapper, log *wlog.Logger, t *tls.Config, store SessionStore) model.Server {
	consume := make(chan model.Connection, 100)
	connStore := NewConnectionStore(log)

	s := &server{
		id:              id,
		receiver:        receiver,
		consume:         consume,
		didFinishListen: make(chan struct{}),
		stopped:         make(chan struct{}),
		client:          outboundim.NewClient(consulAddr, log, t),
		connectionStore: connStore,
		log:             log,
	}
	s.dispatcher = newDispatcher(id, connStore, store, consume, log, s)
	return s
}

func (s *server) Name() string {
	return "IM"
}

func (s *server) Start() error {
	s.startOnce.Do(func() {
		if err := s.client.Start(); err != nil {
			panic(err)
		}
		if err := s.dispatcher.Startup(); err != nil {
			panic(err)
		}
		go s.listen()
	})
	return nil
}

func (s *server) Stop() {
	close(s.didFinishListen)
	s.client.Stop()
	s.dispatcher.Shutdown()
	<-s.stopped
}

func (s *server) Host() string {
	return ""
}

func (s *server) Port() int {
	return 0
}

func (s *server) Consume() <-chan model.Connection {
	return s.consume
}

func (s *server) Type() model.ConnectionType {
	return model.ConnectionTypeIM
}

func (s *server) Cluster(discovery discovery.ServiceDiscovery) error {
	return nil
}

func (s *server) listen() {
	defer func() {
		wlog.Debug("stop listen im server...")
		close(s.stopped)
	}()

	wlog.Debug("start listen im")

	for {
		select {
		case <-s.didFinishListen:
			return
		case c, ok := <-s.receiver:
			if !ok {
				return
			}
			if c.Message.ThreadID == "" {
				s.log.Warn(fmt.Sprintf("received message with empty thread id %v", c))
				continue
			}
			if err := s.dispatcher.Handle(c); err != nil {
				s.log.Warn(fmt.Sprintf("im dispatcher error for msg %v: %v", c, err))
			}
		}
	}
}

func (s *server) stopConnection(c *Connection) {
	s.dispatcher.Unregister(c)
}
