package mq

import (
	"github.com/webitel/flow_manager/model"
)

type QueueEvent any

type MQ interface {
	SendJSON(exchange, key string, data []byte) *model.AppError
	Close()

	ConsumeCallEvent() <-chan model.CallActionData
	ConsumeExec() <-chan model.ChannelExec
	ConsumeIM() <-chan model.IMEventWrapper
	ConsumeCCEvents() <-chan model.CCQueueEvent

	QueueEvent() QueueEvent
}
