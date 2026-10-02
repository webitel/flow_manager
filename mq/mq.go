package mq

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type QueueEvent any

type MQ interface {
	SendJSON(exchange, key string, data []byte) *model.AppError
	Close()
	Ping(ctx context.Context) error

	ConsumeCallEvent() <-chan model.CallActionData
	ConsumeExec() <-chan model.ChannelExec
	ConsumeIM() <-chan any
	ConsumeCCEvents() <-chan model.CCQueueEvent

	QueueEvent() QueueEvent
}
