package emailevent

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/gen/cc"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/store"
)

const (
	pollInterval     = time.Second
	leaseDuration    = time.Minute
	handshakeTimeout = 10 * time.Second
	wakeUpTimeout    = 5 * time.Second
	recoverInterval  = 5 * time.Second
)

// AttemptWaker wakes the active Email attempt of a Thread in call_center.
type AttemptWaker interface {
	EmailWakeUp(ctx context.Context, in *cc.EmailWakeUpRequest) (*cc.EmailWakeUpResponse, error)
}

// Server consumes email-service events into the Inbox and hands them to the
// email router as connections; workers start with the first Consume call.
type Server struct {
	nodeId   string
	runId    string
	cfg      model.EmailEventsSettings
	store    store.EmailEventStore
	waker    AttemptWaker
	joiner   AttemptJoiner
	consumer *consumer
	log      *wlog.Logger
	conns    chan model.Connection

	ctx    context.Context //nolint:containedctx
	cancel context.CancelFunc

	mu          sync.Mutex
	stopped     bool
	workersOnce sync.Once
	closeOnce   sync.Once
	wg          sync.WaitGroup
}

var _ model.Server = (*Server)(nil)

func NewServer(nodeId string, cfg model.EmailEventsSettings, amqpUrl string, st store.EmailEventStore, waker AttemptWaker, joiner AttemptJoiner, log *wlog.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("email events: %w", err)
	}

	// Tells workers and consumers of one process apart from its restarts.
	runId := model.NewId()
	log = log.With(wlog.String("node_id", nodeId))
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		nodeId:   nodeId,
		runId:    runId,
		cfg:      cfg,
		store:    st,
		waker:    waker,
		joiner:   joiner,
		consumer: newConsumer(amqpUrl, nodeId, runId, cfg, st, log),
		log:      log.With(wlog.String("component", "email_events")),
		conns:    make(chan model.Connection),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (s *Server) Name() string {
	return "EmailEvents"
}

func (s *Server) Host() string {
	return ""
}

func (s *Server) Port() int {
	return 0
}

func (s *Server) Type() model.ConnectionType {
	return model.ConnectionTypeEmail
}

func (s *Server) Cluster(discovery.ServiceDiscovery) *model.AppError {
	return nil
}

// Start runs the consumer, the Inbox upkeep and the route watcher; it runs before the
// email router exists, so workers wait for Consume.
func (s *Server) Start() *model.AppError {
	if !s.spawn(s.consumer.Run, s.recoverLoop, s.cleanupLoop, newRouteWatcher(s).run) {
		return model.NewAppError("EmailEvents.Start", "email_events.start.stopped", nil, "server is stopped", http.StatusInternalServerError)
	}

	return nil
}

// Consume starts the workers once; the caller is the email listener.
func (s *Server) Consume() <-chan model.Connection {
	s.workersOnce.Do(func() {
		runs := make([]func(context.Context), 0, s.cfg.Workers)
		for i := range s.cfg.Workers {
			runs = append(runs, newWorker(s, i).run)
		}
		s.spawn(runs...)
	})

	return s.conns
}

// Stop cancels the running Flows too: their connections derive from the server.
func (s *Server) Stop() {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()

	s.cancel()
	s.wg.Wait()
	// Workers, the only senders, have returned.
	s.closeOnce.Do(func() { close(s.conns) })
}

// spawn starts goroutines unless the server is stopped.
func (s *Server) spawn(runs ...func(context.Context)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return false
	}

	s.wg.Add(len(runs))
	for _, run := range runs {
		go func() {
			defer s.wg.Done()
			run(s.ctx)
		}()
	}

	return true
}

// recoverLoop returns rows of workers that stopped holding them to pending.
func (s *Server) recoverLoop(ctx context.Context) {
	ticker := time.NewTicker(recoverInterval)
	defer ticker.Stop()

	for {
		recovered, appErr := s.store.RecoverExpired(ctx)
		switch {
		case appErr != nil && ctx.Err() == nil:
			s.log.Error("email event lease recovery failed", wlog.Err(appErr))
		case recovered > 0:
			s.log.Warn("expired email event leases recovered", wlog.Int64("events", recovered))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		s.cleanup(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// cleanup deletes in batches, so one statement never holds many row locks.
func (s *Server) cleanup(ctx context.Context) {
	before := time.Now().Add(-s.cfg.Retention)

	var total int64
	for ctx.Err() == nil {
		deleted, appErr := s.store.DeleteAcceptedBefore(ctx, before, s.cfg.CleanupBatch)
		if appErr != nil {
			if ctx.Err() == nil {
				s.log.Error("email event cleanup failed", wlog.Err(appErr))
			}

			break
		}

		total += deleted
		if deleted < int64(s.cfg.CleanupBatch) {
			break
		}
	}

	if total > 0 {
		s.log.Info("accepted email events removed", wlog.Int64("deleted", total))
	}
}
