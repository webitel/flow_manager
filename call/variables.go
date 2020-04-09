package call

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
	"net/http"
	"strconv"
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

func getIntValueFromMap(name string, params map[string]interface{}, def int) int {
	var ok bool
	var v interface{}
	var res int

	if v, ok = params[name]; ok {
		switch v.(type) {
		case int:
			return v.(int)
		case float64:
			return int(v.(float64))
		case float32:
			return int(v.(float32))
		case string:
			var err error
			if res, err = strconv.Atoi(v.(string)); err == nil {
				return res
			}
		}
	}

	return def
}

func getBoolValueFromMap(name string, params map[string]interface{}, def bool) bool {
	var ok bool
	if _, ok = params[name]; ok {
		if _, ok = params[name].(bool); ok {
			return params[name].(bool)
		}
	}
	return def
}
