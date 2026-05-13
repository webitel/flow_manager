package chat

import (
	"bytes"
	"context"
	"fmt"

	html "html/template"

	ingrpc "github.com/webitel/flow_manager/internal/adapters/inbound/grpc"
	"github.com/webitel/flow_manager/model"
)

// Adapter implements chat-manager–backed Deps methods.
type ChatMgrAdapter struct {
	mgr           *ingrpc.ChatManager
	templatesPath string
}

func NewChatMgrAdapter(mgr *ingrpc.ChatManager, templatesPath string) *ChatMgrAdapter {
	return &ChatMgrAdapter{mgr: mgr, templatesPath: templatesPath}
}

func (a *ChatMgrAdapter) BroadcastChatMessage(ctx context.Context, domainId int64, req model.BroadcastChat, peers []model.BroadcastPeer) (*model.BroadcastChatResponse, error) {
	return a.mgr.BroadcastMessage(ctx, domainId, req, peers)
}

func (a *ChatMgrAdapter) SenChatAction(ctx context.Context, channelId string, action model.ChatAction) error {
	if err := a.mgr.SendAction(ctx, channelId, action); err != nil {
		return fmt.Errorf("Chat: chat.send_action.error: %w", err)
	}
	return nil
}

func (a *ChatMgrAdapter) ContactLinkToChat(ctx context.Context, conversationId string, contactId string) error {
	if err := a.mgr.LinkContact(ctx, contactId, conversationId); err != nil {
		return fmt.Errorf("Chat: chat.link_contact.error: %w", err)
	}
	return nil
}

func (a *ChatMgrAdapter) ParseChatMessages(messages *[]model.ChatMessage, output string) (string, error) {
	if messages == nil || len(*messages) == 0 {
		return "", fmt.Errorf("messages is nil or empty")
	}
	var messageBuf, wrapperBuf bytes.Buffer
	wrapperTemplate, err := html.ParseFiles(fmt.Sprintf(a.templatesPath+"/%s/wrapper.%s", output, output))
	if err != nil {
		return "", err
	}
	for _, v := range *messages {
		tmpl, tmplErr := a.chatMessageTemplate(&v, output, v.IsInternal)
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

func (a *ChatMgrAdapter) chatMessageTemplate(message *model.ChatMessage, outputType string, isInternal bool) (*html.Template, error) {
	sender := "client"
	if isInternal {
		sender = "agent"
	}
	switch message.Type {
	case "text":
		return html.ParseFiles(fmt.Sprintf(a.templatesPath+"/%s/%s.%s", outputType, sender, outputType))
	case "file":
		return html.ParseFiles(fmt.Sprintf(a.templatesPath+"/%s/%s_file.%s", outputType, sender, outputType))
	}
	return nil, nil
}
