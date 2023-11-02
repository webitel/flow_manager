package chat_route

import (
	"context"
	"strings"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type TextMessage string

type ChatAction struct {
	Action model.ChatAction
}

func (r *Router) sendMessage(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv model.ChatMessageOutbound
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if argv.File != nil {
		if argv.File.Url != "" && argv.File.Id == 0 {
			argv.File.Id = 1
		} else {
			server := argv.File.Server

			if strings.HasSuffix(server, "/") {
				server = server[:len(server)-1]
			}

			if argv.File, err = r.fm.SearchMediaFile(conv.DomainId(), &model.SearchFile{
				Id:   argv.File.Id,
				Name: argv.File.Name,
			}); err != nil {
				return nil, err
			}

			argv.File, err = r.fm.SetupPublicFileUrl(argv.File, conv.DomainId(), server, "media", 604800)
			if err != nil {
				return nil, err
			}

			argv.File.MimeType += ";source=media"
		}
	}

	return conv.SendMessage(ctx, argv)
}

func (r *Router) sendText(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv TextMessage

	if err := r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	return conv.SendTextMessage(ctx, string(argv))
}

func (r *Router) sendAction(ctx context.Context, scope *flow.Flow, conv Conversation, args interface{}) (model.Response, *model.AppError) {
	var argv ChatAction
	var err *model.AppError

	if err = r.Decode(scope, args, &argv); err != nil {
		return nil, err
	}

	if err = r.fm.SenChatAction(ctx, conv.Id(), argv.Action); err != nil {
		return model.CallResponseError, err
	}

	return model.CallResponseOK, nil

}
