package grpc

import (
	"context"
	"fmt"
	"github.com/webitel/flow_manager/model"
	client "github.com/webitel/protos/engine/chat"
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
	id        string
	profileId int64
	domainId  int64
	variables map[string]string
	client    *ChatClientConnection
	mx        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	messages  []*message
	chBridge  chan struct{}

	confirmation    map[string]chan []string
	exportVariables []string
	nodeId          string

	chat *chatApi
}

func NewConversation(client *ChatClientConnection, id string, domainId, profileId int64) *conversation {
	ctx, cancel := context.WithCancel(context.Background())
	return &conversation{
		id:           id,
		profileId:    profileId,
		domainId:     domainId,
		variables:    make(map[string]string),
		client:       client,
		chBridge:     nil,
		mx:           sync.RWMutex{},
		ctx:          ctx,
		cancel:       cancel,
		messages:     make([]*message, 5),
		confirmation: make(map[string]chan []string),
		nodeId:       client.Name(),
	}
}

func (c conversation) Type() model.ConnectionType {
	return model.ConnectionTypeChat
}

func (c *conversation) Id() string {
	return c.id
}

func (c *conversation) NodeId() string {
	return c.nodeId
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
	c.mx.Lock()
	if c.chBridge != nil {
		close(c.chBridge)
	}
	c.mx.Unlock()

	c.cancel()
	return nil
}

func (c *conversation) ProfileId() int64 {
	return c.profileId
}

//todo check not closed
func (c *conversation) SendTextMessage(ctx context.Context, text string) (model.Response, *model.AppError) {
	_, err := c.client.api.SendMessage(ctx, &client.SendMessageRequest{
		ConversationId: c.id,
		Message: &client.Message{
			Type: "text", // FIXME
			Text: text,
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendTextMessage", "conv.send.text.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

//todo refactoring
func (c *conversation) SendMenu(ctx context.Context, menu *model.ChatMenuArgs) (model.Response, *model.AppError) {
	req := &client.Message{
		Type:    "text",
		Text:    menu.Text,
		Buttons: make([]*client.Buttons, 0, len(menu.Buttons)),
	}
	//menu.Set // fixme

	for _, v := range menu.Buttons {
		btns := make([]*client.Button, 0, len(v))
		for _, b := range v {
			btns = append(btns, &client.Button{
				Text: b.Text,
				Type: b.Type,
				Url:  b.Url,
				Code: b.Code,
			})
		}
		req.Buttons = append(req.Buttons, &client.Buttons{
			Button: btns,
		})
	}

	_, err := c.client.api.SendMessage(ctx, &client.SendMessageRequest{
		Message:        req,
		ConversationId: c.Id(),
	})
	if err != nil {
		return nil, model.NewAppError("Conversation.SendMenu", "conv.send.menu.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil

}

// todo from media file
func (c *conversation) SendImageMessage(ctx context.Context, url string) (model.Response, *model.AppError) {
	_, err := c.client.api.SendMessage(ctx, &client.SendMessageRequest{
		ConversationId: c.id,
		Message: &client.Message{
			Type: "file", // FIXME
			File: &client.File{
				Url: url,
			},
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendTextMessage", "conv.send.text.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) ReceiveMessage(ctx context.Context, timeout int) ([]string, *model.AppError) {
	id := model.NewId()

	// TODO rename server api
	res, err := c.client.api.WaitMessage(ctx, &client.WaitMessageRequest{
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
			msgs = append(msgs, m.Text)
		}

		return msgs, nil
	}

	if timeout == 0 {
		timeout = int(res.TimeoutSec)
	}

	t := time.After(time.Second * time.Duration(timeout))

	wlog.Debug(fmt.Sprintf("conversation %s wait message %s", c.id, time.Second*time.Duration(timeout)))

	ch := make(chan []string)
	c.mx.Lock()
	c.confirmation[id] = ch
	c.mx.Unlock()

	select {
	case <-t:
		wlog.Debug(fmt.Sprintf("conversation %s wait message: timeout", c.id))
		break
	case msgs := <-ch:
		wlog.Debug(fmt.Sprintf("conversation %s receive message: %s", c.id, msgs))
		return msgs, nil
	}

	return nil, model.NewAppError("Conversation.WaitMessage", "conv.timeout.msg.app_err", nil, "Timeout", http.StatusInternalServerError)
}

func (c *conversation) NodeName() string {
	return c.NodeId()
}

func (c *conversation) Stop(err *model.AppError) {
	var cause = ""
	if err != nil {
		wlog.Error(fmt.Sprintf("conversation %d stop with error: %s", c.id, err.Error()))
		cause = err.Id
	}

	_, e := c.client.api.CloseConversation(c.ctx, &client.CloseConversationRequest{
		ConversationId: c.id,
		Cause:          cause,
	})

	if e != nil {
		wlog.Error(e.Error())
	}

	c.chat.conversations.Remove(c.id)
	wlog.Debug(fmt.Sprintf("close conversation %s", c.id))
}

func (c *conversation) Export(ctx context.Context, vars []string) (model.Response, *model.AppError) {
	exp := make(map[string]interface{})
	for _, v := range vars {
		exp[fmt.Sprintf("usr_%s", v)], _ = c.Get(v)

		c.exportVariables = append(c.exportVariables, v)
	}

	if len(exp) > 0 {
		return c.Set(ctx, exp)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) Bridge(ctx context.Context, userId int64, timeout int) *model.AppError {

	if c.chBridge != nil {
		return model.NewAppError("Conversation.Bridge", "conv.bridge.app_err", nil, "Not allow two bridge", http.StatusInternalServerError)
	}
	c.chBridge = make(chan struct{})

	vars := make(map[string]string)

	if len(c.exportVariables) > 0 {
		for _, v := range c.exportVariables {
			if val, ok := c.Get(v); ok {
				vars[v] = val
			}
		}
	}

	res, err := c.client.api.InviteToConversation(ctx, &client.InviteToConversationRequest{
		User: &client.User{
			UserId:   userId,
			Type:     "webitel",
			Internal: true,
		},
		DomainId:       c.domainId,
		TimeoutSec:     int64(timeout),
		Variables:      vars,
		ConversationId: c.id,
	})

	if err != nil {
		return model.NewAppError("Conversation.Bridge", "conv.bridge.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	<-c.chBridge

	fmt.Println(res.InviteId)

	return nil
}

func (c *conversation) DumpExportVariables() map[string]string {
	c.mx.RLock()
	defer c.mx.RUnlock()

	var res map[string]string
	if len(c.exportVariables) > 0 {
		res = make(map[string]string)
		for _, v := range c.exportVariables {
			res[v], _ = c.Get(v)
		}
	}
	return res
}

func (c *conversation) actualizeClient(cli *ChatClientConnection) {
	if cli.Name() != c.client.Name() {
		c.mx.Lock()
		wlog.Debug(fmt.Sprintf("conversation [%s] changed client from \"%s\" to \"%s\"", c.id, c.client.Name(), cli.Name()))
		c.client = cli
		c.mx.Unlock()
	}
}
