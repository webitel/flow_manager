package processing

import (
	"context"
	"encoding/json"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) formFile(ctx context.Context, scope *flow.Flow, conn Connection, args interface{}) (model.Response, *model.AppError) {
	var argv model.FormFile

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.Id == "" {
		return nil, model.ErrorRequiredParameter("formComponent", "name")
	}

	val, _ := conn.Get(argv.Id)
	if val == "" {
		argv.Value = make([]interface{}, 0, 0)
	} else {
		argv.Value = setToJson(val)
	}

	conn.SetComponent(argv.Id, argv)

	return model.CallResponseOK, nil
}

func setToJson(src string) interface{} {
	var err error
	l := len(src)
	if l < 2 {
		return src
	}

	s := src[0:1]
	e := src[l-1:]

	if s == "{" && e == "}" {
		var res map[string]interface{}
		err = json.Unmarshal([]byte(src), &res)
		if err == nil {
			return res
		}
	} else if s == "[" && e == "]" {
		var res []interface{}
		err = json.Unmarshal([]byte(src), &res)
		if err == nil {
			return res
		}
	}

	return src
}
