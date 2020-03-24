package queue

import (
	"github.com/webitel/flow_manager/app"
	"github.com/webitel/flow_manager/model"
)

type Queue interface {
	JoinCall(call model.Call)
}

type queue struct {
	fm app.FlowManager
}

func NewQueue(f app.FlowManager) Queue {
	return &queue{
		fm: f,
	}
}

func (q *queue) JoinCall(call model.Call) {
	q.fm.JoinToInboundQueue()
}
