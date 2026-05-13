package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/webitel/flow_manager/api/gen/chat"
	workflow2 "github.com/webitel/flow_manager/api/gen/workflow"
	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/infrastructure/discovery"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	infraCache "github.com/webitel/flow_manager/internal/infrastructure/cache"
)

const (
	activeConversationCacheSize = 50000
	maximumInactiveChat         = 0 // 30 * 60 * 60 * 24 // day
	confirmationBuffer          = 100
)

var (
	microServiceHeaderName = "Micro-From-Service"
	microServiceHeaderId   = "Micro-From-Id"
)

type chatApi struct {
	conversations infraCache.ObjectCache
	*Server
	workflow2.UnsafeFlowChatServerServiceServer
}

func NewChatApi(s *Server) *chatApi {
	return &chatApi{
		Server:        s,
		conversations: infraCache.NewLru(activeConversationCacheSize),
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

func (s *chatApi) Start(ctx context.Context, req *workflow2.StartRequest) (*workflow2.StartResponse, error) {
	if _, ok := s.conversations.Get(req.ConversationId); ok {
		// return nil, errors.New(fmt.Sprintf("Conversation %s already exists", req.ConversationId))
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

		conv.Set(ctx, flow.Variables{
			chatdomain.ConversationStartMessageVariable: strings.Join(messageToText(req.Message), " "),
		})
		conv.storeMessages[chatdomain.ConversationStartMessageVariable], _ = json.Marshal(req.Message)
		conv.saveMessages(req.Message)
	}
	conv.Set(ctx, map[string]any{
		chatdomain.ConversationSessionId: conv.id,
		chatdomain.ConversationProfileId: conv.profileId,
	})

	s.conversations.AddWithExpiresInSecs(req.ConversationId, conv, maximumInactiveChat)

	s.Server.consume <- conv

	return &workflow2.StartResponse{}, nil
}

func (s *chatApi) Break(ctx context.Context, req *workflow2.BreakRequest) (*workflow2.BreakResponse, error) {
	conv, err := s.getConversationFromRequest(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}

	// todo if cause TRANSFER
	if err := conv.Break(req.Cause); err != nil {
		return nil, err
	}

	return &workflow2.BreakResponse{}, nil
}

func (s *chatApi) ConfirmationMessage(ctx context.Context, req *workflow2.ConfirmationMessageRequest) (*workflow2.ConfirmationMessageResponse, error) {
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
		// No active WaitMessage subscription: the runtime may have a native
		// recvMessage or softSleep suspend with an OnInboundMessage handler.
		// Fire those handlers instead of returning an error so the message is
		// not silently dropped.
		text := strings.Join(messageToText(req.Messages...), " ")
		conv.fireInboundHandlers(text)
		return &workflow2.ConfirmationMessageResponse{}, nil
	}

	select {
	case <-time.After(time.Second * 5):
	case conf <- req.Messages:

	}

	return &workflow2.ConfirmationMessageResponse{}, nil
}

func (s *chatApi) BreakBridge(ctx context.Context, in *workflow2.BreakBridgeRequest) (*workflow2.BreakBridgeResponse, error) {
	conv, err := s.getConversationFromRequest(ctx, in.ConversationId)
	if err != nil {
		return nil, err
	}

	isTransfer := strings.EqualFold(in.Cause, chatdomain.BreakChatTransferCause)
	conv.mx.Lock()
	br := conv.chBridge
	conv.mx.Unlock()

	if br == nil && !isTransfer {
		return nil, errors.New("bridge not found")
	}

	// todo
	if isTransfer {
		conv.mx.Lock()
		conv.breakCause = in.Cause
		conv.mx.Unlock()
		conv.setTransferVariable()
	}

	conv.closeIfBreak()

	return &workflow2.BreakBridgeResponse{}, nil
}

func (s *chatApi) TransferChatPlan(ctx context.Context, in *workflow2.TransferChatPlanRequest) (*workflow2.TransferChatPlanResponse, error) {
	// todo
	return &workflow2.TransferChatPlanResponse{}, nil
}

func (s *chatApi) getConversation(id string) (*conversation, error) {
	conv, ok := s.conversations.Get(id)
	if !ok {
		return nil, apperrs.Newf(http.StatusNotFound, "Chat: grpc.chat.conversation.not_found: Conversation %s not found", id)
	}

	return conv.(*conversation), nil
}

func (s *chatApi) getConversationFromRequest(ctx context.Context, id string) (*conversation, error) {
	var conv *conversation
	var err error
	var cli *ChatClientConnection

	conv, err = s.getConversation(id)
	if err != nil {
		return nil, err
	}

	cli, err = s.getClientFromRequest(ctx)
	if err != nil {
		return nil, apperrs.Newf(http.StatusNotFound, "Chat: grpc.chat.client.not_found: %s", err.Error())
	}

	conv.actualizeClient(cli)
	return conv, nil
}

func pettyMessage(msg *chat.Message) chatdomain.ChatMessage {
	m := chatdomain.ChatMessage{
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
