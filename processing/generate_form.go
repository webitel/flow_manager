package processing

import (
	"context"
	"encoding/json"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type GenerateFromArgs struct {
	Id      string                  `json:"id"`
	Title   string                  `json:"title"`
	Actions []*model.FormActionElem `json:"actions"`
	Body    []string                `json:"body"`
}

type Tst struct {
	Id string `json:"id"`
}

func (r *Router) generateForm(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GenerateFromArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {

		return nil, err
	}

	f := model.FormElem{
		Id:      argv.Id,
		Title:   argv.Title,
		Actions: argv.Actions,
		Body:    make([]interface{}, 0, len(argv.Body)),
	}

	for _, v := range argv.Body {
		c := conn.GetComponentByName(v)

		if cmp, er := json.Marshal(c); er == nil {
			var t Tst
			json.Unmarshal(cmp, &t)
			if t.Id != "" {
				f.Body = append(f.Body, c)
			}

		}
	}

	var action *model.FormAction
	action, err = conn.PushForm(f)
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
