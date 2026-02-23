package mq

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type LayeredMQLayer interface {
	MQ
}

type LayeredMQ struct {
	context context.Context
	MQLayer LayeredMQLayer
}

func NewMQ(mq LayeredMQLayer) MQ {
	return &LayeredMQ{
		context: context.TODO(),
		MQLayer: mq,
	}
}

func (l *LayeredMQ) SendJSON(exchange, key string, data []byte) *model.AppError {
	return l.MQLayer.SendJSON(exchange, key, data)
}

func (l *LayeredMQ) Close() {
	l.MQLayer.Close()
}

func (l *LayeredMQ) ConsumeCallEvent() <-chan model.CallActionData {
	return l.MQLayer.ConsumeCallEvent()
}

func (l *LayeredMQ) ConsumeExec() <-chan model.ChannelExec {
	return l.MQLayer.ConsumeExec()
}

func (l *LayeredMQ) ConsumeIM() <-chan model.MessageWrapper {
	return l.MQLayer.ConsumeIM()
}

func (l *LayeredMQ) QueueEvent() QueueEvent {
	return l.MQLayer.QueueEvent()
}

func (l *LayeredMQ) ConsumeCallMediaStatsEvent() <-chan model.CallMediaStats {
	return l.MQLayer.ConsumeCallMediaStatsEvent()
}
