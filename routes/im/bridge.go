package im

import (
	"context"
	"fmt"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type Bridge struct {
	UserID  int64 `json:"userId"`
	Timeout int   `json:"timeout"`
}

func (r *Router) bridge(ctx context.Context, scope *flow.Flow, conv Dialog, args any) (model.Response, *model.AppError) {
	var argv Bridge

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	wlog.Debug(fmt.Sprintf("conversation %s bridge to %d", conv.Id(), argv.UserID))

	if err := conv.Bridge(ctx, argv.UserID, argv.Timeout); err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
