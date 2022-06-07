package flow

import (
	"context"
	"encoding/json"

	"github.com/tidwall/gjson"

	"github.com/webitel/flow_manager/model"
)

func (r *router) set(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var vars model.Variables
	if err := scope.Decode(args, &vars); err != nil {
		return nil, err
	}

	for k, v := range vars {
		switch v.(type) {
		case map[string]interface{}:
			if data, err := json.Marshal(v); err == nil {
				vars[k] = gjson.ParseBytes(data)
			}
		}
	}

	return conn.Set(ctx, vars)
}
