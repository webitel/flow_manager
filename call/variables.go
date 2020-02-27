package call

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (r *Router) setAll(call model.Call, args interface{}) (model.Response, *model.AppError) {
	if vars, ok := args.(map[string]interface{}); ok {
		return call.SetAll(vars)
	}

	return nil, model.NewAppError("Call.SetAll", "router.call.set_all.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
}

func (r *Router) setNoLocal(call model.Call, args interface{}) (model.Response, *model.AppError) {
	if vars, ok := args.(map[string]interface{}); ok {
		return call.SetNoLocal(vars)
	}

	return nil, model.NewAppError("Call.SetAll", "router.call.set_all.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
}
