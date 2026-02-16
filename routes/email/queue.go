package email

import "github.com/webitel/flow_manager/model"

type QueueParameters struct {
	QueueId   int
	QueueName string
}

func (r *Router) joinQueue(email model.EmailConnection, args interface{}) (model.Response, *model.AppError) {
	return nil, nil
}
