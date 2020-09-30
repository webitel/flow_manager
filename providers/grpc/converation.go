package grpc

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/flow_manager/providers/grpc/client"
	"github.com/webitel/wlog"
	"net/http"
	"sync"
	"time"
)

type conversationClient interface {
	Id() string
}

type message struct {
}

type conversation struct {
	id        int64
	profileId int64
	domainId  int64
	variables map[string]string
	client    client.ChatServiceClient
	mx        sync.RWMutex
	ctx       context.Context
	messages  []*message

	confirmation map[string]chan []string

	chat *chatApi
}

func NewConversation(client *ChatClientConnection, id, domainId, profileId int64) *conversation {
	return &conversation{
		id:           id,
		profileId:    profileId,
		domainId:     domainId,
		variables:    make(map[string]string),
		client:       client.api,
		mx:           sync.RWMutex{},
		ctx:          context.Background(),
		messages:     make([]*message, 5),
		confirmation: make(map[string]chan []string),
	}
}

func (c conversation) Type() model.ConnectionType {
	return model.ConnectionTypeChat
}

func (c *conversation) Id() string {
	return fmt.Sprintf("%d", c.id) //todo
}

func (c *conversation) NodeId() string {
	//TODO
	return "FIXME"
}

func (c *conversation) DomainId() int64 {
	return c.domainId
}

func (c *conversation) Context() context.Context {
	return c.ctx
}

func (c *conversation) Get(name string) (string, bool) {
	v, ok := c.variables[name]
	return v, ok
}

func (c *conversation) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v)
	}
	return model.CallResponseOK, nil
}

func (c *conversation) ParseText(text string) string {
	text = compileVar.ReplaceAllStringFunc(text, func(varName string) (out string) {
		r := compileVar.FindStringSubmatch(varName)
		if len(r) > 0 {
			out, _ = c.Get(r[1])
		}

		return
	})

	return text
}

func (c *conversation) Close() *model.AppError {
	return nil // fixme
}

func (c *conversation) Break() *model.AppError {
	c.ctx.Done() //todo
	return nil
}

func (c *conversation) ProfileId() int64 {
	return c.profileId
}

func (c *conversation) SendTextMessage(ctx context.Context, text string) (model.Response, *model.AppError) {
	_, err := c.client.SendMessage(ctx, &client.SendMessageRequest{
		ConversationId: c.id,
		FromFlow:       true,
		Message: &client.Message{
			Type: "text", // FIXME
			Value: &client.Message_TextMessage_{
				TextMessage: &client.Message_TextMessage{
					Text: text,
				},
			},
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendTextMessage", "conv.send.text.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) WaitMessage(ctx context.Context, timeout int) ([]string, *model.AppError) {
	id := model.NewId()

	res, err := c.client.WaitMessage(ctx, &client.WaitMessageRequest{
		ConversationId: c.id,
		ConfirmationId: id,
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.WaitMessage", "conv.wait.msg.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	if len(res.Messages) > 0 {
		//FIXME save msg
		msgs := make([]string, 0, len(res.Messages))
		for _, m := range res.Messages {
			switch x := m.Value.(type) {
			case *client.Message_TextMessage_:
				msgs = append(msgs, x.TextMessage.Text)
			}
		}

		return msgs, nil
	}

	if timeout == 0 {
		timeout = int(res.TimeoutSec)
	}

	t := time.After(time.Second * time.Duration(timeout))

	wlog.Debug(fmt.Sprintf("conversation %d wait message %s", c.id, time.Second*time.Duration(timeout)))

	ch := make(chan []string)
	c.mx.Lock()
	c.confirmation[id] = ch
	c.mx.Unlock()

	select {
	case <-t:
		wlog.Debug(fmt.Sprintf("conversation %d wait message: timeout", c.id))
		break
	case msgs := <-ch:
		wlog.Debug(fmt.Sprintf("conversation %d receive message: %s", c.id, msgs))
		return msgs, nil
	}

	return nil, model.NewAppError("Conversation.WaitMessage", "conv.timeout.msg.app_err", nil, "Timeout", http.StatusInternalServerError)
}

func (c *conversation) Stop(err *model.AppError) {
	var cause = ""
	if err != nil {
		wlog.Error(fmt.Sprintf("conversation %d stop with error: %s", c.id, err.Error()))
		cause = err.Id
	}

	_, e := c.client.CloseConversation(c.ctx, &client.CloseConversationRequest{
		ConversationId: c.id,
		FromFlow:       true,
		Cause:          cause,
	})

	if e != nil {
		wlog.Error(e.Error())
	}

	c.chat.conversations.Remove(c.id)
	wlog.Debug(fmt.Sprintf("close conversation %d", c.id))
}
