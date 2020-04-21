package email

import (
	"github.com/webitel/flow_manager/model"
	"net/http"
)

func (r *Router) reply(email model.EmailConnection, args interface{}) (model.Response, *model.AppError) {
	var props map[string]interface{}
	var ok bool

	if props, ok = args.(map[string]interface{}); !ok {
		return model.CallResponseError, model.NewAppError("Reply", "email.reply.valid.args", nil, "bad arguments", http.StatusBadRequest)
	}

	//TODO response..
	_, err := email.Reply(email.ParseText(getStringValueFromMap("body", props, "")))
	if err != nil {

		return model.CallResponseError, err
	}
	return model.CallResponseOK, nil
}
