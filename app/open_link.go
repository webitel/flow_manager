package app

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"strconv"
)

const (
	engineExchange = "engine"
)

func (fm *FlowManager) OpenLink(n model.Notification) {
	err := fm.eventQueue.SendJSON(engineExchange, "notification."+strconv.Itoa(int(n.DomainId)), n.ToJson())
	if err != nil {
		wlog.Error(err.Error())
	}
}
