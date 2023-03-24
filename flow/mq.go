package flow

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

type MQArgs struct {
	Exchange string `json:"exchange"`
	Topic    string `json:"topic"`
	Body     interface{}
	SetErr   string `json:"setErr"`
}

func (r *router) mq(ctx context.Context, scope *Flow, call model.Connection, args interface{}) (model.Response, *model.AppError) {
	var argv MQArgs
	err := scope.Decode(args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Exchange == "" {
		return nil, ErrorRequiredParameter("mq", "exchange")
	}

	if argv.Topic == "" {
		return nil, ErrorRequiredParameter("mq", "topic")
	}

	data, jsonErr := json.Marshal(argv.Body)
	if jsonErr != nil {
		if argv.SetErr != "" {
			call.Set(ctx, map[string]interface{}{
				argv.SetErr: jsonErr.Error(),
			})
		}
		return model.CallResponseError, model.NewAppError("Flow", "flow.app.mq.valid.body", nil, jsonErr.Error(), http.StatusInternalServerError)
	}

	data = []byte(call.ParseText(string(data)))

	err = r.fm.SendMQJson(argv.Exchange, argv.Topic, data)
	if err != nil {
		if argv.SetErr != "" {
			call.Set(ctx, map[string]interface{}{
				argv.SetErr: err.Error(),
			})
		}
		return nil, err
	}

	return model.CallResponseOK, nil
}
