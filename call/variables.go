package call

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type UnSetArg []string

func (r *Router) setAll(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	if vars, ok := args.(map[string]interface{}); ok {
		return call.SetAll(ctx, vars)
	}

	return nil, model.NewAppError("Call.SetAll", "router.call.set_all.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
}

func (r *Router) setNoLocal(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	if vars, ok := args.(map[string]interface{}); ok {
		return call.SetNoLocal(ctx, vars)
	}

	return nil, model.NewAppError("Call.SetAll", "router.call.set_all.valid.args", nil, fmt.Sprintf("bad arguments %v", args), http.StatusBadRequest)
}

func (r *Router) UnSet(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	var argv UnSetArg

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}
	if len(argv) == 0 {
		return nil, ErrorRequiredParameter("unSet", "value")
	}

	for _, v := range argv {
		if res, err := call.UnSet(ctx, v); err != nil {
			return res, err
		}
	}

	return model.CallResponseOK, nil
}

func getStringValueFromMap(name string, params map[string]interface{}, def string) (res string) {
	var ok bool
	var v interface{}

	if v, ok = params[name]; ok {

		switch v.(type) {
		case map[string]interface{}:
		case []interface{}:
			return def

		default:
			return fmt.Sprint(v)
		}
	}

	return def
}
