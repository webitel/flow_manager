package email

import (
	"context"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type Reply struct {
	Body string
}

func (r *Router) reply(ctx context.Context, scope *flow.Flow, email model.EmailConnection, args interface{}) (model.Response, *model.AppError) {
	var argv Reply

	err := r.Decode(scope, args, &argv)
	if err != nil {
		return nil, err
	}

	if argv.Body == "" {
		return model.CallResponseError, model.NewAppError("Reply", "email.reply.valid.args", nil, "bad arguments", http.StatusBadRequest)
	}

	err = r.fm.ReplyEmail(email, argv.Body)
	if err != nil {
		return nil, err
	}

	return model.CallResponseOK, nil
}
