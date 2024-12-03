package flow

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/webitel/flow_manager/model"
)

func (r *router) broadcastChatMessage(ctx context.Context, scope *Flow, conn model.Connection, args interface{}) (model.Response, *model.AppError) {
	var err *model.AppError
	var argv = model.BroadcastChat{
		Type: "text",
	}

	if err = scope.Decode(args, &argv); err != nil {
		return nil, err
	}

	if len(argv.Peer) == 0 {
		return nil, ErrorRequiredParameter("broadcastChatMessage", "peer")
	}

	resp, err := r.fm.BroadcastChatMessage(ctx, conn.DomainId(), argv)
	if err != nil {
		return nil, err
	}
	if len(resp.Failed) != 0 && (argv.FailedReceivers != "" || argv.ResponseCode != "") {
		// save previous logic with response code saved from first peer error message
		status, err := conn.Set(ctx, model.Variables{
			argv.ResponseCode: resp.Failed[0].Error,
		})
		if err != nil {
			return status, err
		}

		// new logic when all failed receivers saved to the variable
		bytes, commonError := json.Marshal(resp)
		if commonError != nil {
			return nil, model.NewAppError("", "flow.chat.broadcast_chat_message.marshal_failed.marshal_error", nil, commonError.Error(), http.StatusInternalServerError)
		}
		status, err = conn.Set(ctx, model.Variables{
			argv.FailedReceivers: string(bytes),
		})
		if err != nil {
			return status, err
		}
	}

	// if the chat_manager service wants to set new variables let him do this
	for key, value := range resp.Variables {
		status, err := conn.Set(ctx, model.Variables{
			key: value,
		})
		if err != nil {
			return status, err
		}
	}

	return model.CallResponseOK, nil
}
