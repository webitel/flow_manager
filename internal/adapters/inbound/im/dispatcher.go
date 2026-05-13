package im

import (
	"fmt"

	"github.com/webitel/wlog"

	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/flow"
)

// Dispatcher routes inbound MQ messages to the correct Connection.
// For an existing in-memory connection it delivers directly via OnMessage.
// For unknown connections it claims ownership via SessionStore and starts a
// new flow by pushing a new Connection to the consume channel.
type Dispatcher struct {
	appID        string
	connStore    *ConnectionStore
	sessionStore SessionStore
	consume      chan<- flow.Connection
	log          *wlog.Logger
	srv          *server
}

func newDispatcher(appID string, store *ConnectionStore, ss SessionStore, consume chan<- flow.Connection, log *wlog.Logger, srv *server) *Dispatcher {
	return &Dispatcher{
		appID:        appID,
		connStore:    store,
		sessionStore: ss,
		consume:      consume,
		log:          log,
		srv:          srv,
	}
}

// Startup cleans up any stale ownership records left by a previous process
// crash. Must be called once before Handle is used.
func (d *Dispatcher) Startup() error {
	return d.sessionStore.RemoveAll(d.appID)
}

// Shutdown releases all owned connections from the session store.
func (d *Dispatcher) Shutdown() {
	if err := d.sessionStore.RemoveAll(d.appID); err != nil {
		d.log.Error(fmt.Sprintf("im dispatcher: RemoveAll on shutdown failed: %v", err))
	}
}

// Unregister removes a connection from the in-memory store and releases the
// ownership claim in the session store.
func (d *Dispatcher) Unregister(c *Connection) {
	d.connStore.Delete(c)
	if err := d.sessionStore.Remove(c.id, d.appID); err != nil {
		d.log.Warn(fmt.Sprintf("im dispatcher: failed to remove session for id=%s: %v", c.id, err))
	}
}

// Handle processes one inbound message. It is safe for concurrent use.
func (d *Dispatcher) Handle(msg chatdomain.MessageWrapper) error {
	if msg.Message.From.Issuer == "bot" {
		return nil
	}

	for _, endpoint := range msg.Message.To {
		if endpoint.Issuer != "bot" {
			continue
		}

		id := fmt.Sprintf("%s.%s", msg.Message.ThreadID, endpoint.Sub)

		// Fast path: connection already live in this process.
		if conn, ok := d.connStore.Get(id); ok {
			conn.OnMessage(msg)
			continue
		}

		// Slow path: attempt to claim ownership for a new connection.
		seq, err := d.sessionStore.Touch(id, d.appID)
		if err != nil {
			return err
		}
		if seq == nil {
			// Another node already owns this connection.
			continue
		}

		if *seq > 1 {
			d.log.Warn(fmt.Sprintf("im dispatcher: unexpected seq=%d for id=%s", *seq, id))
		}

		conn := newConnection(d.srv, id, endpoint, msg)
		d.connStore.Add(conn)
		conn.log.Debug("start dialog " + id)
		d.consume <- conn
	}

	return nil
}
