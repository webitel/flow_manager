package flow

import (
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type LogArg string

func (r *router) Log(conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var log LogArg
	if err := Decode(conn, args, &log); err != nil {
		return nil, err
	}

	// send FS ?
	wlog.Info(string(log))

	return model.CallResponseOK, nil
}
