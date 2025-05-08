package fs

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/fs/eventsocket"
	"github.com/webitel/wlog"
)

const (
	EVENT_HANGUP_COMPLETE  = "CHANNEL_HANGUP_COMPLETE"
	EVENT_EXECUTE_COMPLETE = "CHANNEL_EXECUTE_COMPLETE"
	EVENT_ANSWER           = "CHANNEL_ANSWER"
	EVENT_BRIDGE           = "CHANNEL_BRIDGE"
)

type Config struct {
	Host           string
	Port           int
	RecordResample int
}

type server struct {
	cfg             *Config
	didFinishListen chan struct{}
	consume         chan model.Connection
	listener        net.Listener
	stopped         bool
	sync.RWMutex

	log *wlog.Logger
}

func NewServer(cfg *Config) model.Server {
	return &server{
		cfg:             cfg,
		didFinishListen: make(chan struct{}),
		consume:         make(chan model.Connection),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "fs server"),
		),
	}
}

func (s server) Name() string {
	return "FreeSWITCH"
}

func (s *server) Cluster(discovery discovery.ServiceDiscovery) *model.AppError {
	return nil
}

func (s server) Type() model.ConnectionType {
	return model.ConnectionTypeCall
}

func (s *server) Start() *model.AppError {
	address := s.getAddress()
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return model.NewAppError(s.Name(), "fs.start_server.error", nil, err.Error(), http.StatusInternalServerError)
	}

	// todo validate ?
	if s.cfg.RecordResample != 0 {
		s.log.Info(fmt.Sprintf("recordings resample to %d Hz", s.cfg.RecordResample))
	}

	s.listener = lis
	go s.listen(lis)
	return nil
}

func (s *server) listen(lis net.Listener) {
	defer s.log.Debug("close listening")
	s.log.Info(fmt.Sprintf("server listening %s", lis.Addr().String()))

	err := eventsocket.Listen(lis, s.handleConnection)
	s.RLock()
	stopped := s.stopped
	s.RUnlock()

	if err != nil && !stopped {
		s.log.With(
			wlog.String("address", lis.Addr().String()),
		).Err(err)

		panic(err.Error())
	}
	close(s.didFinishListen)
}

func (s *server) Stop() {
	s.Lock()
	s.stopped = true
	s.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}
	close(s.consume)
	<-s.didFinishListen
}

func (s *server) Host() string {
	return s.cfg.Host
}

func (s *server) Port() int {
	return s.cfg.Port
}

func (s *server) getAddress() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func (s *server) Consume() <-chan model.Connection {
	return s.consume
}

func (s *server) handleConnection(c *eventsocket.Connection) {
	e, err := c.Send("connect")
	if err != nil {
		wlog.Error(fmt.Sprintf("connect to call %v error: %s", c.RemoteAddr(), err.Error()))
		return
	}

	uuid := e.Get(HEADER_ID_NAME)

	_, err = c.Send("linger 30")
	if err != nil {
		s.log.Error("linger error: "+err.Error(), wlog.String("call_id", uuid))
		return
	}

	_, err = c.Send("filter unique-id " + uuid)
	if err != nil {
		s.log.Error("filter error: "+err.Error(), wlog.String("call_id", uuid))
		return
	}

	_, err = c.Send(fmt.Sprintf("events plain %s %s %s %s", EVENT_HANGUP_COMPLETE, EVENT_EXECUTE_COMPLETE, EVENT_ANSWER, EVENT_BRIDGE))
	if err != nil {
		s.log.Error("events error: "+err.Error(), wlog.String("call_id", uuid))
		return
	}

	connection := newConnection(c, e)
	connection.resample = s.cfg.RecordResample

	defer func() {

		if connection.Stopped() {
			connection.log.Debug("stopped connection")
		} else {
			connection.log.Warn("bad close connection")
		}

		connection.Lock()
		connection.closeHookBridge()
		if len(connection.callbackMessages) > 0 {
			for k, v := range connection.callbackMessages {
				v <- &eventsocket.Event{}
				close(v)
				delete(connection.callbackMessages, k)
			}
		}
		connection.Unlock()

		if connection.lastEvent.Get(HEADER_EVENT_NAME) != EVENT_HANGUP_COMPLETE {
			connection.log.Warn("no found event hangup")
		}

		connection.connection.Close()
		connection.cancelFn()
	}()

	connection.log.Debug("start call")
	s.consume <- connection

	for {
		if connection.Stopped() {
			break
		}

		e, err = c.ReadEvent()
		if err == io.EOF {
			return
		} else if err != nil {
			connection.log.Error("socket error: " + err.Error())
			continue
		}

		connection.setEvent(e)
	}
}
