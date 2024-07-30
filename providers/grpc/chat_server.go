package grpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	chat "buf.build/gen/go/webitel/chat/protocolbuffers/go"
	gogrpc "buf.build/gen/go/webitel/workflow/grpc/go/_gogrpc"
	workflow "buf.build/gen/go/webitel/workflow/protocolbuffers/go"
	"github.com/webitel/engine/discovery"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"google.golang.org/grpc/metadata"
)

const (
	activeConversationCacheSize = 50000
	maximumInactiveChat         = 0 //30 * 60 * 60 * 24 // day
	confirmationBuffer          = 100
)

var (
	microServiceHeaderName = "Micro-From-Service"
	microServiceHeaderId   = "Micro-From-Id"
)

type chatApi struct {
	conversations utils.ObjectCache
	*server
	gogrpc.UnsafeFlowChatServerServiceServer
}

func NewChatApi(s *server) *chatApi {
	return &chatApi{
		server:        s,
		conversations: utils.NewLru(activeConversationCacheSize),
	}
}

func compactHeaderValue(src []string) string {
	if len(src) > 0 {
		return src[0]
	}

	return ""
}

func (s *chatApi) getClientFromRequest(ctx context.Context) (*ChatClientConnection, error) {
	if m, ok := metadata.FromIncomingContext(ctx); ok {
		id := fmt.Sprintf("%s-%s", compactHeaderValue(m.Get(microServiceHeaderName)),
			compactHeaderValue(m.Get(microServiceHeaderId)))
		return s.chatManager.getClient(id)
	}

	return nil, discovery.ErrNotFoundConnection
}

func (s *chatApi) Start(ctx context.Context, req *workflow.StartRequest) (*workflow.StartResponse, error) {
	if _, ok := s.conversations.Get(req.ConversationId); ok {
		//return nil, errors.New(fmt.Sprintf("Conversation %s already exists", req.ConversationId))
	}

	client, err := s.getClientFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	conv := NewConversation(client, req.ConversationId, req.DomainId, req.ProfileId, req.SchemaId, req.UserId)
	conv.chat = s

	if req.Message != nil {
		if req.Message.Variables != nil {
			for k, v := range req.Message.Variables {
				conv.variables.Store(k, v)
			}
		}

		conv.Set(ctx, model.Variables{
			model.ConversationStartMessageVariable: strings.Join(messageToText(req.Message), " "),
		})
		conv.saveMessages(req.Message)
	}
	conv.Set(ctx, map[string]interface{}{
		model.ConversationSessionId: conv.id,
		model.ConversationProfileId: conv.profileId,
	})

	s.conversations.AddWithExpiresInSecs(req.ConversationId, conv, maximumInactiveChat)

	s.server.consume <- conv

	return &workflow.StartResponse{}, nil
}

func (s *chatApi) Break(ctx context.Context, req *workflow.BreakRequest) (*workflow.BreakResponse, error) {
	conv, err := s.getConversationFromRequest(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}

	//todo if cause TRANSFER
	if err := conv.Break(req.Cause); err != nil {
		return nil, err
	}

	return &workflow.BreakResponse{}, nil
}

func (s *chatApi) ConfirmationMessage(ctx context.Context, req *workflow.ConfirmationMessageRequest) (*workflow.ConfirmationMessageResponse, error) {
	var conf chan []*chat.Message
	var ok bool

	conv, err := s.getConversationFromRequest(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}

	conv.mx.Lock()
	conf, ok = conv.confirmation[req.ConfirmationId]
	conv.mx.Unlock()

	if !ok {
		return nil, model.NewAppError("ConfirmationMessage", "chat.confirmation_message.not_found", nil, "Not found", http.StatusNotFound)
	}

	select {
	case <-time.After(time.Second * 5):
	case conf <- req.Messages:

	}

	return &workflow.ConfirmationMessageResponse{}, nil
}

func (s *chatApi) BreakBridge(ctx context.Context, in *workflow.BreakBridgeRequest) (*workflow.BreakBridgeResponse, error) {
	conv, err := s.getConversationFromRequest(ctx, in.ConversationId)
	if err != nil {
		return nil, err
	}

	isTransfer := strings.EqualFold(in.Cause, model.BreakChatTransferCause)
	conv.mx.Lock()
	br := conv.chBridge
	conv.mx.Unlock()

	if br == nil && !isTransfer {
		return nil, errors.New("bridge not found")
	}

	//todo
	if isTransfer {
		conv.mx.Lock()
		conv.breakCause = in.Cause
		conv.mx.Unlock()
		conv.setTransferVariable()
	}

	conv.closeIfBreak()

	return &workflow.BreakBridgeResponse{}, nil
}

func (s *chatApi) TransferChatPlan(ctx context.Context, in *workflow.TransferChatPlanRequest) (*workflow.TransferChatPlanResponse, error) {
	//todo
	return &workflow.TransferChatPlanResponse{}, nil
}

func (s *chatApi) getConversation(id string) (*conversation, *model.AppError) {
	conv, ok := s.conversations.Get(id)
	if !ok {
		return nil, model.NewAppError("Chat", "grpc.chat.conversation.not_found", nil,
			fmt.Sprintf("Conversation %s not found", id), http.StatusNotFound)
	}

	return conv.(*conversation), nil
}

func (s *chatApi) getConversationFromRequest(ctx context.Context, id string) (*conversation, *model.AppError) {
	var conv *conversation
	var appErr *model.AppError
	var err error
	var cli *ChatClientConnection

	conv, appErr = s.getConversation(id)
	if appErr != nil {
		return nil, appErr
	}

	cli, err = s.getClientFromRequest(ctx)
	if err != nil {
		return nil, model.NewAppError("Chat", "grpc.chat.client.not_found", nil,
			err.Error(), http.StatusNotFound)
	}

	conv.actualizeClient(cli)
	return conv, nil
}

func pettyMessage(msg *chat.Message) model.ChatMessage {
	m := model.ChatMessage{
		Text:       msg.Text,
		CreatedAt:  "",
		Type:       msg.Type,
		User:       "", // TODO
		IsInternal: true,
	}

	if m.Text == "" {
		if msg.Contact != nil {
			m.Text = msg.Contact.Contact
		} else if msg.File != nil {
			m.Text = msg.File.Name
		} // todo buttons ?
	}

	if msg.From != nil {
		m.User = msg.From.FirstName
		m.IsInternal = true
	}

	return m
}

func messageToText(messages ...*chat.Message) []string {
	msgs := make([]string, 0, len(messages))

	for _, m := range messages {
		switch m.Type {
		case "contact":
			if m.Contact != nil {
				msgs = append(msgs, m.Contact.Contact)
			}
		default:
			msgs = append(msgs, m.Text)
		}
	}

	return msgs
}
