package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"strings"
)

/*
{
    "park": {
        "name": "myPark",
        "lot": "1000-2000",
        "auto": "in"
    }
}
*/
func (r *Router) park(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
	parameters, _ := args.(map[string]interface{})
	if parameters == nil {
		return model.CallResponseError, nil
	}

	var name = getStringValueFromMap("name", parameters, "")
	if name == "" {
		return model.CallResponseError, ErrorRequiredParameter("park", "name")
	}

	var lot = getStringValueFromMap("lot", parameters, "")
	if lot == "" {
		return model.CallResponseError, ErrorRequiredParameter("park", "lot")
	}

	var auto = getStringValueFromMap("auto", parameters, "in")

	lots := strings.Split(lot, "-")
	var fromLot, toLot string
	if len(lots) > 0 {
		fromLot = call.ParseText(lots[0])
	}
	if len(lots) > 1 {
		toLot = call.ParseText(lots[1])
	}

	return call.Park(ctx, name, call.ParseText(auto) == "in", fromLot, toLot)
}
