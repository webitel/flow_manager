package flow

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

/*
{
    "calendar": {
        "name": "my Business Calendar",
        "extended": false,
        "setVar": "isWorkDay"
    }
}
*/

type CalendarArgs struct {
	Name     *string
	Id       *int
	SetVar   string `json:"setVar"`
	Extended bool
}

func (r *router) Calendar(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv CalendarArgs
	var value = "false"
	if err := scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if argv.SetVar == "" {
		return nil, ErrorRequiredParameter("calendar", "setVar")
	}

	check, err := r.fm.CheckCalendar(conn.DomainId(), argv.Id, argv.Name)
	if err != nil {
		return nil, err
	}

	if check.Accept && !check.Expire && check.Excepted == nil {
		value = "true"
	} else if argv.Extended {
		if check.Expire {
			value = "expire"
		} else if check.Excepted != nil && *check.Excepted != "" {
			value = *check.Excepted
		} else {
			// TODO ahead
		}
	}

	return conn.Set(context.Background(), map[string]interface{}{
		argv.SetVar: value,
	})
}
