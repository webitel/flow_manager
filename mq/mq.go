package mq

import (
	"github.com/webitel/flow_manager/model"
)

type MQ interface {
	SendJSON(exchange string, key string, data []byte) *model.AppError
	Close()

	ConsumeCallEvent() <-chan model.CallActionData
	ConsumeExec() <-chan model.ChannelExec

	QueueEvent() QueueEvent
}

type QueueEvent interface {
}
