package channel

import (
	"sync"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/infrastructure/discovery"
)

type server struct {
	receiver        <-chan flow.ChannelExec
	consume         chan flow.Connection
	didFinishListen chan struct{}
	stopped         chan struct{}
	startOnce       sync.Once
}

func New(receiver <-chan flow.ChannelExec) flow.Server {
	return &server{
		receiver:        receiver,
		consume:         make(chan flow.Connection, 100),
		didFinishListen: make(chan struct{}),
		stopped:         make(chan struct{}),
	}
}

func (s *server) Name() string {
	return "Channel queue"
}

func (s *server) Start() error {
	s.startOnce.Do(func() {
		go s.listen()
	})
	return nil
}

func (s *server) Stop() {
	close(s.didFinishListen)
	<-s.stopped
}

func (s *server) Host() string {
	return ""
}

func (s *server) Port() int {
	return 0
}

func (s *server) Consume() <-chan flow.Connection {
	return s.consume
}

func (s *server) Type() flow.ConnectionType {
	return flow.ConnectionTypeCall
}

func (s *server) Cluster(discovery discovery.ServiceDiscovery) error {
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
				if c.DomainId == 0 || c.SchemaId == 0 {
					wlog.Warn("channel connection required domain_id & schema_id")
				} else {
					s.consume <- newConnection(c)
				}
			}

		}
	}
}
