package rabbit

import (
	"github.com/webitel/flow_manager/mq"
)

type RQueueEventMQ struct {
	amqp mq.MQ
}

func NewQueueMQ(amqp mq.MQ) mq.QueueEvent {
	return &RQueueEventMQ{amqp}
}
