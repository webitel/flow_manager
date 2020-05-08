package call

import (
	"encoding/json"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func testArgs(args interface{}) []byte {
	data, _ := json.Marshal(args)
	return data
}

type SipRedirectArgs []string

func (r *Router) FromJson(data []byte, res interface{}) *model.AppError {
	err := json.Unmarshal(data, res)
	if err != nil {
		return model.NewAppError("Router", "router.parser.err", nil, err.Error(), http.StatusBadRequest)
	}

	return nil
}

func (r *Router) SipRedirect(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var uri SipRedirectArgs
	if err := r.FromJson(testArgs(args), &uri); err != nil {
		return nil, err
	}

	return call.Redirect(uri)
}
