package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/webitel/wlog"

	ai_bots2 "github.com/webitel/flow_manager/api/gen/ai_bots"
	chat2 "github.com/webitel/flow_manager/api/gen/chat"
	apperrs "github.com/webitel/flow_manager/internal/infrastructure/errors"
	"github.com/webitel/flow_manager/model"
)

var ErrWaitMessageTimeout = apperrs.New(http.StatusInternalServerError, "Conversation.WaitMessage: conv.timeout.msg: Timeout")

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

	confirmation    map[string]chan []*chat2.Message
	exportVariables []string
	nodeId          string
	userId          int64
	queueKey        *model.InQueueKey
	messages        []*chat2.Message

	inboundHandlers map[uint64]func(text string)
	nextHandlerID   uint64

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
		confirmation:  make(map[string]chan []*chat2.Message),
		nodeId:        cli.Name(),
		log: wlog.GlobalLogger().With(
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

func (c *conversation) Type() model.ConnectionType {
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

func (c *conversation) Set(ctx context.Context, vars model.Variables) (model.Response, error) {
	for k, v := range vars {
		c.variables.Store(k, fmt.Sprintf("%v", v))
	}
	return model.CallResponseOK, nil
}

func (c *conversation) ParseText(text string, ops ...model.ParseOption) string {
	return model.ParseText(c, text, ops...)
}

func (c *conversation) Close() error {
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

func (c *conversation) Break(cause string) error {
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
		c.client.api.SetVariables(context.TODO(), &chat2.SetVariablesRequest{
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

func (c *conversation) SendMessage(ctx context.Context, msg model.ChatMessageOutbound) (model.Response, error) {
	err := c.sendMessage(ctx, &chat2.SendMessageRequest{
		ConversationId: c.id,
		Message: &chat2.Message{
			Type:    msg.Type,
			Text:    msg.Text,
			Buttons: getChatButtons(msg.Buttons),
			Inline:  getChatButtons(msg.Inline),
			File:    getFile(msg.File),
			NoInput: msg.NoInput,
			Kind:    msg.Kind,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Conversation.SendMessage: conv.send.any.app_err: %w", err)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) SendTextMessage(ctx context.Context, text string) (model.Response, error) {
	err := c.sendMessage(ctx, &chat2.SendMessageRequest{
		ConversationId: c.id,
		Message: &chat2.Message{
			Type: "text", // FIXME
			Text: text,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Conversation.SendTextMessage: conv.send.text.app_err: %w", err)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) sendMessage(ctx context.Context, req *chat2.SendMessageRequest) error {
	_, err := c.client.api.SendMessage(ctx, req)
	if err != nil {
		textErr := err.Error()
		if strings.Index(textErr, `"id":"chat.send.channel.from.closed"`) != -1 {
			c.client.api.CloseConversation(c.ctx, &chat2.CloseConversationRequest{
				ConversationId: c.id,
				Cause:          chat2.CloseConversationCause_flow_err,
			})
			c.Break("error")
		}

		return fmt.Errorf("Conversation.SendTextMessage: conv.send_msg.app_err: %s", textErr)
	}

	c.saveMessages(req.Message)

	return nil
}

func (c *conversation) SendMenu(ctx context.Context, menu *model.ChatMenuArgs) (model.Response, error) {
	req := &chat2.Message{
		Type:    "text",
		Text:    menu.Text,
		Buttons: getChatButtons(menu.Buttons),
		NoInput: menu.NoInput,
		Kind:    menu.Kind,
	}
	// menu.Set // fixme

	err := c.sendMessage(ctx, &chat2.SendMessageRequest{
		Message:        req,
		ConversationId: c.Id(),
	})
	if err != nil {
		return nil, fmt.Errorf("Conversation.SendMenu: conv.send.menu.app_err: %w", err)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) SendImageMessage(ctx context.Context, url, name, text, kind string) (model.Response, error) {
	err := c.sendMessage(ctx, &chat2.SendMessageRequest{
		ConversationId: c.id,
		Message: &chat2.Message{
			Type: "file", // FIXME
			Text: text,
			Kind: kind,
			File: &chat2.File{
				Url:  url,
				Name: name,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Conversation.SendImageMessage: conv.send.image.app_err: %w", err)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) SendFile(ctx context.Context, text string, f *model.File, kind string) (model.Response, error) {
	err := c.sendMessage(ctx, &chat2.SendMessageRequest{
		ConversationId: c.id,
		Message: &chat2.Message{
			Type: "file", // FIXME
			Text: text,
			Kind: kind,
			File: getFile(f),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Conversation.SendFile: conv.send.file.app_err: %w", err)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) proto(ctx context.Context, url, name, text string) (model.Response, error) {
	err := c.sendMessage(ctx, &chat2.SendMessageRequest{
		ConversationId: c.id,
		Message: &chat2.Message{
			Text: text,
			Type: "file", // FIXME
			File: &chat2.File{
				Id:   1, // TODO
				Url:  url,
				Name: name,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Conversation.SendTextMessage: conv.send.text.app_err: %w", err)
	}

	return model.CallResponseOK, nil
}

func (c *conversation) addConfirmationId(id string, ch chan []*chat2.Message) {
	c.mx.Lock()
	c.confirmation[id] = ch
	c.mx.Unlock()
}

func (c *conversation) deleteConfirmationId(id string) {
	c.mx.Lock()
	delete(c.confirmation, id)
	c.mx.Unlock()
}

func (c *conversation) saveMessages(msgs ...*chat2.Message) {
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

func (c *conversation) ReceiveMessage(ctx context.Context, name string, timeout, messageTimeout int) ([]string, error) {
	msgs, err := c.receive(ctx, timeout)
	if err != nil {
		return nil, err
	}

	if messageTimeout > 0 {
		var m []*chat2.Message
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

func (c *conversation) receive(ctx context.Context, timeout int) ([]*chat2.Message, error) {
	id := model.NewId()

	ch := make(chan []*chat2.Message)
	c.addConfirmationId(id, ch)
	defer c.deleteConfirmationId(id)

	// TODO rename server api
	res, err := c.client.api.WaitMessage(ctx, &chat2.WaitMessageRequest{
		ConversationId: c.id,
		ConfirmationId: id,
	})
	if err != nil {
		return nil, fmt.Errorf("Conversation.WaitMessage: conv.wait.msg.app_err: %w", err)
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
		return nil, fmt.Errorf("Conversation.WaitMessage: conv.timeout.msg.app_err: Cancel")
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

func (c *conversation) Stop(err error, cause chat2.CloseConversationCause) {
	if err != nil {
		c.log.Err(err)
		cause = chat2.CloseConversationCause_flow_err
	}

	// When breakCause != "" - messages-srv initiated close via Break,
	// skip CloseConversation callback to avoid duplicate close.
	// Flow-originated causes (flow_end, flow_err) should still close.
	if c.breakCause == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, e := c.client.api.CloseConversation(ctx, &chat2.CloseConversationRequest{
			ConversationId: c.id,
			Cause:          cause,
		})

		if e != nil {
			c.log.Err(e)
		}
	}

	c.chat.conversations.Remove(c.id)
	c.log.Debug("close", wlog.String("cause", cause.String()), wlog.String("breakCause", c.breakCause))
}

// TODO transferVars
func (c *conversation) Export(ctx context.Context, vars []string) (model.Response, error) {
	exp := make(map[string]any)
	transferVars := make(map[string]string)
	for _, v := range vars {
		tmp, _ := c.Get(v)
		tmp = strings.ToValidUTF8(tmp, "")
		exp[fmt.Sprintf("usr_%s", v)] = tmp
		transferVars[v] = tmp
		c.exportVariables = append(c.exportVariables, v)
	}

	if len(exp) > 0 {
		if c.BreakCause() == "" {
			_, err := c.client.api.SetVariables(ctx, &chat2.SetVariablesRequest{
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

func (c *conversation) UnSet(ctx context.Context, varKeys []string) (model.Response, error) {
	vars := model.Variables{}
	req := &chat2.SetVariablesRequest{
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

func (c *conversation) Bridge(ctx context.Context, userId int64, timeout int) error {
	if c.chBridge != nil {
		return fmt.Errorf("Conversation.Bridge: conv.bridge.app_err: Not allow two bridge")
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

	res, err := c.client.api.InviteToConversation(ctx, &chat2.InviteToConversationRequest{
		User: &chat2.User{
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
		return fmt.Errorf("Conversation.Bridge: conv.bridge.app_err: %w", err)
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
			tmp, _ := c.Get(v)
			res[v] = strings.ToValidUTF8(tmp, "")
		}
	}
	return res
}

func (c *conversation) Variables() map[string]string {
	return c.variables.Data()
}

// StartWaiting signals the external chat server to forward the next inbound
// message by calling WaitMessage (required by the chat delivery protocol).
// When the message arrives it is delivered via fireInboundHandlers so the
// native recvMessage op (suspended via OnInboundMessage) can resume.
// Runs in a goroutine and uses the connection's own context so it survives
// past the flow suspension point.
func (c *conversation) StartWaiting(timeout int) {
	go func() {
		id := model.NewId()
		ch := make(chan []*chat2.Message, 1)
		c.addConfirmationId(id, ch)
		defer c.deleteConfirmationId(id)

		ctx := c.ctx
		res, err := c.client.api.WaitMessage(ctx, &chat2.WaitMessageRequest{
			ConversationId: c.id,
			ConfirmationId: id,
		})
		if err != nil {
			return
		}

		var msgs []*chat2.Message
		if len(res.Messages) > 0 {
			msgs = res.Messages
		} else {
			waitSec := int(res.TimeoutSec)
			if timeout > 0 {
				waitSec = timeout
			}
			if waitSec <= 0 {
				waitSec = 3600
			}
			t := time.After(time.Duration(waitSec) * time.Second)
			select {
			case <-ctx.Done():
				return
			case <-t:
				c.fireInboundHandlers("")
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				msgs = m
			}
		}

		text := strings.Join(messageToText(msgs...), " ")
		c.fireInboundHandlers(text)
	}()
}

// OnInboundMessage registers a handler called when the remote peer sends a
// message while the runtime has no active WaitMessage subscription (e.g.
// during a softSleep or a native recvMessage suspend). The handler must not
// block. Returns an unregister function that must be called exactly once.
func (c *conversation) OnInboundMessage(handler func(text string)) (unregister func()) {
	c.mx.Lock()
	if c.inboundHandlers == nil {
		c.inboundHandlers = make(map[uint64]func(text string))
	}
	id := c.nextHandlerID
	c.nextHandlerID++
	c.inboundHandlers[id] = handler
	c.mx.Unlock()
	return func() {
		c.mx.Lock()
		delete(c.inboundHandlers, id)
		c.mx.Unlock()
	}
}

// fireInboundHandlers delivers text to all registered OnInboundMessage handlers.
// Called by ConfirmationMessage when no legacy WaitMessage confirmation is active.
func (c *conversation) fireInboundHandlers(text string) {
	c.mx.RLock()
	handlers := make([]func(text string), 0, len(c.inboundHandlers))
	for _, fn := range c.inboundHandlers {
		handlers = append(handlers, fn)
	}
	c.mx.RUnlock()
	for _, fn := range handlers {
		fn(text)
	}
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

func (c *conversation) Bot(ctx context.Context, cli ai_bots2.ConverseServiceClient, id string) (model.Response, error) {
	var res *ai_bots2.ConverseResponse
	stream, err := cli.Converse(ctx)
	if err != nil {
		return model.CallResponseError, nil
	}

	defer stream.CloseSend()

	err = stream.Send(&ai_bots2.ConverseRequest{
		RequestType: &ai_bots2.ConverseRequest_Config{
			Config: &ai_bots2.Config{
				ConversationId: c.id,
				DialogId:       id,
				UserData:       nil,
				Rate:           "",
			},
		},
	})
	if err != nil {
		return model.CallResponseError, nil
	}

	go func() {
		for {
			msg, err := c.ReceiveMessage(ctx, "", 10000, 0)
			if err != nil {
				return
			}

			err2 := stream.Send(&ai_bots2.ConverseRequest{
				RequestType: &ai_bots2.ConverseRequest_Input{
					Input: &ai_bots2.Input{
						Data: &ai_bots2.Input_TextData{
							TextData: strings.Join(msg, "."),
						},
					},
				},
			})
			if err2 != nil {
				return
			}
		}
	}()

	for {
		res, err = stream.Recv()
		if err != nil {
			return model.CallResponseError, nil
		}

		if res.TextData != "" {
			res.TextData = strings.Replace(res.TextData, "**", "*", -1)
			_, err2 := c.SendTextMessage(ctx, res.TextData)
			if err2 != nil {
				c.log.Error(err2.Error())
			}
		}

		if res.StopTalk {
			// break
		}
	}

	return model.CallResponseOK, nil
}

func (c *conversation) actualizeClient(cli *ChatClientConnection) {
	if cli.Name() != c.client.Name() {
		c.mx.Lock()
		c.log.Debug(fmt.Sprintf("changed client from \"%s\" to \"%s\"", c.client.Name(), cli.Name()))
		c.client = cli
		c.mx.Unlock()
	}
}

func getChatButtons(buttons [][]model.ChatButton) []*chat2.Buttons {
	l := len(buttons)

	if l == 0 {
		return nil
	}

	res := make([]*chat2.Buttons, 0, l)

	for _, v := range buttons {
		btns := make([]*chat2.Button, 0, len(v))
		for _, b := range v {
			btns = append(btns, &chat2.Button{
				Text: b.Text,
				Type: b.Type,
				Url:  b.Url,
				Code: b.Code,
			})
		}

		res = append(res, &chat2.Buttons{
			Button: btns,
		})
	}

	return res
}

func getFile(f *model.File) *chat2.File {
	if f == nil {
		return nil
	}

	return &chat2.File{
		Id:   int64(f.Id), // TODO
		Url:  f.Url,
		Mime: f.MimeType,
		Name: f.Name,
		Size: f.Size,
	}
}
