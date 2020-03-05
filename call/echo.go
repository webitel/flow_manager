package call

import (
	"github.com/webitel/flow_manager/model"
	"strconv"
)

func (r *Router) echo(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var delay = 0
	switch args.(type) {
	case string:
		delay, _ = strconv.Atoi(args.(string))

	case int:
		delay = args.(int)
	}

	return call.Echo(delay)
}
