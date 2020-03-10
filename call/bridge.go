package call

import (
	"encoding/json"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
)

func (r *Router) bridge(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var props map[string]interface{}
	var ok bool
	var endpoints model.Applications
	var err *model.AppError

	if props, ok = args.(map[string]interface{}); !ok {
		return model.CallResponseError, model.NewAppError("Bridge", "call.bridge.valid.args", nil, "bad arguments", http.StatusBadRequest)
	}

	if _, ok = props["endpoints"]; !ok {
		return model.CallResponseError, model.NewAppError("Bridge", "call.bridge.valid.endpoints", nil, "bad arguments", http.StatusBadRequest)
	}

	endpoints, err = replaceBridgeRequest(call.(model.Connection), props["endpoints"])
	if err != nil {
		return model.CallResponseError, model.NewAppError("Bridge", "call.bridge.valid.endpoints", nil, err.Error(), http.StatusBadRequest)
	}

	if len(endpoints) == 0 {
		return model.CallResponseError, model.NewAppError("Bridge", "call.bridge.valid.endpoints", nil, "bad arguments", http.StatusBadRequest)
	}

	var e []*model.Endpoint
	e, err = getRemoteEndpoints(r, call, endpoints)
	if err != nil {
		return model.CallResponseError, err
	}
	return call.Bridge(call, getStringValueFromMap("strategy", props, ""), nil, e)
}

func getRemoteEndpoints(r *Router, call model.Call, endpoints model.Applications) ([]*model.Endpoint, *model.AppError) {
	length := len(endpoints)
	endp, err := r.fm.Store.Endpoint().Get(int64(call.DomainId()), "NAME", "NUMBER", endpoints)
	if err != nil {
		return nil, err
	}

	for key, e := range endp {
		if key > length {
			break
		}

		switch e.TypeName {
		case "gateway":
			if e.Destination != nil {
				e.Destination = model.NewString(fmt.Sprintf("%s@%s", getStringValueFromMap("dialString", endpoints[key], ""), *e.Destination))
			}
		case "user":
			//if e.Destination != nil {
			//	e.Destination = model.NewString(fmt.Sprintf("%s@%s", *e.Destination, call.DomainName()))
			//}
		default:
			wlog.Warn(fmt.Sprintf("call %s skip bridge endpoint %v - unknown type ", call.Id(), e))
		}
	}

	return endp, nil
}

func replaceBridgeRequest(c model.Connection, arr interface{}) (model.Applications, *model.AppError) {
	data, err := json.Marshal(arr)
	var res model.Applications
	if err != nil {
		return nil, model.NewAppError("Bridge", "call.bridge.valid.endpoints", nil, "bad arguments", http.StatusBadRequest)
	}

	if err = json.Unmarshal([]byte(c.ParseText(string(data))), &res); err != nil {
		return nil, model.NewAppError("Bridge", "call.bridge.valid.endpoints", nil, "bad arguments", http.StatusBadRequest)
	}

	return res, nil
}
