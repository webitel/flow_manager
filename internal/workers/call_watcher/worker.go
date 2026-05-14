// Package call_watcher handles call events from the event bus and periodically
// moves completed calls to history in the store.
package call_watcher

import (
	"sync"
	"time"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/infrastructure/watcher"
	"github.com/webitel/flow_manager/internal/storage"
)

const (
	// RefreshMissedNotification is the event name that triggers a missed-call
	// notification refresh.
	RefreshMissedNotification = "refresh_missed"
)

// CallEventDeps is the narrow interface required by the call-event listener.
type CallEventDeps interface {
	// ConsumeCallEvent returns a channel of incoming call-action events.
	ConsumeCallEvent() <-chan call.CallActionData
	// NotificationMissedCalls sends a missed-call notification.
	NotificationMissedCalls(c call.MissedCall)
}

// Worker owns both the periodic history-flush watcher and the call-event
// listener goroutine.
type Worker struct {
	store              storage.Store
	deps               CallEventDeps
	log                *wlog.Logger
	startOnce          sync.Once
	callHistoryWatcher *watcher.Watcher
}

// New creates a Worker.
func New(st storage.Store, deps CallEventDeps, log *wlog.Logger) *Worker {
	return &Worker{
		store: st,
		deps:  deps,
		log:   log,
	}
}

func (c *Worker) Start(stop chan struct{}) {
	c.startOnce.Do(func() {
		go func() {
			c.callHistoryWatcher = watcher.MakeWatcher("call-history", 1000, c.storeHangupCalls)
			c.callHistoryWatcher.Start()
		}()

		go c.listenCallEvents(stop)
	})
}

func (c *Worker) Stop() {
	if c.callHistoryWatcher != nil {
		c.callHistoryWatcher.Stop()
	}
}

func (c *Worker) listenCallEvents(stop chan struct{}) {
	c.log.Info("listen call events...")
	defer c.log.Debug("stop listening call events...")
	for {
		select {
		case <-stop:
			return

		case ev, ok := <-c.deps.ConsumeCallEvent():
			if !ok {
				return
			}

			if ev.DomainId == 0 && ev.CallAction.Event != call.CallActionStatsName {
				c.log.Error("bad domain", wlog.Namespace("call"),
					wlog.Int64("domain_id", ev.DomainId),
					wlog.String("call_id", ev.Id),
					wlog.String("event_name", ev.Event),
				)
				continue
			}

			go c.handleCallAction(ev)
		}
	}
}

func (c *Worker) handleCallAction(data call.CallActionData) {
	action := data.GetEvent()

	log := c.log.With(
		wlog.Namespace("context"),
		wlog.String("call_id", data.Id),
		wlog.String("scope", "call"),
		wlog.String("event_name", data.Event),
	)

	switch call := action.(type) {
	case *call.CallActionRinging:
		if err := c.store.Call().Save(call); err != nil {
			log.Error(err.Error())
		}
	case *call.CallActionBridge:
		if err := c.store.Call().SetBridged(call); err != nil {
			log.Error(err.Error())
		}

	case *call.CallActionTranscript:
		if err := c.store.Call().SaveTranscript(call); err != nil {
			log.Error(err.Error())
		}

	case *call.CallActionHeartbeat:
		if err := c.store.Call().SetHeartbeat(call.Id); err != nil {
			log.Error(err.Error())
		}
	case *call.CallActionHangup:
		if call.CDR != nil && !*call.CDR {
			if err := c.store.Call().Delete(call.Id); err != nil {
				log.Error(err.Error())
			}
		} else {
			if err := c.store.Call().SetHangup(call); err != nil {
				log.Error(err.Error())
			}
		}
	case *call.CallActionMediaStats:
		err := c.store.Call().SaveMediaStats(call)
		if err != nil {
			log.Error(err.Error())
		}

	default:
		if data.Event == "eavesdrop" || data.Event == "dtmf" || data.Event == "update" || data.Event == "transcript" {
			return
		}
		if err := c.store.Call().SetState(&data.CallAction); err != nil {
			log.Error(err.Error())
		}
	}
}

func (c *Worker) storeHangupCalls() {
	if missed, err := c.store.Call().MoveToHistory(); err != nil {
		wlog.Error(err.Error())
		time.Sleep(time.Second * 5)
	} else if len(missed) != 0 {
		for _, v := range missed {
			c.deps.NotificationMissedCalls(v)
		}
	}
}
