package flow

import (
	"context"
	"github.com/webitel/flow_manager/model"
)

func (r *router) set(conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var vars model.Variables
	if err := Decode(conn, args, &vars); err != nil {
		return nil, err
	}

	return conn.Set(context.Background(), vars)
}
