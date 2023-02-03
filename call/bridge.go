package call

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/lib/pq"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

type EndpointVariableArgs struct {
	Type       string              `json:"type"`
	Name       *string             `json:"name"`
	Gateway    *model.SearchEntity `json:"gateway"`
	Id         *int                `json:"id"`
	Idle       bool                `json:"idle"`
	Parameters model.Variables     `json:"parameters"`
}

type BridgeArgs struct {
	Strategy     string                 `json:"strategy"`
	SendOnAnswer string                 `json:"sendOnAnswer"`
	Codecs       []string               `json:"codecs"`
	Parameters   model.Variables        `json:"parameters"`
	Endpoints    []EndpointVariableArgs `json:"endpoints"`
	Bridged      []interface{}          `json:"bridged"`
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

	t := call.GetVariable("variable_transfer_history")
	var br chan struct{} = nil

	if hookApps, ok := props["bridged"].([]interface{}); ok {
		br = make(chan struct{})
		go func() {
			select {
			case _, ok := <-br:
				if ok {
					go flow.Route(ctx, scope.Fork("hook-bridged", flow.ArrInterfaceToArrayApplication(hookApps)), r)
				}
			}
		}()
	}

	var vars = make(map[string]string)
	if sendOnAnswer, ok := props["sendOnAnswer"].(string); ok {
		vars["execute_on_answer"] = "send_dtmf " + strings.Replace(sendOnAnswer, "'", "", -1)
	}

	if glob, ok := props["parameters"].(map[string]interface{}); ok && len(glob) > 0 {
		for k, v := range glob {
			vars[k] = fmt.Sprintf("%v", v)
		}
	}

	pickup := getStringValueFromMap("pickup", props, "")
	if pickup != "" {
		pickup = call.ParseText(pickup)
	}

	res, err := call.Bridge(ctx, call, getStringValueFromMap("strategy", props, ""), vars, e, codecs, br, pickup)
	if err != nil {
		return res, err
	}

	if t != call.GetVariable("variable_transfer_history") && (call.GetVariable("variable_hangup_after_bridge") == "" || call.GetVariable("variable_hangup_after_bridge") == "true") {
		scope.SetCancel()
	}

	//TODO variable_last_bridge_hangup_cause variable_bridge_hangup_cause
	if (call.GetVariable("variable_bridge_hangup_cause") == "NORMAL_CLEARING" || call.GetVariable("variable_last_bridge_hangup_cause") == "NORMAL_CLEARING") && call.GetVariable("variable_hangup_after_bridge") == "true" {
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
	//call.Dump()

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

			e.Variables = getVars(endpoints[key], e.Variables)

		case "user":
			e.Variables = getVars(endpoints[key], e.Variables)
			if e.HasPush {
				e.Variables = append(e.Variables, "execute_on_originate=wbt_send_hook")
			}
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

func getVars(src model.ApplicationObject, res pq.StringArray) pq.StringArray {
	if v, ok := src["parameters"].(map[string]interface{}); ok {
		for k, vv := range v {
			res = append(res, fmt.Sprintf("'%s'='%s'", k, vv))
		}
	}

	return res
}
