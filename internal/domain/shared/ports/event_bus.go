package ports

import (
	"context"

	"github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/flow"
)

// EventBus is the outbound port for publishing and consuming broker events.
type EventBus interface {
	Publish(ctx context.Context, exchange, key string, data []byte) error
	Close()
	Start() error

	ConsumeCallEvent() <-chan call.CallActionData
	ConsumeExec() <-chan flow.ChannelExec
	ConsumeIM() <-chan chat.MessageWrapper
	ConsumeCCEvents() <-chan chat.CCQueueEvent
}
