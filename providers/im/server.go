package im

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

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
	client          *Client
	log             *wlog.Logger
	connectionStore *ConnectionStore
	sessionStore    SessionStore
}

func NewServer(id, consulAddr string, receiver <-chan model.MessageWrapper, log *wlog.Logger, t *tls.Config, store SessionStore) model.Server {
	return &server{
		id:              id,
		receiver:        receiver,
		consume:         make(chan model.Connection, 100),
		didFinishListen: make(chan struct{}),
		stopped:         make(chan struct{}),
		client:          NewClient(consulAddr, log, t),
		sessionStore:    store,
		connectionStore: NewConnectionStore(log),
		log:             log,
	}
}

func (s *server) Name() string {
	return "IM"
}

func (s *server) Start() *model.AppError {
	s.startOnce.Do(func() {
		err := s.client.Start()
		if err != nil {
			panic(err)
		}

		err = s.sessionStore.RemoveAll(s.id)
		if err != nil {
			panic(err)
		}

		go s.listen()
	})
	return nil
}

func (s *server) Stop() {
	close(s.didFinishListen)
	s.client.Stop()

	err := s.sessionStore.RemoveAll(s.id)
	if err != nil {
		s.log.Error("failed to remove session store", wlog.Err(err))
	}
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

func (s *server) Cluster(discovery discovery.ServiceDiscovery) *model.AppError {
	return nil
}

func (s *server) listen() {
	defer func() {
		wlog.Debug("stop listen channel server...")
		close(s.stopped)
	}()

	wlog.Debug("start listen channel")

	for {
		select {
		case <-s.didFinishListen:
			return
		case c, ok := <-s.receiver:
			if ok {
				if c.Message.ThreadID == "" {
					s.log.Warn(fmt.Sprintf("received message with empty thread id %v", c))
					continue
				}
				if err := s.nodeMessage(c); err != nil {
					s.log.Warn(fmt.Sprintf("failed to handle message %v: %v", c, err))
				}
			}
		}
	}
}

func (s *server) stopConnection(c *Connection) {
	c.srv.connectionStore.Delete(c)
	err := s.sessionStore.Remove(c.id, s.id)
	if err != nil {
		s.log.Warn("failed to remove session store connection")
	}
}

func (s *server) nodeMessage(msg model.MessageWrapper) error {
	if msg.Message.From.Sub == "2522" {
		println("todo: skip my message")
		return nil
	}

	conn, ok := s.connectionStore.Get(msg.Message.ThreadID)
	if ok {
		conn.OnMessage(msg)
		return nil
	}

	seq, err := s.sessionStore.Touch(msg.Message.ThreadID, s.id)
	if err != nil {
		return err
	}
	if seq == nil {
		return nil
	}

	if *seq > 1 {
		s.log.Warn(fmt.Sprintf("received message with seq thread id %v", *seq))
	}

	dialog := newConnection(s, msg)
	s.connectionStore.Add(dialog)
	dialog.log.Debug("start dialog")
	s.consume <- dialog

	return nil
}
