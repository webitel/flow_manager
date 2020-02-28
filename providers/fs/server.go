package fs

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/fs/eventsocket"
	"github.com/webitel/wlog"
	"io"
	"net"
	"net/http"
)

const (
	EVENT_HANGUP_COMPLETE  = "CHANNEL_HANGUP_COMPLETE"
	EVENT_EXECUTE_COMPLETE = "CHANNEL_EXECUTE_COMPLETE"
	EVENT_ANSWER           = "CHANNEL_ANSWER"
)

type Config struct {
	Host string
	Port int
}

type server struct {
	cfg             *Config
	didFinishListen chan struct{}
	consume         chan model.Connection
	listener        net.Listener
}

func NewServer(cfg *Config) model.Server {
	return &server{
		cfg:             cfg,
		didFinishListen: make(chan struct{}),
		consume:         make(chan model.Connection),
	}
}

func (s server) Name() string {
	return "FreeSWITCH"
}

func (s *server) GetApplication(string) (*model.Application, *model.AppError) {
	return nil, nil //FIXME
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
	s.listener = lis
	go s.listen(lis)
	return nil
}

func (s *server) listen(lis net.Listener) {
	defer wlog.Debug(fmt.Sprintf("[%s] close server listening", s.Name()))
	wlog.Debug(fmt.Sprintf("[%s] server listening %s", s.Name(), lis.Addr().String()))

	eventsocket.Listen(lis, s.handleConnection)
	close(s.didFinishListen)
}

func (s *server) Stop() {
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
		wlog.Error(fmt.Sprintf("set linger call %s error: %s", uuid, err.Error()))
		return
	}

	_, err = c.Send("filter unique-id " + uuid)
	if err != nil {
		wlog.Error(fmt.Sprintf("call %s filter events error: %s", uuid, err.Error()))
		return
	}

	_, err = c.Send(fmt.Sprintf("events plain %s %s %s", EVENT_HANGUP_COMPLETE, EVENT_EXECUTE_COMPLETE, EVENT_ANSWER))
	if err != nil {
		wlog.Error(fmt.Sprintf("call %s events error: %s", uuid, err.Error()))
		return
	}

	connection := newConnection(c, e)

	defer func() {

		if connection.Stopped() {
			wlog.Debug(fmt.Sprintf("call %s stopped connect %v", uuid, c.RemoteAddr()))
		} else {
			wlog.Warn(fmt.Sprintf("call %s bad close connection %v", uuid, c.RemoteAddr()))
		}

		connection.Lock()
		if len(connection.callbackMessages) > 0 {
			for k, v := range connection.callbackMessages {
				v <- &eventsocket.Event{}
				close(v)
				delete(connection.callbackMessages, k)
			}
		}
		connection.Unlock()

		if connection.lastEvent.Get(HEADER_EVENT_NAME) != EVENT_HANGUP_COMPLETE {
			wlog.Warn(fmt.Sprintf("call %s no found event hangup", connection.Id()))
		}

		connection.connection.Close()
		close(connection.disconnected)
	}()

	wlog.Debug(fmt.Sprintf("receive new call %s connect %v", uuid, c.RemoteAddr()))
	s.consume <- connection

	for {
		if connection.Stopped() {
			break
		}

		e, err = c.ReadEvent()
		if err == io.EOF {
			return
		} else if err != nil {
			wlog.Error(fmt.Sprintf("call %s socket error: %s", uuid, err.Error()))
			continue
		}

		connection.setEvent(e)
	}
}
