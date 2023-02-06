package app

import (
	"net/http"

	"github.com/webitel/flow_manager/model"
)

var ErrAllowUseMQ = model.NewAppError("App", "app.settings.mq.allow_use.disabled", nil, "Allow push message to MQ is disabled", http.StatusForbidden)

func (f *FlowManager) SendMQJson(exchange, key string, body []byte) *model.AppError {
	if !f.Config().AllowUseMQ {
		return ErrAllowUseMQ
	}
	return f.eventQueue.SendJSON(exchange, key, body)
}
