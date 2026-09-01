package im

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"sync"

	"google.golang.org/grpc/metadata"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/model"
)

type SessionStore interface {
	Touch(id, appId string) (*int, error)
	Remove(id, appId string) error
	RemoveByThread(threadID string) error
	RemoveAll(appId string) error
}

type server struct {
	id              string
	receiver        <-chan any
	consume         chan model.Connection
	didFinishListen chan struct{}
	stopped         chan struct{}
	startOnce       sync.Once
	client          *Client
	log             *wlog.Logger
	connectionStore *ConnectionStore
	sessionStore    SessionStore
	gateFactory     *GateHandlerFactory
}

func NewServer(id, consulAddr string, receiver <-chan any, log *wlog.Logger, t *tls.Config, store SessionStore) model.Server {
	client := NewClient(consulAddr, log, t)
	fabric := NewGateHandlerFactory(
		NewFacebookGateHandler(client),
		NewViberGateHandler(client),
	)

	return &server{
		id:              id,
		receiver:        receiver,
		consume:         make(chan model.Connection, 100),
		didFinishListen: make(chan struct{}),
		stopped:         make(chan struct{}),
		client:          client,
		sessionStore:    store,
		connectionStore: NewConnectionStore(log),
		log:             log,
		gateFactory:     fabric,
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

			switch m := c.(type) {
			case model.IMBotControlGrantedEvent:
				if err := s.handleBotControlGranted(m); err != nil {
					s.log.Error("handling bot control granted",
						wlog.String("thread_id", m.ThreadID),
						wlog.Int("sub", m.Sub),
						wlog.Err(err),
					)
				}

			case model.IMEventWrapper:
				if m.GetType() == model.IMEventTypeBotControlReleased {
					s.handleBotControlReleased(m)

					continue
				}

				if m.GetPayload().GetThreadID() == "" {
					s.log.Warn("received message with empty thread ID", wlog.String("message_id", m.GetPayload().MessageID()))

					continue
				}

				if err := s.nodeMessage(m); err != nil {
					s.log.Error("handling message", wlog.String("message_id", m.GetPayload().MessageID()), wlog.Err(err))
				}
			}
		}
	}
}

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

	if err := s.sessionStore.RemoveByThread(threadID); err != nil {
		s.log.Error("bot control released: failed to clear sessions",
			wlog.String("thread_id", threadID),
			wlog.Err(err),
		)
	}

	s.log.Debug("bot control released: stopped running schema",
		wlog.String("thread_id", threadID),
		wlog.Int("connections", broken),
	)
}

// handleBotControlGranted reacts to a bot.control.granted.v1 event. When an existing
// connection already owns the thread (bot->bot resume, or the client message already
// started the schema), it just re-points that connection at the new bot via onTransfer.
// Otherwise — the human->bot case — it starts a fresh schema for the granted bot
// IMMEDIATELY, without waiting for the next inbound client message. The grant event
// carries no customer peer, so the thread participants are fetched to synthesize the
// start message (from = customer, to = bot).
func (s *server) handleBotControlGranted(m model.IMBotControlGrantedEvent) error {
	compositeSessionID := m.ThreadID + "." + strconv.Itoa(m.Sub)

	s.log.Debug("bot control granted",
		wlog.String("thread_id", m.ThreadID),
		wlog.Int("bot_sub", m.Sub),
		wlog.Int("released_sub", m.ReleasedSub),
		wlog.String("member_id", m.MemberID),
		wlog.Any("is_resume", m.IsResume),
		wlog.Any("auto_leave", m.AutoLeave),
		wlog.String("session_id", compositeSessionID),
	)

	// The granted event names the bot being released (ReleasedSub). Tear its
	// connection down here so a superseded bot never keeps running alongside the new
	// controller. The schema goroutine unwinds and removes itself from the cache.
	if m.ReleasedSub != 0 && m.ReleasedSub != m.Sub {
		releasedSessionID := m.ThreadID + "." + strconv.Itoa(m.ReleasedSub)
		if released, ok := s.connectionStore.Get(releasedSessionID); ok {
			s.log.Debug("releasing previous bot connection on grant",
				wlog.String("released_session_id", releasedSessionID),
				wlog.Int("released_sub", m.ReleasedSub),
			)
			released.Break()
		}
	}

	if conn, ok := s.connectionStore.Get(compositeSessionID); ok {
		s.log.Debug("live connection exists, resuming instead of starting a fresh schema",
			wlog.String("session_id", compositeSessionID),
		)
		conn.onTransfer(m)

		return nil
	}

	// No live connection for this session. Schemas start ONLY from a grant (never from
	// an inbound message), so both a first-time grant and a resume (pop of the control
	// stack) fall through to the fresh-start path below. Previously resume deferred to
	// the next inbound message to re-establish the schema; that is gone now that
	// nodeMessage only delivers to already-live connections.

	to := model.ImEndpoint{
		Sub:      strconv.Itoa(m.Sub),
		Issuer:   IMUserTypeBot,
		MemberID: m.MemberID,
	}

	from, err := s.resolveCustomerPeer(m, to)
	if err != nil {
		s.log.Error("resolve customer peer failed, schema will not start",
			wlog.String("thread_id", m.ThreadID),
			wlog.Int("bot_sub", m.Sub),
			wlog.Err(err),
		)

		return err
	}

	if from.Sub == "" {
		s.log.Error("no customer peer in thread, schema will not start",
			wlog.String("thread_id", m.ThreadID),
			wlog.Int("bot_sub", m.Sub),
		)

		return fmt.Errorf("bot control granted: no customer peer resolved for thread %s", m.ThreadID)
	}

	s.log.Debug("starting schema on grant",
		wlog.String("thread_id", m.ThreadID),
		wlog.Int("bot_sub", m.Sub),
		wlog.String("customer_sub", from.Sub),
		wlog.String("customer_name", from.Name),
		wlog.String("session_id", compositeSessionID),
	)

	msg := s.synthesizeGrantMessage(m, from, to)

	return s.startDialog(compositeSessionID, to, msg)
}

// resolveCustomerPeer loads the thread participants and returns the customer endpoint —
// the non-bot member that is not the granted bot itself. It reuses the same gateway
// Search RPC as Connection.treadInfo, addressed with schema metadata for the granted bot.
func (s *server) resolveCustomerPeer(m model.IMBotControlGrantedEvent, to model.ImEndpoint) (model.ImEndpoint, error) {
	schemaID, _ := strconv.Atoi(to.Sub)
	hdrs := metadata.New(map[string]string{
		"x-webitel-type":   "schema",
		"x-webitel-schema": fmt.Sprintf("%d.%d", m.DomainID, schemaID),
	})

	result, err := s.client.threadService.Api.Search(
		metadata.NewOutgoingContext(s.client.ctx, hdrs),
		&p.ThreadSearchRequest{
			Ids:  []string{m.ThreadID},
			Size: 1,
		},
	)
	if err != nil {
		return model.ImEndpoint{}, fmt.Errorf("searching thread %s: %w", m.ThreadID, err)
	}

	if len(result.GetItems()) == 0 {
		return model.ImEndpoint{}, fmt.Errorf("thread %s not found", m.ThreadID)
	}

	members := result.GetItems()[0].GetMembers()

	peer := selectCustomerPeer(members, to.Sub)
	s.log.Debug("resolved customer peer from thread members",
		wlog.String("thread_id", m.ThreadID),
		wlog.String("bot_sub", to.Sub),
		wlog.String("customer_sub", peer.Sub),
		wlog.Any("found", peer.Sub != ""),
		wlog.Int("members_count", len(members)),
	)

	return peer, nil
}

// selectCustomerPeer returns the first human (non-bot) participant that is not the
// granted bot itself, mapped to an ImEndpoint. Returns a zero endpoint (Sub == "")
// when the thread carries no resolvable customer.
func selectCustomerPeer(members []*p.ThreadMember, botSub string) model.ImEndpoint {
	for _, member := range members {
		contact := member.GetContact()
		if contact == nil {
			continue
		}

		// Skip bots (both the granted one and any other automatic participant).
		if contact.GetIsBot() || contact.GetSub() == botSub {
			continue
		}

		return model.ImEndpoint{
			Sub:      contact.GetSub(),
			Issuer:   contact.GetIss(),
			Name:     contact.GetName(),
			MemberID: member.GetId(),
			Role:     int(member.GetRole()),
		}
	}

	return model.ImEndpoint{}
}

// synthesizeGrantMessage builds a MessageWrapper that mimics an inbound customer
// message so newConnection/setupVariables can start the schema. It satisfies the
// IMEvent contract (Sender/Receivers/GetThreadID/GetDomainID/Message) with the
// resolved customer peer as sender and the granted bot as the receiver.
func (s *server) synthesizeGrantMessage(m model.IMBotControlGrantedEvent, from, to model.ImEndpoint) model.IMEventWrapper {
	return model.MessageWrapper[model.Message]{
		DomainID: int64(m.DomainID),
		Type:     model.IMEventTypeMessage,
		Message: model.Message{
			ThreadID: m.ThreadID,
			DomainID: m.DomainID,
			From:     from,
			To:       []model.ImEndpoint{to},
		},
	}
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

		// No live connection for this bot receiver: start a fresh schema straight from the
		// inbound message. This covers a plain thread start (no transfer/grant) where the
		// first customer message is what kicks the bot off. startDialog is idempotent and
		// claims the session, so it will not double-start or run on another node's session.
		if err := s.startDialog(compositeSessionID, endpoint, msg); err != nil {
			return err
		}
	}

	return nil
}

// startDialog claims the session for this node and, if the claim succeeds and no
// dialog is already running for the composite id, builds a new connection and hands
// it to the schema runner via s.consume. It is shared by the inbound-message path
// (nodeMessage) and the bot-control-granted path (handleBotControlGranted) so a fresh
// schema starts identically regardless of what triggered it.
func (s *server) startDialog(compositeSessionID string, to model.ImEndpoint, msg model.IMEventWrapper) error {
	if _, ok := s.connectionStore.Get(compositeSessionID); ok {
		// A dialog for this thread+bot already exists: do not double-start.
		return nil
	}

	seq, err := s.sessionStore.Touch(compositeSessionID, s.id)
	if err != nil {
		return err
	}

	if seq == nil {
		s.log.Warn("session owned by another node, skipping dialog start",
			wlog.String("id", compositeSessionID),
			wlog.String("thread_id", msg.GetPayload().GetThreadID()),
			wlog.String("message_id", msg.GetPayload().MessageID()),
		)

		return nil
	}

	if *seq > 1 {
		s.log.Warn("received message with sequance thread ID", wlog.Int("sequance", *seq))
	}

	dialog := newConnection(s, compositeSessionID, to, msg)
	dialog.setupVariables()

	s.connectionStore.Add(dialog)
	dialog.log.Debug("dispatched dialog to schema runner",
		wlog.String("session_id", compositeSessionID),
		wlog.String("bot_sub", to.Sub),
	)
	s.consume <- dialog

	return nil
}
