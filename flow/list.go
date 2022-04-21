package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
)

type ListArgs struct {
	Name        *string
	Id          *int
	Destination string
	Actions     []interface{}
}

type listAddCommunicationArgs struct {
	Destination string             `json:"destination"`
	Description *string            `json:"description"`
	List        model.SearchEntity `json:"list"`
}

func (r *router) List(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = ListArgs{}
	var exists bool
	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Actions == nil || len(argv.Actions) == 0 {
		return nil, ErrorRequiredParameter("list", "actions")
	}

	// fixme domain id in scope
	exists, err = r.fm.ListCheckNumber(scope.Connection.DomainId(), argv.Destination, argv.Id, argv.Name)
	if err != nil {
		return nil, err
	}

	if exists {
		scope2 := scope.Fork(fmt.Sprintf("list"), ArrInterfaceToArrayApplication(argv.Actions))
		Route(ctx, scope2, scope.handler)
		// cancel root scope ?
		scope.SetCancel()
	}

	return model.CallResponseOK, nil
}

func (r *router) listAddCommunication(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = listAddCommunicationArgs{}
	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}
	if argv.Destination == "" {
		return nil, ErrorRequiredParameter("listAdd", "destination")
	}

	err = r.fm.ListAddCommunication(conn.DomainId(), &argv.List, &model.ListCommunication{
		Destination: argv.Destination,
		Description: argv.Description,
	})
	if err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
