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
	routing, storeErr := fm.Store.Chat().RoutingFromProfile(domainId, profileId)
	if storeErr != nil {
		return nil, model.NewAppError("GetChatRouteFromProfile", "store.chat.routing_from_profile", nil, storeErr.Error(), http.StatusInternalServerError)
	}

	var err *model.AppError
	routing.Schema, err = fm.GetSchema(domainId, routing.SchemaId, routing.SchemaUpdatedAt)
	if err != nil {
		return nil, err
	}

	return routing, nil
}


// ParseChatMessages converts all chat message models to the given output type.
// All the available templates for these messages filtered by output type are lying in the message_templates folder.
func (fm *FlowManager) ParseChatMessages(messages *[]model.ChatMessage, output string) (string, error) {
	if messages == nil || len(*messages) == 0 {
		return "", fmt.Errorf("messages is nil or empty")
	}
	var (
		messageBuf bytes.Buffer
		wrapperBuf bytes.Buffer
	)
	wrapperTemplate, err := html.ParseFiles(fmt.Sprintf(fm.config.ChatTemplatesSettings.Path+"/%s/wrapper.%s", output, output))
	if err != nil {
		return "", err
	}

	for _, v := range *messages {
		tmpl, tmplErr := fm.getChatMessageTemplate(&v, output, v.IsInternal)
		if tmplErr != nil {
			return "", tmplErr
		}
		if err := tmpl.Execute(&messageBuf, v); err != nil {
			return "", err
		}
	}
	if err = wrapperTemplate.Execute(&wrapperBuf, html.HTML(messageBuf.String())); err != nil {
		return "", err
	}
	return wrapperBuf.String(), nil
}

func (fm *FlowManager) getChatMessageTemplate(message *model.ChatMessage, outputType string, isInternal bool) (*html.Template, error) {
	sender := "client"
	if isInternal {
		sender = "agent"
	}
	return fm.getMessageTemplateByType(message.Type, sender, outputType)
}

func (fm *FlowManager) getMessageTemplateByType(messageType string, sender string, outputType string) (*html.Template, error) {
	switch messageType {
	case "text":
		return html.ParseFiles(fmt.Sprintf(fm.config.ChatTemplatesSettings.Path+"/%s/%s.%s", outputType, sender, outputType))
	case "file":
		return html.ParseFiles(fmt.Sprintf(fm.config.ChatTemplatesSettings.Path+"/%s/%s_file.%s", outputType, sender, outputType))
	}
	return nil, nil
}

func (fm *FlowManager) GetChatRouteFromSchemaId(domainId int64, schemaId int32) (*model.Routing, *model.AppError) {
	routing, storeErr := fm.Store.Chat().RoutingFromSchemaId(domainId, schemaId)
	if storeErr != nil {
		return nil, model.NewAppError("GetChatRouteFromSchemaId", "store.chat.routing_from_schema", nil, storeErr.Error(), http.StatusInternalServerError)
	}

	var err *model.AppError
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

func (fm *FlowManager) BroadcastChatMessage(ctx context.Context, domainId int64, req model.BroadcastChat, peers []model.BroadcastPeer) (*model.BroadcastChatResponse, error) {
	return fm.chatManager.BroadcastMessage(ctx, domainId, req, peers)
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
