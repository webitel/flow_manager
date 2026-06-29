package im

import (
	"crypto/tls"
	"sync"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/model"
)

const JWTPayloadVar string = "jwt"

type SessionStore interface {
	Touch(id, appId string) (*int, error)
	Remove(id, appId string) error
	RemoveAll(appId string) error
}

type server struct {
	id              string
	receiver        <-chan model.IMEventWrapper
	consume         chan model.Connection
	didFinishListen chan struct{}
	stopped         chan struct{}
	startOnce       sync.Once
	client          *Client
	log             *wlog.Logger
	connectionStore *ConnectionStore
	sessionStore    SessionStore
}

func NewServer(id, consulAddr string, receiver <-chan model.IMEventWrapper, log *wlog.Logger, t *tls.Config, store SessionStore) model.Server {
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

func (s *server) Name() string { return "IM" }

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
		wlog.Debug("stop listen IM channel server...")
		close(s.stopped)
	}()

	wlog.Debug("start listen IM channel")

	for {
		select {
		case <-s.didFinishListen:
			return
		case c, ok := <-s.receiver:
			if !ok {
				continue //? switch to return or break to skip infinity loop?
			}

			if c.GetPayload().GetThreadID() == "" {
				s.log.Warn("received message with empty thread ID", wlog.String("message_id", c.GetPayload().MessageID()))
				continue
			}

			if c.GetType() == model.IMEventTypeBotControlReleased {
				s.handleBotControlReleased(c)
				continue
			}

			if err := s.nodeMessage(c); err != nil {
				s.log.Error("handling message", wlog.String("message_id", c.GetPayload().MessageID()), wlog.Err(err))
			}
		}
	}
}

// handleBotControlReleased stops the running schema(s) for a thread when a user releases
// bot control (the "/close" command). Other release reasons (e.g. a schema that completed
// on its own) are ignored — there is nothing left to cancel.
func (s *server) handleBotControlReleased(msg model.IMEventWrapper) {
	released, ok := msg.GetPayload().(model.BotControlReleased)
	if !ok {
		s.log.Warn("bot control released: unexpected payload type")

		return
	}

	if released.Reason != model.BotControlReasonClientLeave {
		return
	}

	threadID := released.GetThreadID()
	broken := s.connectionStore.BreakByThread(threadID)

	s.log.Debug("bot control released: stopped running schema",
		wlog.String("thread_id", threadID),
		wlog.Int("connections", broken),
	)
}

func (s *server) stopConnection(c *Connection) {
	c.srv.connectionStore.Delete(c)
	err := s.sessionStore.Remove(c.id, s.id)
	if err != nil {
		s.log.Warn("failed to remove session store connection")
	}
}

const IMUserTypeBot string = "bot"

func (s *server) nodeMessage(msg model.IMEventWrapper) error {
	if msg.GetPayload().Sender().Issuer == IMUserTypeBot {
		return nil
	}

	for _, endpoint := range msg.GetPayload().Receivers() {
		if endpoint.Issuer != IMUserTypeBot {
			continue
		}

		compositeSessionID := msg.GetPayload().GetThreadID() + "." + endpoint.Sub

		if conn, ok := s.connectionStore.Get(compositeSessionID); ok {
			conn.OnMessage(msg)
			continue
		}

		seq, err := s.sessionStore.Touch(compositeSessionID, s.id)
		if err != nil {
			return err
		}

		if seq == nil {
			return nil
		}

		if *seq > 1 {
			s.log.Warn("received message with sequance thread ID", wlog.Int("sequance", *seq))
		}

		dialog := newConnection(s, compositeSessionID, endpoint, msg)
		dialog.setupVariables()

		s.connectionStore.Add(dialog)
		dialog.log.Debug("start dialog " + compositeSessionID)
		s.consume <- dialog
	}

	return nil
}
