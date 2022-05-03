package processing

import (
	"context"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type GenerateFromArgs struct {
	Name    string                  `json:"name"`
	Title   string                  `json:"title"`
	Set     string                  `json:"set"`
	Actions []*model.FormActionElem `json:"actions"`
	Body    []string                `json:"body"`
}

func (r *Router) generateForm(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv GenerateFromArgs
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {

		return nil, err
	}

	f := model.FormElem{
		Name:    argv.Name,
		Title:   argv.Title,
		Actions: argv.Actions,
		Body:    make([]model.FormComponent, 0, len(argv.Body)),
	}

	for _, v := range argv.Body {
		e := conn.GetComponentByName(v)
		if e != nil {
			f.Body = append(f.Body, model.FormComponent{
				Name: v,
				View: e,
			})
		}
	}

	var action *model.FormAction
	action, err = conn.PushForm(f)
	if err != nil {

		return nil, err
	}

	if argv.Set != "" {
		_, err = conn.Set(ctx, model.Variables{
			argv.Set: action.Name,
		})
		if err != nil {

			return nil, err
		}
	}

	return conn.Set(ctx, action.Fields)
}
