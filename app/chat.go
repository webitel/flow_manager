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
// All the available templates for this messages filtered by output type are lying in the message_templates folder.
func (fm *FlowManager) ParseChatMessages(messages *[]model.ChatMessage, output string) (string, *model.AppError) {
	if messages == nil || len(*messages) == 0 {
		return "", model.NewAppError("Flow", "flow_manager.parse_chat_messages.invalid_args", nil, "messages is nil or empty", http.StatusBadRequest)
	}
	var (
		messageBuf bytes.Buffer
		wrapperBuf bytes.Buffer
		name       string
	)
	// get wrapper for messages
	wrapperTemplate, err := html.ParseFiles(fmt.Sprintf(fm.config.ChatTemplatesSettings.Path+"/%s/wrapper.%s", output, output))
	if err != nil {
		return "", model.NewAppError("Flow", "flow_manager.parse_chat_messages.parse_wrapper_template.fail", nil, err.Error(), http.StatusInternalServerError)
	}

	// fill with data and insert all found messages to the message buffer
	for i, v := range *messages {
		if i == 0 { // check for sender or reciever
			name = v.User
		}
		template, appErr := fm.getChatMessageTemplate(&v, output, v.User == name)
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

func (fm *FlowManager) getChatMessageTemplate(message *model.ChatMessage, outputType string, isSender bool) (*html.Template, *model.AppError) {
	var sender string
	if isSender {
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

func (fm *FlowManager) BroadcastChatMessage(ctx context.Context, domainId int64, req model.BroadcastChat) *model.AppError {
	err := fm.chatManager.BroadcastMessage(ctx, domainId, req)
	if err != nil {
		return model.NewAppError("Chat", "chat.broadcast.error", nil, err.Error(), http.StatusInternalServerError)
	}

	return nil
}

func (c *FlowManager) LastBridgedChat(domainId int64, number, hours string, queueIds []int, mapRes model.Variables) (model.Variables, *model.AppError) {
	return c.Store.Chat().LastBridged(domainId, number, hours, queueIds, mapRes)
}
