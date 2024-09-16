package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	proto "buf.build/gen/go/webitel/chat/protocolbuffers/go"
	"github.com/tidwall/gjson"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

var (
	ErrWaitMessageTimeout = model.NewAppError("Conversation.WaitMessage", "conv.timeout.msg", nil, "Timeout", http.StatusInternalServerError)
)

type conversation struct {
	id            string
	profileId     int64
	schemaId      int32
	domainId      int64
	variables     *model.ThreadSafeStringMap
	client        *ChatClientConnection
	mx            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	storeMessages map[string][]byte
	chBridge      chan struct{}
	breakCause    string

	confirmation    map[string]chan []*proto.Message
	exportVariables []string
	nodeId          string
	userId          int64
	queueKey        *model.InQueueKey
	messages        []*proto.Message

	chat *chatApi

	log *wlog.Logger
}

func NewConversation(cli *ChatClientConnection, id string, domainId, profileId int64, schemaId int32, userId int64) *conversation {
	ctx, cancel := context.WithCancel(context.Background())
	return &conversation{
		id:            id,
		profileId:     profileId,
		schemaId:      schemaId,
		domainId:      domainId,
		variables:     model.NewThreadSafeStringMap(),
		client:        cli,
		chBridge:      nil,
		mx:            sync.RWMutex{},
		ctx:           ctx,
		cancel:        cancel,
		userId:        userId,
		storeMessages: make(map[string][]byte),
		confirmation:  make(map[string]chan []*proto.Message),
		nodeId:        cli.Name(),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("conversation_id", id),
			wlog.Int64("domain_id", domainId),
			wlog.Int64("bot_id", profileId),
			wlog.Int("schema_id", int(schemaId)),
		),
	}
}

func (c *conversation) Log() *wlog.Logger {
	return c.log
}

func (c *conversation) UserId() int64 {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.userId
}

func (c *conversation) SchemaId() int32 {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.schemaId
}

func (c *conversation) BreakCause() string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.breakCause
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
	idx := strings.Index(name, ".")
	if idx > 0 {
		nameRoot := name[0:idx]
		if m, ok := c.storeMessages[nameRoot]; ok {
			return gjson.GetBytes(m, name[idx+1:]).String(), true
		}

		if v, ok := c.variables.Load(nameRoot); ok {
			return gjson.GetBytes([]byte(v), name[idx+1:]).String(), true
		}
	}
	v, ok := c.variables.Load(name)
	return v, ok
}

func (c *conversation) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	for k, v := range vars {
		c.variables.Store(k, fmt.Sprintf("%v", v))
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

func (c *conversation) closeIfBreak() {
	if c.chBridge != nil {
		close(c.chBridge) // todo move to fn
		c.chBridge = nil
	}
}

func (c *conversation) IsTransfer() bool {
	if c.breakCause == "" {
		return false
	}
	return strings.EqualFold(c.breakCause, model.BreakChatTransferCause)
}

func (c *conversation) Break(cause string) *model.AppError {
	c.mx.Lock()
	c.closeIfBreak()
	c.breakCause = cause
	c.mx.Unlock()

	c.setTransferVariable()

	c.cancel()
	return nil
}

func (c *conversation) setTransferVariable() {
	if c.IsTransfer() {
		vars := c.DumpExportVariables()
		if vars == nil {
			vars = make(map[string]string)
		}
		vars["chat_transferred"] = "true"
		c.client.api.SetVariables(context.TODO(), &proto.SetVariablesRequest{
			ChannelId: c.id,
			Variables: vars,
		})
		c.Set(context.TODO(), model.Variables{
			"chat_transferred": "false",
		})
	} else {
		c.Set(context.TODO(), model.Variables{
			"chat_transferred": "false",
		})
	}
}

func (c *conversation) ProfileId() int64 {
	return c.profileId
}

func (c *conversation) SendMessage(ctx context.Context, msg model.ChatMessageOutbound) (model.Response, *model.AppError) {
	err := c.sendMessage(ctx, &proto.SendMessageRequest{
		ConversationId: c.id,
		Message: &proto.Message{
			Type:    msg.Type,
			Text:    msg.Text,
			Buttons: getChatButtons(msg.Buttons),
			Inline:  getChatButtons(msg.Inline),
			File:    getFile(msg.File),
			NoInput: msg.NoInput,
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendMessage", "conv.send.any.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) SendTextMessage(ctx context.Context, text string) (model.Response, *model.AppError) {
	err := c.sendMessage(ctx, &proto.SendMessageRequest{
		ConversationId: c.id,
		Message: &proto.Message{
			Type: "text", // FIXME
			Text: text,
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendTextMessage", "conv.send.text.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) sendMessage(ctx context.Context, req *proto.SendMessageRequest) error {
	_, err := c.client.api.SendMessage(ctx, req)
	if err != nil {
		textErr := err.Error()
		if strings.Index(textErr, `"id":"chat.send.channel.from.closed"`) != -1 {
			c.client.api.CloseConversation(c.ctx, &proto.CloseConversationRequest{
				ConversationId: c.id,
				Cause:          proto.CloseConversationCause_flow_err,
			})
			c.Break("error")
		}

		return model.NewAppError("Conversation.SendTextMessage", "conv.send_msg.app_err", nil, textErr, http.StatusInternalServerError)
	}

	c.saveMessages(req.Message)

	return nil
}

func (c *conversation) SendMenu(ctx context.Context, menu *model.ChatMenuArgs) (model.Response, *model.AppError) {
	req := &proto.Message{
		Type:    "text",
		Text:    menu.Text,
		Buttons: getChatButtons(menu.Buttons),
		NoInput: menu.NoInput,
	}
	//menu.Set // fixme

	err := c.sendMessage(ctx, &proto.SendMessageRequest{
		Message:        req,
		ConversationId: c.Id(),
	})
	if err != nil {
		return nil, model.NewAppError("Conversation.SendMenu", "conv.send.menu.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil

}

func (c *conversation) SendImageMessage(ctx context.Context, url string, name string, text string) (model.Response, *model.AppError) {
	err := c.sendMessage(ctx, &proto.SendMessageRequest{
		ConversationId: c.id,
		Message: &proto.Message{
			Type: "file", // FIXME
			Text: text,
			File: &proto.File{
				Url:  url,
				Name: name,
			},
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendImageMessage", "conv.send.image.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) SendFile(ctx context.Context, text string, f *model.File) (model.Response, *model.AppError) {
	err := c.sendMessage(ctx, &proto.SendMessageRequest{
		ConversationId: c.id,
		Message: &proto.Message{
			Type: "file", // FIXME
			Text: text,
			File: getFile(f),
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendFile", "conv.send.file.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) proto(ctx context.Context, url string, name string, text string) (model.Response, *model.AppError) {
	err := c.sendMessage(ctx, &proto.SendMessageRequest{
		ConversationId: c.id,
		Message: &proto.Message{
			Text: text,
			Type: "file", // FIXME
			File: &proto.File{
				Id:   1, //TODO
				Url:  url,
				Name: name,
			},
		},
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.SendTextMessage", "conv.send.text.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) addConfirmationId(id string, ch chan []*proto.Message) {
	c.mx.Lock()
	c.confirmation[id] = ch
	c.mx.Unlock()
}

func (c *conversation) deleteConfirmationId(id string) {
	c.mx.Lock()
	delete(c.confirmation, id)
	c.mx.Unlock()
}

func (c *conversation) saveMessages(msgs ...*proto.Message) {
	c.mx.Lock()
	c.messages = append(c.messages, msgs...)
	c.mx.Unlock()
}

func (c *conversation) countMessages() int {
	c.mx.RLock()
	cnt := len(c.messages)
	c.mx.RUnlock()
	return cnt
}

func (c *conversation) LastMessages(limit int) []model.ChatMessage {
	cnt := c.countMessages()
	if cnt == 0 {
		return nil
	}

	if cnt < limit {
		limit = cnt
	}
	res := make([]model.ChatMessage, 0, limit)
	c.mx.Lock()
	for _, v := range c.messages[(cnt - limit):] {
		res = append(res, pettyMessage(v))
	}
	c.mx.Unlock()
	return res
}

func (c *conversation) ReceiveMessage(ctx context.Context, name string, timeout int, messageTimeout int) ([]string, *model.AppError) {
	msgs, err := c.receive(ctx, timeout)
	if err != nil {
		return nil, err
	}

	if messageTimeout > 0 {
		var m []*proto.Message
		for err == nil {
			m, err = c.receive(ctx, messageTimeout)
			msgs = append(msgs, m...)
		}
	}
	if len(msgs) > 0 && name != "" {
		c.storeMessages[name], _ = json.Marshal(msgs[0])
	}
	c.saveMessages(msgs...)
	return messageToText(msgs...), nil
}

func (c *conversation) receive(ctx context.Context, timeout int) ([]*proto.Message, *model.AppError) {
	id := model.NewId()

	ch := make(chan []*proto.Message)
	c.addConfirmationId(id, ch)
	defer c.deleteConfirmationId(id)

	// TODO rename server api
	res, err := c.client.api.WaitMessage(ctx, &proto.WaitMessageRequest{
		ConversationId: c.id,
		ConfirmationId: id,
	})

	if err != nil {
		return nil, model.NewAppError("Conversation.WaitMessage", "conv.wait.msg.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	if len(res.Messages) > 0 {
		return res.Messages, nil
	}

	if timeout == 0 {
		timeout = int(res.TimeoutSec)
	}

	t := time.After(time.Second * time.Duration(timeout))

	c.log.Debug("wait message", wlog.Duration("wait_sec", time.Second*time.Duration(timeout)))

	select {
	case <-c.Context().Done():
		c.log.Debug("cancel")
		return nil, model.NewAppError("Conversation.WaitMessage", "conv.timeout.msg.app_err", nil, "Cancel", http.StatusInternalServerError)
	case <-t:
		c.log.Debug("timeout")
		return nil, ErrWaitMessageTimeout
	case msgsProto := <-ch:
		c.log.Debug(fmt.Sprintf("receive message: %v", msgsProto))
		return msgsProto, nil
	}
}

func (c *conversation) NodeName() string {
	return c.NodeId()
}

func (c *conversation) Stop(err *model.AppError, cause proto.CloseConversationCause) {
	if err != nil {
		c.log.Err(err)
		cause = proto.CloseConversationCause_flow_err
	}

	_, e := c.client.api.CloseConversation(c.ctx, &proto.CloseConversationRequest{
		ConversationId: c.id,
		Cause:          cause,
	})

	if e != nil {
		c.log.Err(e)
	}

	c.chat.conversations.Remove(c.id)
	c.log.Debug("close")
}

// TODO transferVars
func (c *conversation) Export(ctx context.Context, vars []string) (model.Response, *model.AppError) {
	exp := make(map[string]interface{})
	transferVars := make(map[string]string)
	for _, v := range vars {
		exp[fmt.Sprintf("usr_%s", v)], _ = c.Get(v)
		transferVars[v], _ = c.Get(v)
		c.exportVariables = append(c.exportVariables, v)
	}

	if len(exp) > 0 {
		if c.BreakCause() == "" {
			_, err := c.client.api.SetVariables(ctx, &proto.SetVariablesRequest{
				ChannelId: c.id,
				Variables: transferVars,
			})

			if err != nil {
				c.log.Warn(fmt.Sprintf("set variables error: %s", err.Error()))
			}
		}
		return c.Set(ctx, exp)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) UnSet(ctx context.Context, varKeys []string) (model.Response, *model.AppError) {
	vars := model.Variables{}
	req := &proto.SetVariablesRequest{
		ChannelId: c.id,
		Variables: make(map[string]string),
	}

	for _, v := range varKeys {
		// TODO
		vars[v] = ""
		req.Variables[v] = ""
	}

	_, err := c.client.api.SetVariables(ctx, req)

	if err != nil {
		c.log.Warn(fmt.Sprintf("set variables error: %s", err.Error()))
	}

	return c.Set(ctx, vars)
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

	res, err := c.client.api.InviteToConversation(ctx, &proto.InviteToConversationRequest{
		User: &proto.User{
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

func (c *conversation) Variables() map[string]string {
	return c.variables.Data()
}

func (c *conversation) SetQueue(key *model.InQueueKey) bool {
	c.mx.Lock()
	defer c.mx.Unlock()

	if c.queueKey == key {
		return false
	}

	c.queueKey = key
	return true
}

func (c *conversation) GetQueueKey() *model.InQueueKey {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.queueKey
}

func (c *conversation) actualizeClient(cli *ChatClientConnection) {
	if cli.Name() != c.client.Name() {
		c.mx.Lock()
		c.log.Debug(fmt.Sprintf("changed client from \"%s\" to \"%s\"", c.client.Name(), cli.Name()))
		c.client = cli
		c.mx.Unlock()
	}
}

func getChatButtons(buttons [][]model.ChatButton) []*proto.Buttons {
	l := len(buttons)

	if l == 0 {
		return nil
	}

	res := make([]*proto.Buttons, 0, l)

	for _, v := range buttons {
		btns := make([]*proto.Button, 0, len(v))
		for _, b := range v {
			btns = append(btns, &proto.Button{
				Text: b.Text,
				Type: b.Type,
				Url:  b.Url,
				Code: b.Code,
			})
		}

		res = append(res, &proto.Buttons{
			Button: btns,
		})
	}

	return res
}

func getFile(f *model.File) *proto.File {
	if f == nil {
		return nil
	}

	return &proto.File{
		Id:   int64(f.Id), //TODO
		Url:  f.Url,
		Mime: f.MimeType,
		Name: f.Name,
		Size: f.Size,
	}
}
