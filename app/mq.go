package app

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

var ErrAllowUseMQ = model.NewAppError("App", "app.settings.mq.allow_use.disabled", nil, "Allow push message to MQ is disabled", http.StatusForbidden)

func (f *FlowManager) SendMQJson(exchange, key string, body []byte) *model.AppError {
	if !f.Config().AllowUseMQ {
		return ErrAllowUseMQ
	}
	if err := f.eventQueue.Publish(context.Background(), exchange, key, body); err != nil {
		return model.NewAppError("MQ", "mq.publish.err", nil, err.Error(), http.StatusInternalServerError)
	}
	return nil
}
