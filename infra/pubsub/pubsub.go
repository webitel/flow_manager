package pubsub

import (
	"context"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/webitel/wlog"
)

type OnConnetFn func(conn *amqp.Connection, pubCh *Channel) error

type Manager struct {
	address string

	log *wlog.Logger

	conn    *amqp.Connection
	channel *Channel

	mu             sync.Mutex
	connected      bool
	close          chan bool
	waitConnection chan struct{}

	runningFetches  sync.WaitGroup
	runningHandlers sync.WaitGroup

	onConnect []OnConnetFn
}

func New(log *wlog.Logger, address string, hooks ...OnConnetFn) (*Manager, error) {
	m := &Manager{
		address:         address,
		log:             log,
		close:           make(chan bool),
		waitConnection:  make(chan struct{}),
		runningFetches:  sync.WaitGroup{},
		runningHandlers: sync.WaitGroup{},
		onConnect:       hooks,
	}

	if err := m.tryConnect(); err != nil {
		return nil, err
	}

	// Its bad case of nil == waitConnection, so close it at start.
	close(m.waitConnection)

	return m, nil
}

func (m *Manager) Start() error {
	m.mu.Lock()

	if m.connected {
		m.mu.Unlock()
		return nil
	}

	// Check it was closed.
	select {
	case <-m.close:
		m.close = make(chan bool)
	default:
		// Noop, new conn.
	}

	m.mu.Unlock()

	return m.connect()
}

// Shutdown stops the manager from fetching new messages and processing them.
func (m *Manager) Shutdown() error {
	// Once it's time to force-close tasks, cancel the base context.
	go func() {
		m.closeConn()
	}()

	m.runningFetches.Wait()

	// Wait for running handlers to finish.
	m.runningHandlers.Wait()

	// Finally, close all connections to the PubSub providers.
	m.closeConn()

	m.log.Debug("shutdown pubsub")

	return nil
}

func (m *Manager) Channel() *Channel {
	return m.channel
}

func (m *Manager) AddOnConnect(fn OnConnetFn) {
	m.onConnect = append(m.onConnect, fn)
}

// Publish sends a message to the broker. It blocks until the connection is
// ready, then delegates to the publishing channel.
func (m *Manager) Publish(ctx context.Context, exchange, key string, body []byte) error {
	select {
	case <-m.waitConnection:
	case <-ctx.Done():
		return ctx.Err()
	}
	return m.channel.Publish(ctx, exchange, key, body)
}

func (m *Manager) connect() error {
	// Try connect.
	if err := m.tryConnect(); err != nil {
		return err
	}

	m.mu.Lock()
	m.connected = true
	m.mu.Unlock()

	// Create reconnect loop.
	go m.reconnect()

	return nil
}

func (m *Manager) reconnect() {
	// Skip first connect.
	var connect bool

	for {
		if connect {
			if err := m.tryConnect(); err != nil {
				time.Sleep(1 * time.Second)

				continue
			}

			m.mu.Lock()
			m.connected = true
			m.mu.Unlock()

			// Unblock resubscribe a cycle - close Channel.
			// At this point Channel is created and unclosed - close it without any additional checks.
			close(m.waitConnection)
		}

		connect = true
		notifyClose := make(chan *amqp.Error)
		m.conn.NotifyClose(notifyClose)

		chanNotifyClose := make(chan *amqp.Error)
		m.Channel().channel.NotifyClose(chanNotifyClose)

		// To avoid deadlocks it is necessary to consume the messages from all channels.
		for notifyClose != nil || chanNotifyClose != nil {
			select {
			case err := <-chanNotifyClose:
				m.log.Error("closed rabbitmq/pubsub connection.. attempting to reconnect", wlog.Err(err))

				// Block all resubscribe attempt - they are useless because there is no connection to rabbitmq.
				// Create Channel 'waitConnection' (at this point Channel is nil or closed,
				// create it without unnecessary checks).
				m.mu.Lock()
				m.connected = false
				m.waitConnection = make(chan struct{})
				m.mu.Unlock()
				chanNotifyClose = nil

			case err := <-notifyClose:
				m.log.Error("closed rabbitmq/pubsub channel.. attempting to reconnect", wlog.Err(err))

				// Block all resubscribe attempt - they are useless because there is no connection to rabbitmq.
				// Create Channel 'waitConnection' (at this point Channel is nil or closed,
				// create it without unnecessary checks).
				m.mu.Lock()
				m.connected = false
				m.waitConnection = make(chan struct{})
				m.mu.Unlock()
				notifyClose = nil

			case <-m.close:
				return
			}
		}
	}
}

func (m *Manager) tryConnect() error {
	var err error

	m.conn, err = amqp.Dial(m.address)
	if err != nil {
		return err
	}

	m.channel, err = newChannel(m.conn, 0, false, true)
	if err != nil {
		return err
	}

	for _, f := range m.onConnect {
		if err = f(m.conn, m.channel); err != nil {
			m.log.Error(err.Error())
			// TODO
		}
	}

	return nil
}

func (m *Manager) closeConn() {
	m.mu.Lock()
	defer m.mu.Unlock()

	select {
	case <-m.close:
		return
	default:
		close(m.close)
		m.connected = false
	}
}
