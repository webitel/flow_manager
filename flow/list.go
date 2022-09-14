package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/model"
)

type ListArgs struct {
	Name        *string
	Id          *int
	List        model.SearchEntity
	Destination string
	Actions     []interface{}
}

type listAddCommunicationArgs struct {
	Destination string             `json:"destination"`
	Description *string            `json:"description"`
	ExpireAt    *int64             `json:"expireAt"`
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

	// todo deprecated
	if argv.List.Id == nil && argv.Id != nil {
		argv.List.Id = argv.Id
	}
	// todo deprecated
	if argv.List.Name == nil && argv.Name != nil {
		argv.List.Name = argv.Name
	}

	// fixme domain id in scope
	exists, err = r.fm.ListCheckNumber(scope.Connection.DomainId(), argv.Destination, argv.List.Id, argv.Name)
	if err != nil {
		return nil, err
	}

	if exists {
		scope2 := scope.Fork(fmt.Sprintf("list"), ArrInterfaceToArrayApplication(argv.Actions))
		Route(ctx, scope2, scope.handler)
		if scope2.IsCancel() {
			scope.SetCancel()
		}
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

	req := model.ListCommunication{
		Destination: argv.Destination,
		Description: argv.Description,
	}

	if argv.ExpireAt != nil && *argv.ExpireAt > 0 {
		t := time.Unix(0, *argv.ExpireAt*int64(time.Millisecond))
		if scope.timezone != nil {
			t.In(scope.timezone)
		}
		req.ExpireAt = &t
	}

	err = r.fm.ListAddCommunication(conn.DomainId(), &argv.List, &req)
	if err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
