package app

import (
	"fmt"
	"sync"
	"time"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

const (
	refreshMissedNotification = "refresh_missed"
)

/*
root@dev:/usr/local/bin# journalctl -t flow_manager | grep 2e5cb2b7-4bef-4ec2-907c-ddfde1b6a3e3
Nov 24 16:57:53 dev flow_manager[13448]: 2020-11-24T16:57:53.758+0200        debug        rabbit/client.go:150        call 2e5cb2b7-4bef-4ec2-907c-ddfde1b6a3e3 [hangup]
Nov 24 16:57:54 dev flow_manager[13448]: 2020-11-24T16:57:54.185+0200        debug        rabbit/client.go:150        call 2e5cb2b7-4bef-4ec2-907c-ddfde1b6a3e3 [ringing]
*/
type callWatcher struct {
	fm                 *FlowManager
	startOnce          sync.Once
	callTasks          Pool
	callHistoryWatcher *discovery.Watcher
}

func NewCallWatcher(fm *FlowManager) *callWatcher {
	return &callWatcher{
		fm: fm,
		//callTasks: NewPool(5, 10),
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
	f.log.Info(fmt.Sprintf("listen call events..."))
	defer f.log.Debug(fmt.Sprintf("stop listening call events..."))
	for {
		select {
		case <-stop:
			return
		case c, ok := <-f.eventQueue.ConsumeCallEvent():
			if !ok {
				return
			}

			if c.DomainId == 0 {
				f.log.Error("bad domain", wlog.Namespace("call"),
					wlog.Int64("domain_id", c.DomainId),
					wlog.String("call_id", c.Id),
					wlog.String("event_name", c.Event),
				)
				continue
			}

			//TODO POOL
			go f.handleCallAction(c)
		}
	}
}

func (f *FlowManager) handleCallAction(data model.CallActionData) {
	action := data.GetEvent()

	log := f.Log().With(
		wlog.Namespace("context"),
		wlog.String("call_id", data.Id),
		wlog.String("scope", "call"),
		wlog.String("event_name", data.Event),
	)

	switch call := action.(type) {
	case *model.CallActionRinging:
		if err := f.Store.Call().Save(call); err != nil {
			log.Error(err.Error())
		}
	case *model.CallActionBridge:
		if err := f.Store.Call().SetBridged(call); err != nil {
			log.Error(err.Error())
		}

	case *model.CallActionTranscript:
		if err := f.Store.Call().SaveTranscript(call); err != nil {
			log.Error(err.Error())
		}

	case *model.CallActionHeartbeat:
		if err := f.Store.Call().SetHeartbeat(call.Id); err != nil {
			log.Error(err.Error())
		}
	case *model.CallActionHangup:
		if call.CDR != nil && !*call.CDR {
			if err := f.Store.Call().Delete(call.Id); err != nil {
				log.Error(err.Error())
			}
		} else {
			if err := f.Store.Call().SetHangup(call); err != nil {
				log.Error(err.Error())
			}
		}

	default:
		if data.Event == "eavesdrop" || data.Event == "dtmf" {
			// skip that events
			return
		}
		if err := f.Store.Call().SetState(&data.CallAction); err != nil {
			log.Error(err.Error())
		}
	}
}

func (c *callWatcher) storeHangupCalls() {
	if missed, err := c.fm.Store.Call().MoveToHistory(); err != nil {
		wlog.Error(err.Error())
		time.Sleep(time.Second * 5)
	} else if len(missed) != 0 {
		for _, v := range missed {
			c.fm.NotificationMissedCalls(v)
		}
	}
}

func (c *FlowManager) UpdateCallFrom(id string, name, number *string) *model.AppError {
	return c.Store.Call().UpdateFrom(id, name, number)
}

func (c *FlowManager) LastBridgedCall(domainId int64, number, hours string, dialer, inbound, outbound *string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	return c.Store.Call().LastBridged(domainId, number, hours, dialer, inbound, outbound, queueIds, mapRes)
}

func (c *FlowManager) SetCallUserId(domainId int64, id string, userId int64) *model.AppError {
	return c.Store.Call().SetUserId(domainId, id, userId)
}

func (f *FlowManager) SetBlindTransferNumber(domainId int64, callId string, destination string) *model.AppError {
	return f.Store.Call().SetBlindTransfer(domainId, callId, destination)
}

func (f *FlowManager) CallSetContactId(domainId int64, callId string, contactId int64) *model.AppError {
	return f.Store.Call().SetContactId(domainId, callId, contactId)
}

func (f *FlowManager) StoreCallVariables(id string, vars map[string]string) *model.AppError {
	if len(vars) == 0 {
		return nil
	}

	cv := make(model.CallVariables)
	for k, v := range vars {
		cv[k] = v
	}

	return f.Store.Call().SetVariables(id, &cv)
}
