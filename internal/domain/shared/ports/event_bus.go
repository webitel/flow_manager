package ports

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

// EventBus is the outbound port for publishing and consuming broker events.
type EventBus interface {
	Publish(ctx context.Context, exchange, key string, data []byte) error
	Close()

	ConsumeCallEvent() <-chan model.CallActionData
	ConsumeExec() <-chan model.ChannelExec
	ConsumeIM() <-chan model.MessageWrapper
	ConsumeCCEvents() <-chan model.CCQueueEvent
}
