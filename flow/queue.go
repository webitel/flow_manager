package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

type SearchQueue struct {
	Id   *int
	Name *string
}

type GetQueueInfo struct {
	BucketId    *int         `json:"bucket_id"`
	Metric      string       `json:"metric"`
	LastMinutes int          `json:"lastMinutes"`
	Queue       *SearchQueue `json:"queue"`
	Set         string
	Field       string `json:"field"` // ?????
	Calls       string `json:"calls"` // ?????
}

func (r *router) getQueueInfo(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GetQueueInfo
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.Queue == nil {
		return model.CallResponseError, ErrorRequiredParameter("getQueueInfo", "queue")
	}
	if argv.Set == "" {
		return model.CallResponseError, ErrorRequiredParameter("getQueueInfo", "set")
	}

	req := &model.SearchQueue{
		Id:          argv.Queue.Id,
		Name:        argv.Queue.Name,
		LastMinutes: argv.LastMinutes,
		Result:      nil,
		BucketId:    argv.BucketId,
	}

	res, err := r.fm.Store.Queue().Statistics(c.DomainId(), req, argv.Metric)

	if err != nil {
		return nil, err
	}

	return c.Set(ctx, model.Variables{
		argv.Set: res,
	})
}
