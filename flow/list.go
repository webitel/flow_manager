package flow

import "github.com/webitel/flow_manager/model"

type ListArgs struct {
	Name        *string
	Id          *int
	Destination string
}

// TODO
func (r *Router) List(conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = ListArgs{}
	err := Decode(conn, args, &argv)
	if err != nil {
		return nil, err
	}

	return model.CallResponseError, nil
}
