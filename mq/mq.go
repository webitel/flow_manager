package mq

import (
	"github.com/webitel/flow_manager/model"
)

type MQ interface {
	SendJSON(exchange, key string, data []byte) *model.AppError
	Close()

	ConsumeCallEvent() <-chan model.CallActionData
	ConsumeCallMediaStatsEvent() <-chan model.CallMediaStats
	ConsumeExec() <-chan model.ChannelExec
	ConsumeIM() <-chan model.MessageWrapper

	QueueEvent() QueueEvent
}

type QueueEvent any
