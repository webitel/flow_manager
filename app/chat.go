package app

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	html "html/template"

	"github.com/webitel/flow_manager/model"
)

func (fm *FlowManager) GetChatRouteFromProfile(domainId, profileId int64) (*model.Routing, *model.AppError) {
	routing, err := fm.Store.Chat().RoutingFromProfile(domainId, profileId)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = fm.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}

func (fm *FlowManager) GetChatMessagesByConversationId(ctx context.Context, domainId int64, conversationId string, limit int64) (*[]model.ChatMessage, *model.AppError) {
	messages, err := fm.Store.Chat().GetMessagesByConversation(ctx, domainId, conversationId, limit)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// ParseChatMessages converts all chat message models to the given output type.
// All the available templates for these messages filtered by output type are lying in the message_templates folder.
func (fm *FlowManager) ParseChatMessages(messages *[]model.ChatMessage, output string) (string, *model.AppError) {
	if messages == nil || len(*messages) == 0 {
		return "", model.NewAppError("Flow", "flow_manager.parse_chat_messages.invalid_args", nil, "messages is nil or empty", http.StatusBadRequest)
	}
	var (
		messageBuf bytes.Buffer
		wrapperBuf bytes.Buffer
	)
	// get wrapper for messages
	wrapperTemplate, err := html.ParseFiles(fmt.Sprintf(fm.config.ChatTemplatesSettings.Path+"/%s/wrapper.%s", output, output))
	if err != nil {
		return "", model.NewAppError("Flow", "flow_manager.parse_chat_messages.parse_wrapper_template.fail", nil, err.Error(), http.StatusInternalServerError)
	}

	// fill with data and insert all found messages to the message buffer
	for _, v := range *messages {
		template, appErr := fm.getChatMessageTemplate(&v, output, v.IsInternal)
		if appErr != nil {
			return "", appErr
		}
		err := template.Execute(&messageBuf, v)
		if err != nil {
			return "", model.NewAppError("Flow", "flow_manager.parse_chat_messages.execute_template.fail", nil, err.Error(), http.StatusInternalServerError)
		}
	}
	// insert all messages inside a wrapper
	err = wrapperTemplate.Execute(&wrapperBuf, html.HTML(messageBuf.String()))
	if err != nil {
		return "", model.NewAppError("Flow", "flow_manager.parse_chat_messages.execute_template.fail", nil, err.Error(), http.StatusInternalServerError)
	}
	return wrapperBuf.String(), nil
}

func (fm *FlowManager) getChatMessageTemplate(message *model.ChatMessage, outputType string, isInternal bool) (*html.Template, *model.AppError) {
	var sender string
	if isInternal {
		sender = "agent"
	} else {
		sender = "client"
	}
	return fm.getMessageTemplateByType(message.Type, sender, outputType)
}

func (fm *FlowManager) getMessageTemplateByType(messageType string, sender string, outputType string) (*html.Template, *model.AppError) {
	var (
		template *html.Template
		err      error
	)
	switch messageType {
	case "text":
		template, err = html.ParseFiles(fmt.Sprintf(fm.config.ChatTemplatesSettings.Path+"/%s/%s.%s", outputType, sender, outputType))
		if err != nil {
			return nil, model.NewAppError("Flow", "flow_manager.parse_chat_messages.parse_template.fail", nil, err.Error(), http.StatusInternalServerError)
		}
	case "file":
		template, err = html.ParseFiles(fmt.Sprintf(fm.config.ChatTemplatesSettings.Path+"/%s/%s_file.%s", outputType, sender, outputType))
		if err != nil {
			return nil, model.NewAppError("Flow", "flow_manager.parse_chat_messages.parse_file_template.fail", nil, err.Error(), http.StatusInternalServerError)
		}
	}
	return template, nil
}

func (fm *FlowManager) GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*model.Routing, *model.AppError) {
	routing, err := fm.Store.Chat().RoutingFromSchemaId(domainId, schemaId)
	if err != nil {
		return nil, err
	}

	routing.Schema, err = fm.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}

func (fm *FlowManager) GetChatRouteFromUserId(domainId int64, userId int64) (*model.Routing, *model.AppError) {
	routing := &model.Routing{
		SourceId:   0,
		SourceName: "Blind transfer to user",
		SourceData: "Blind transfer to user",
		DomainId:   domainId,
		Schema: &model.Schema{
			DomainId: domainId,
			Name:     "transfer to user",
			Schema: model.Applications{
				{
					"bridge": map[string]interface{}{
						"userId": userId,
					},
				},
			},
		},
	}

	return routing, nil
}

func (fm *FlowManager) BroadcastChatMessage(ctx context.Context, domainId int64, req model.BroadcastChat, peers []model.BroadcastPeer) (*model.BroadcastChatResponse, *model.AppError) {
	resp, err := fm.chatManager.BroadcastMessage(ctx, domainId, req, peers)
	if err != nil {
		return nil, model.NewAppError("Chat", "chat.broadcast.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return resp, nil
}

func (c *FlowManager) LastBridgedChat(domainId int64, number, hours string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	return c.Store.Chat().LastBridged(domainId, number, hours, queueIds, mapRes)
}

func (c *FlowManager) ChatProfileType(domainId int64, profileId int) (string, *model.AppError) {
	return c.Store.Chat().ProfileType(domainId, profileId)
}

func (fm *FlowManager) SenChatAction(ctx context.Context, channelId string, action model.ChatAction) *model.AppError {
	err := fm.chatManager.SendAction(ctx, channelId, action)
	if err != nil {
		return model.NewAppError("Chat", "chat.send_action.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (fm *FlowManager) ContactLinkToChat(ctx context.Context, conversationId string, contactId string) *model.AppError {
	err := fm.chatManager.LinkContact(ctx, contactId, conversationId)
	if err != nil {
		return model.NewAppError("Chat", "chat.link_contact.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}
