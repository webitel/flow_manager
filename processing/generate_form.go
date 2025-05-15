package processing

import (
	"context"
	"github.com/webitel/flow_manager/pkg/processing"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type GenerateFromArgs struct {
	Id      string                       `json:"id"`
	Title   string                       `json:"title"`
	Actions []*processing.FormActionElem `json:"actions"`
	Body    []string                     `json:"body"`
}

func (r *Router) generateForm(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GenerateFromArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {

		return nil, err
	}

	f := processing.FormElem{
		Id:      argv.Id,
		Title:   argv.Title,
		Actions: argv.Actions,
		Body:    make([]interface{}, 0, len(argv.Body)),
	}

	for _, v := range argv.Body {
		c := conn.GetComponentByName(v)
		if c != nil {
			f.Body = append(f.Body, c)
		}
	}

	var action *processing.FormAction
	action, err = conn.PushForm(ctx, f)
	if err != nil {

		return nil, err
	}

	if argv.Id != "" {
		if action.Fields == nil {
			action.Fields = make(model.Variables)
		}
		action.Fields[argv.Id] = action.Name
	}

	return conn.Set(ctx, action.Fields)
}
