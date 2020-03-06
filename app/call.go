package app

import (
	"fmt"
	"github.com/webitel/engine/discovery"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"sync"
	"time"
)

type callWatcher struct {
	fm                 *FlowManager
	startOnce          sync.Once
	callTasks          Pool
	callHistoryWatcher *discovery.Watcher
}

func NewCallWatcher(fm *FlowManager) *callWatcher {
	return &callWatcher{
		fm: fm,
	}
}

func (c *callWatcher) Start() {
	c.startOnce.Do(func() {
		go func() {
			c.callHistoryWatcher = discovery.MakeWatcher("call-history", 1000, c.storeHangupCalls)
			c.callHistoryWatcher.Start()
		}()
	})
}

func (c *callWatcher) Stop() {
	if c.callHistoryWatcher != nil {
		c.callHistoryWatcher.Stop()
	}
}

func (f *FlowManager) listenCallEvents(stop chan struct{}) {
	wlog.Info(fmt.Sprintf("listen call events..."))
	defer wlog.Debug(fmt.Sprintf("stop listening call events..."))
	for {
		select {
		case <-stop:
			return
		case c, ok := <-f.eventQueue.ConsumeCallEvent():
			if !ok {
				return
			}

			if c.DomainId == 0 {
				wlog.Error(fmt.Sprintf("call %s not found domain: %v", c.Id, c))
				continue
			}

			//TODO POOL
			go f.handleCallAction(c)
		}
	}
}

func (f *FlowManager) handleCallAction(data model.CallActionData) {
	action := data.GetEvent()

	switch action.(type) {
	case *model.CallActionRinging:
		if err := f.Store.Call().Save(action.(*model.CallActionRinging)); err != nil {
			wlog.Error(err.Error())
		}
	case *model.CallActionHangup:
		if err := f.Store.Call().SetHangup(action.(*model.CallActionHangup)); err != nil {
			wlog.Error(err.Error())
		}

	default:
		if err := f.Store.Call().SetState(&data.CallAction); err != nil {
			wlog.Error(err.Error())
		}
	}
}

func (c *callWatcher) storeHangupCalls() {
	if err := c.fm.Store.Call().MoveToHistory(); err != nil {
		wlog.Error(err.Error())
		time.Sleep(time.Second * 5)
	}
}
