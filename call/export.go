package call

import (
	"fmt"
	"github.com/webitel/flow_manager/model"
)

func (r *Router) export(call model.Call, args interface{}) (model.Response, *model.AppError) {
	switch args.(type) {
	case []string:
		return call.Export(args.([]string))
	case []interface{}:
		vars := make([]string, 0, len(args.([]interface{})))
		for _, v := range args.([]interface{}) {
			vars = append(vars, fmt.Sprintf("%v", v))
		}
		return call.Export(vars)
	default:
		return model.CallResponseError, nil
	}
}
