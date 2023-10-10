package flow

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type GetQueueMetrics struct {
	Bucket      *model.SearchEntity `json:"bucket"`
	Queue       *model.SearchEntity `json:"queue"`
	Metric      string              `json:"metric"`
	LastMinutes int                 `json:"lastMinutes"`
	Set         string
	Field       string `json:"field"` // ?????
	Calls       string `json:"calls"` // ?????
	SlSec       int    `json:"slSec"`
}

type GetQueueInfo struct {
	Queue *model.SearchEntity `json:"queue"`
	Set   model.Variables
}

type GetQueueAgents struct {
	Channel string              `json:"channel"`
	Queue   *model.SearchEntity `json:"queue"`
	Set     model.Variables     `json:"set"`
}

func (r *router) getQueueInfo(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetQueueInfo
	var err *model.AppError
	var res model.Variables

	err = scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Queue == nil {
		return model.CallResponseError, ErrorRequiredParameter("getQueueInfo", "queue")
	}
	if len(argv.Set) == 0 {
		return model.CallResponseError, ErrorRequiredParameter("getQueueInfo", "set")
	}

	res, err = r.fm.Store.Queue().GetQueueData(c.DomainId(), argv.Queue, argv.Set)
	if err != nil {
		return nil, err
	}

	return c.Set(ctx, res)

}

func (r *router) getQueueMetrics(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetQueueMetrics
	var res float64

	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Queue == nil {
		return model.CallResponseError, ErrorRequiredParameter("getQueueMetrics", "queue")
	}
	if argv.Set == "" {
		return model.CallResponseError, ErrorRequiredParameter("getQueueMetrics", "set")
	}

	switch argv.Calls {
	case "complete":
		if argv.Metric == "count" {

		} else {
			req := &model.SearchQueueCompleteStatistics{
				QueueId:     argv.Queue.Id,
				QueueName:   argv.Queue.Name,
				BucketId:    nil,
				BucketName:  nil,
				LastMinutes: argv.LastMinutes,
				Metric:      argv.Metric,
				Field:       argv.Field,
				SlSec:       argv.SlSec,
			}

			if argv.Bucket != nil {
				req.BucketId = argv.Bucket.Id
				req.BucketName = argv.Bucket.Name
			}

			if res, err = r.fm.Store.Queue().HistoryStatistics(c.DomainId(), req); err != nil {
				return nil, err
			}
		}
	case "":
	}

	//req := &model.SearchQueue{
	//	Id:          argv.Queue.Id,
	//	Name:        argv.Queue.Name,
	//	LastMinutes: argv.LastMinutes,
	//	Result:      nil,
	//	BucketId:    argv.BucketId,
	//}
	//
	//res, err := r.fm.Store.Queue().Statistics(c.DomainId(), req, argv.Metric)
	//
	//if err != nil {
	//	return nil, err
	//}

	return c.Set(ctx, model.Variables{
		argv.Set: res,
	})
}

func (r *router) getQueueAgents(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = GetQueueAgents{
		Channel: scope.ChannelType(),
	}
	var res model.Variables

	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Queue == nil || argv.Queue.Id == nil {
		return model.CallResponseError, ErrorRequiredParameter("getQueueAgent", "queue")
	}

	res, err = r.fm.Store.Queue().GetQueueAgents(c.DomainId(), *argv.Queue.Id, argv.Channel, argv.Set)
	if err != nil {
		return model.CallResponseError, err
	}

	return c.Set(ctx, res)
}
