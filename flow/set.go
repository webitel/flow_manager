package flow

import (
	"github.com/webitel/flow_manager/model"
)

func (r *Router) set(conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var vars model.Variables
	if err := Decode(conn, args, &vars); err != nil {
		return nil, err
	}

	return conn.Set(vars)
}
