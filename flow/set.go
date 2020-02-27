package flow

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (r *Router) set(conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	if vars, ok := args.(map[string]interface{}); ok {
		return conn.Set(vars)
	}

	return nil, model.NewAppError("Flow.Set", "flow.app.set.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
}
