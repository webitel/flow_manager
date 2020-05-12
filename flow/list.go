package flow

import "github.com/webitel/flow_manager/model"

type ListArgs struct {
	Name        *string
	Id          *int
	Destination string
}

func (r *Router) List(conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = ListArgs{}
	var exists bool
	err := Decode(conn, args, &argv)
	if err != nil {
		return nil, err
	}

	exists, err = r.fm.ListCheckNumber(conn.DomainId(), argv.Destination, argv.Id, argv.Name)
	if err != nil {
		return nil, err
	}

	if exists {

	}

	return model.CallResponseError, nil
}
