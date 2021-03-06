package mq

import (
	"github.com/webitel/flow_manager/model"
)

type MQ interface {
	SendJSON(name string, data []byte) *model.AppError
	Close()

	ConsumeCallEvent() <-chan model.CallActionData

	QueueEvent() QueueEvent
}

type QueueEvent interface {
}
