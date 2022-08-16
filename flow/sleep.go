package flow

import (
	"context"
	"time"

	"github.com/webitel/flow_manager/model"
)

type SleepArgs int

func (r *router) sleep(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var timeout int
	if err := scope.Decode(args, &timeout); err != nil {
		return nil, err
	}

	if timeout > 0 {
		select {
		case <-c.Context().Done():
			return ResponseErr, nil
		case <-time.After(time.Millisecond * (time.Duration(timeout))):
			return ResponseOK, nil
		}
	}
	return ResponseOK, nil
}
