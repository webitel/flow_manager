package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
	"time"
)

type SleepArgs int

func (r *router) sleep(ctx context.Context, scope *Flow, c model.Connection, args interface{}) (model.Response, *model.AppError) {
	var timeout int
	if err := scope.Decode(args, &timeout); err != nil {
		return nil, err
	}

	if timeout > 0 {
		time.Sleep(time.Millisecond * (time.Duration(timeout)))
	}
	return ResponseOK, nil
}
