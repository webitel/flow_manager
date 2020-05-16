package flow

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"time"
)

/*
WTEL-902
*/

type TimezoneArgs struct {
	Name *string
	Id   *int
}

func (r *router) SetTimezone(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv = TimezoneArgs{}
	var loc *time.Location

	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Id != nil {
		loc = r.fm.GetLocation(*argv.Id)
	} else if argv.Name != nil {
		loc, _ = time.LoadLocation(*argv.Name)
	}

	if loc == nil {
		return ResponseErr, nil
	}

	scope.SetTimezone(loc)
	return ResponseOK, nil
}

func (i *Flow) Now() time.Time {
	i.RLock()
	defer i.RUnlock()
	if i.timezone != nil {
		return time.Now().In(i.timezone)
	}

	return time.Now()
}

func (i *Flow) SetTimezone(loc *time.Location) {
	old := i.timezone
	i.Lock()
	i.timezone = loc
	i.Unlock()
	if old == nil {
		wlog.Debug(fmt.Sprintf("scope \"%s\" new location %s", i.name, loc.String()))
	} else {
		wlog.Debug(fmt.Sprintf("scope \"%s\" changed location from %s to %s", i.name, old.String(), loc.String()))
	}
}
