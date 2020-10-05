package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/webitel/engine/utils"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/grpc/workflow"
	"net/http"
)

const (
	activeConversationCacheSize = 50000
	maximumInactiveChat         = 60 * 60 * 24 // day
	confirmationBuffer          = 100
)

type chatApi struct {
	conversations utils.ObjectCache
	*server
}

func NewChatApi(s *server) *chatApi {
	return &chatApi{
		server:        s,
		conversations: utils.NewLru(activeConversationCacheSize),
	}
}

func (s *chatApi) Start(ctx context.Context, req *workflow.StartRequest) (*workflow.StartResponse, error) {
	if _, ok := s.conversations.Get(req.ConversationId); ok {
		return &workflow.StartResponse{
			Error: &workflow.Error{
				Id:      "grpc.chat.start.valid.conversation_id",
				Message: fmt.Sprintf("Conversation %d already exists", req.ConversationId),
			},
		}, nil
	}
	client, err := s.chatManager.getClient()
	if err != nil {
		return nil, err
	}

	conv := NewConversation(client, req.ConversationId, req.DomainId, req.ProfileId)
	conv.chat = s

	s.conversations.AddWithExpiresInSecs(req.ConversationId, conv, maximumInactiveChat)

	s.server.consume <- conv

	return &workflow.StartResponse{}, nil
}

func (s *chatApi) Break(_ context.Context, req *workflow.BreakRequest) (*workflow.BreakResponse, error) {
	conv, err := s.getConversation(req.ConversationId)
	if err != nil {
		return &workflow.BreakResponse{
			Error: &workflow.Error{
				Id:      err.Id,
				Message: err.Error(),
			},
		}, nil
	}

	if err := conv.Break(); err != nil {
		return &workflow.BreakResponse{
			Error: &workflow.Error{
				Id:      err.Id,
				Message: err.Error(),
			},
		}, nil
	}

	return &workflow.BreakResponse{}, nil
}

func (s *chatApi) ConfirmationMessage(_ context.Context, req *workflow.ConfirmationMessageRequest) (*workflow.ConfirmationMessageResponse, error) {
	var conf chan []string
	var ok bool

	conv, err := s.getConversation(req.ConversationId)
	if err != nil {
		return &workflow.ConfirmationMessageResponse{
			Error: &workflow.Error{
				Id:      err.Id,
				Message: err.Error(),
			},
		}, nil
	}

	conv.mx.RLock()
	conf, ok = conv.confirmation[req.ConfirmationId]
	if ok {
		delete(conv.confirmation, req.ConfirmationId)
	}
	conv.mx.RUnlock()

	if !ok {
		return &workflow.ConfirmationMessageResponse{
			Error: &workflow.Error{
				Id:      "chat.grpc.conversation.confirmation.not_found",
				Message: fmt.Sprintf("Confirmation %s not found", req.ConfirmationId),
			},
		}, nil
	}

	conf <- messageToText(req.Messages...)

	return &workflow.ConfirmationMessageResponse{}, nil
}

func (s *chatApi) BreakBridge(_ context.Context, in *workflow.BreakBridgeRequest) (*workflow.BreakBridgeResponse, error) {
	conv, err := s.getConversation(in.ConversationId)
	if err != nil {
		return nil, err
	}

	defer conv.mx.Unlock()
	conv.mx.Lock()
	if conv.chBridge == nil {
		return nil, errors.New("bridge not found")
	}

	close(conv.chBridge)
	conv.chBridge = nil

	return &workflow.BreakBridgeResponse{
		Error: nil,
	}, nil
}

func (s *chatApi) getConversation(id int64) (*conversation, *model.AppError) {
	conv, ok := s.conversations.Get(id)
	if !ok {
		return nil, model.NewAppError("Chat", "grpc.chat.conversation.not_found", nil,
			fmt.Sprintf("Conversation %d not found", id), http.StatusNotFound)
	}

	return conv.(*conversation), nil
}

func messageToText(messages ...*workflow.Message) []string {
	msgs := make([]string, 0, len(messages))

	for _, m := range messages {
		switch x := m.Value.(type) {
		case *workflow.Message_TextMessage_:
			msgs = append(msgs, x.TextMessage.Text)
		}
	}

	return msgs
}
