package call

import (
	"github.com/webitel/flow_manager/model"
)

func (r *Router) conference(call model.Call, args interface{}) (model.Response, *model.AppError) {
	parameters, _ := args.(map[string]interface{})
	if parameters == nil {
		return model.CallResponseError, nil
	}

	var name = getStringValueFromMap("name", parameters, "global")
	var profile = getStringValueFromMap("profile", parameters, "default")
	return call.Conference(name, profile)
}
