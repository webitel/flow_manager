package webhook

import (
	"context"

	"github.com/webitel/flow_manager/providers/web_hook"

	"github.com/webitel/flow_manager/flow"

	"github.com/webitel/flow_manager/model"
)

type HttpResponse struct {
	Headers      map[string]string `json:"headers"`
	ResponseCode *int              `json:"responseCode" db:"responseCode"`
	Body         *model.JsonValue  `json:"body"`
}

func (r *Router) httpResponse(ctx context.Context, scope *flow.Flow, hook *web_hook.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv HttpResponse

	err := r.Decode(scope, args, &argv)
	if err != nil {
		return nil, err
	}

	for k, v := range argv.Headers {
		hook.SetHeader(k, v)
	}

	if argv.Body != nil && len(*argv.Body) > 0 {
		hook.WriteBody(*argv.Body)
	}

	if argv.ResponseCode != nil {
		hook.WriteCode(*argv.ResponseCode)
	}

	err = hook.Close()
	if err != nil {
		return model.CallResponseError, err
	}

	return model.CallResponseOK, nil
}
