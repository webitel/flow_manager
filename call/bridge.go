package call

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
	"net/http"
)

type EndpointVariableArgs struct {
	Type       string          `json:"type"`
	Name       *string         `json:"name"`
	Id         *int            `json:"id"`
	Parameters model.Variables `json:"parameters"`
}

type BridgeArgs struct {
	Strategy   string                 `json:"strategy"`
	Codecs     []string               `json:"codecs"`
	Parameters model.Variables        `json:"parameters"`
	Endpoints  []EndpointVariableArgs `json:"endpoints"`
}

func (r *Router) bridge(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError) {
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

	codecs, _ := getArrayStringFromMap("codecs", props)

	var e []*model.Endpoint
	e, err = getRemoteEndpoints(r, call, endpoints)
	if err != nil {
		return model.CallResponseError, err
	}
	res, err := call.Bridge(ctx, call, getStringValueFromMap("strategy", props, ""), nil, e, codecs)
	if err != nil {
		return res, err
	}

	//TODO variable_last_bridge_hangup_cause variable_bridge_hangup_cause
	if call.GetVariable("variable_bridge_hangup_cause") == "NORMAL_CLEARING" && call.GetVariable("variable_hangup_after_bridge") == "true" {
		scope.SetCancel()
	}

	//TODO
	if call.GetVariable("variable_last_bridge_hangup_cause") == "ORIGINATOR_CANCEL" &&
		call.GetVariable("variable_originate_disposition") == "ORIGINATOR_CANCEL" &&
		call.GetVariable("variable_sip_redirect_dialstring") != "" &&
		call.GetVariable("variable_webitel_detect_redirect") != "false" {
		wlog.Warn(fmt.Sprintf("call %s detect sip redirect to %s, break this route", call.Id(), call.GetVariable("variable_sip_redirect_dialstring")))
		scope.SetCancel()
	}

	return model.CallResponseOK, nil
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
				e.Number = model.NewString(getStringValueFromMap("dialString", endpoints[key], ""))
				e.Destination = model.NewString(fmt.Sprintf("%s@%s", *e.Number, *e.Destination))
			}
		case "user":
			//if e.Destination != nil {
			//e.Destination = model.NewString(fmt.Sprintf("%s@%s", *e.Destination, call.DomainName()))
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

func getArrayStringFromMap(name string, params map[string]interface{}) (res []string, ok bool) {
	var tmp []interface{}
	var i interface{}

	if _, ok = params[name]; !ok {
		return
	}

	if tmp, ok = params[name].([]interface{}); !ok {
		return
	}

	for _, i = range tmp {
		if _, ok = i.(string); ok {
			res = append(res, i.(string))
		}
	}
	ok = true
	return
}
