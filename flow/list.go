package flow

import "github.com/webitel/flow_manager/model"

type ListArgs struct {
	Name        *string
	Id          *int
	Destination string
}

func (r *router) List(scope *Flow, args interface{}) (model.Response, *model.AppError) {
	var argv = ListArgs{}
	var exists bool
	err := Decode(scope.Connection, args, &argv)
	if err != nil {
		return nil, err
	}

	// fixme domain id in scope
	exists, err = r.fm.ListCheckNumber(scope.Connection.DomainId(), argv.Destination, argv.Id, argv.Name)
	if err != nil {
		return nil, err
	}

	if exists {

	}

	return model.CallResponseError, nil
}
