package flow

import (
	"context"
	"fmt"
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
	Set   string
	Field string `json:"field"`
}

func (r *router) getQueueInfo(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetQueueInfo
	var res *model.QueueData

	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Queue == nil {
		return model.CallResponseError, ErrorRequiredParameter("getQueueInfo", "queue")
	}
	if argv.Set == "" {
		return model.CallResponseError, ErrorRequiredParameter("getQueueInfo", "set")
	}
	if argv.Field == "" {
		return model.CallResponseError, ErrorRequiredParameter("getQueueInfo", "field")
	}

	res, err = r.fm.Store.Queue().GetQueueData(c.DomainId(), argv.Queue)
	if err != nil {
		return nil, err
	}

	fmt.Println(res)

	var set = ""

	switch argv.Field {
	case "enabled":
		set = fmt.Sprintf("%v", res.Enabled)
	case "type":
		set = fmt.Sprintf("%v", res.Type)
	case "priority":
		set = fmt.Sprintf("%v", res.Priority)
	}

	return c.Set(ctx, model.Variables{
		argv.Set: set,
	})

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
