package im

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/model"
)

var ErrWaitMessageTimeout = model.NewAppError("Dialog.WaitMessage", "dialog.timeout.msg", nil, "Timeout", http.StatusInternalServerError)

var _ model.IMDialog = (*Connection)(nil)

type Connection struct {
	id        string
	ctx       context.Context
	domainId  int64
	schemaId  int
	srv       *server
	variables map[string]string
	msg       model.Message
	sync.RWMutex
	log         *wlog.Logger
	waitMsgChan chan model.MessageWrapper
	hdrs        metadata.MD
}

func newConnection(s *server, msg model.MessageWrapper) *Connection {
	id := msg.Message.ThreadID // todo
	schemaId := 2522
	conn := &Connection{
		id:  id,
		srv: s,
		hdrs: metadata.New(map[string]string{
			"x-webitel-type":   "schema",
			"x-webitel-schema": fmt.Sprintf("%d.%d", msg.DomainID, schemaId),
		}),
		msg:       msg.Message,
		ctx:       context.Background(),
		domainId:  msg.DomainID,
		variables: toVariables(nil), // todo
		schemaId:  schemaId,
		RWMutex:   sync.RWMutex{},
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "im"),
			wlog.String("id", id),
			wlog.String("thread_id", msg.Message.ThreadID),
			wlog.Int64("domain_id", msg.DomainID),
			wlog.Int("schema_id", schemaId),
		),
	}
	if conn.variables == nil {
		conn.variables = make(map[string]string)
	}

	conn.variables[model.ConversationStartMessageVariable] = msg.Message.Text
	return conn
}

func (c *Connection) OnMessage(msg model.MessageWrapper) {
	if msg.Message.From.Sub == c.msg.To.Sub {
		c.log.Debug("message from sub changed")
		return
	}

	c.Lock()
	ch := c.waitMsgChan
	c.Unlock()
	if ch != nil { // todo skip flow messages
		ch <- msg
		return
	}
	c.log.Debug("message "+msg.Message.Text, wlog.String("thread_id", msg.Message.ThreadID))
}

func (c *Connection) setStateWaitMessage(ch chan model.MessageWrapper) error {
	c.Lock()
	defer c.Unlock()

	if c.waitMsgChan != nil && ch != nil {
		return errors.New("already set wait message chan")
	}

	c.waitMsgChan = ch
	return nil
}

func (c *Connection) SendMessage(ctx context.Context, msg model.ChatMessageOutbound) (model.Response, *model.AppError) {
	var docs []*p.ImageInput

	if msg.File != nil {
		f := msg.File
		docs = append(docs, &p.ImageInput{
			// Id:       strconv.Itoa(f.Id),
			Name:     f.Name,
			Link:     f.Url,
			MimeType: f.MimeType,
		})
	}

	_, err := c.srv.client.Api.SendImage(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendImageRequest{
		To: &p.Peer{
			Kind: &p.Peer_Contact{
				Contact: &p.PeerIdentity{
					Sub: c.msg.From.Sub,
					Iss: c.msg.From.Issuer,
				},
			},
		},
		Image: &p.ImageRequest{
			Images: docs,
			Body:   msg.Text,
		},
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendMessage", "conv.msg", nil, err.Error(), http.StatusInternalServerError)
	}

	return model.CallResponseOK, nil
}

func (c *Connection) SendTextMessage(ctx context.Context, text string) (model.Response, *model.AppError) {
	_, err := c.srv.client.Api.SendText(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendTextRequest{
		To: &p.Peer{
			Kind: &p.Peer_Contact{
				Contact: &p.PeerIdentity{
					Sub: c.msg.From.Sub,
					Iss: c.msg.From.Issuer,
				},
			},
		},

		Body: text,
	})
	// println(res)
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendTextMessage", "conv.msg", nil, err.Error(), http.StatusInternalServerError)
	}
	return model.CallResponseOK, nil
}

func (c *Connection) ReceiveMessage(ctx context.Context, name string, timeout, messageTimeout int) ([]string, *model.AppError) {
	msgs, err := c.receive(ctx, timeout)
	if err != nil {
		return nil, err
	}

	if messageTimeout > 0 {
		var m []model.MessageWrapper
		for err == nil {
			m, err = c.receive(ctx, messageTimeout)
			msgs = append(msgs, m...)
		}
	}
	//if len(msgs) > 0 && name != "" {
	//	c.storeMessages[name], _ = json.Marshal(msgs[0])
	//}
	//c.saveMessages(msgs...)
	return messageToText(msgs...), nil
}

func (c *Connection) IsTransfer() bool {
	return false
}

func (c *Connection) Stop(err error) {
	c.srv.stopConnection(c)
}

func (c *Connection) receive(ctx context.Context, timeout int) ([]model.MessageWrapper, *model.AppError) {
	ch := make(chan model.MessageWrapper)
	defer func() {
		c.setStateWaitMessage(nil)
	}()

	err := c.setStateWaitMessage(ch)
	if err != nil {
		c.log.Warn("Failed to set wait message chan")
		return nil, model.NewAppError("Conversation.WaitMessage", "conv.timeout.msg", nil, "Timeout", http.StatusInternalServerError)
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
		return []model.MessageWrapper{msgsProto}, nil
	}
}

func (c *Connection) Type() model.ConnectionType {
	return model.ConnectionTypeIM
}

func (c *Connection) Log() *wlog.Logger {
	return c.log
}

func (c *Connection) Id() string {
	return c.id
}

func (c *Connection) SchemaId() int {
	return c.schemaId
}

func (c *Connection) NodeId() string {
	return ""
}

func (c *Connection) DomainId() int64 {
	return c.domainId
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) Get(key string) (string, bool) {
	c.RLock()
	defer c.RUnlock()

	idx := strings.Index(key, ".")
	if idx > 0 {
		nameRoot := key[0:idx]

		if v, ok := c.variables[nameRoot]; ok {
			return gjson.GetBytes([]byte(v), key[idx+1:]).String(), true
		}
	}
	v, ok := c.variables[key]
	return v, ok
}

func (c *Connection) Set(ctx context.Context, vars model.Variables) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v) // TODO
	}

	return model.CallResponseOK, nil
}

func (c *Connection) ParseText(text string, ops ...model.ParseOption) string {
	return model.ParseText(c, text, ops...)
}

func (c *Connection) Close() *model.AppError {
	return nil
}

func (c *Connection) Variables() map[string]string {
	c.RLock()
	defer c.RUnlock()

	return maps.Clone(c.variables)
}

func toVariables(in map[string]json.RawMessage) map[string]string {
	vars := make(map[string]string)

	for k, v := range in {
		vars[k] = string(v)
	}

	return vars
}

func messageToText(messages ...model.MessageWrapper) []string {
	msgs := make([]string, 0, len(messages))

	for _, m := range messages {
		msgs = append(msgs, m.Message.Text)
	}

	return msgs
}
