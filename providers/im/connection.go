package im

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/model"
)

var _ model.IMDialog = (*Connection)(nil)

type Connection struct {
	id       string
	threadId string
	ctx      context.Context
	domainId int64
	schemaId int
	srv      *server
	sync.RWMutex
	variables       map[string]string
	msg             model.Message
	lastMsg         model.Message
	from            model.ImEndpoint
	to              model.ImEndpoint
	log             *wlog.Logger
	hdrs            metadata.MD
	queueKey        *model.InQueueKey
	exportVariables []string
	messages        []model.MessageWrapper

	inboundHandlers map[uint64]func(text string)
	nextHandlerID   uint64
}

func newConnection(s *server, id string, to model.ImEndpoint, msg model.MessageWrapper) *Connection {
	schemaId, _ := strconv.Atoi(to.Sub)

	conn := &Connection{
		id:        id,
		threadId:  msg.Message.ThreadID,
		srv:       s,
		from:      msg.Message.From,
		to:        to,
		lastMsg:   msg.Message,
		msg:       msg.Message,
		ctx:       context.Background(),
		domainId:  msg.DomainID,
		schemaId:  schemaId,
		variables: make(map[string]string),
		hdrs: metadata.New(map[string]string{
			"x-webitel-type":   "schema",
			"x-webitel-schema": fmt.Sprintf("%d.%d", msg.DomainID, schemaId),
		}),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "im"),
			wlog.String("id", id),
			wlog.String("thread_id", msg.Message.ThreadID),
			wlog.Int64("domain_id", msg.DomainID),
			wlog.Int("schema_id", schemaId),
		),
	}

	conn.variables[model.ConversationStartMessageVariable] = msg.Message.Text
	return conn
}

// OnMessage delivers an inbound message to all registered handlers.
func (c *Connection) OnMessage(msg model.MessageWrapper) {
	if msg.Message.From.Sub == c.to.Sub {
		c.log.Debug("message from sub changed")
		return
	}

	c.Lock()
	c.messages = append(c.messages, msg)
	c.lastMsg = msg.Message
	var handlers []func(text string)
	for _, fn := range c.inboundHandlers {
		handlers = append(handlers, fn)
	}
	c.Unlock()

	for _, fn := range handlers {
		fn(msg.Message.Text)
	}

	if len(handlers) == 0 {
		c.log.Debug("message "+msg.Message.Text, wlog.String("thread_id", msg.Message.ThreadID))
	}
}

// OnInboundMessage registers handler to be called when the remote end sends a
// message. The handler must not block. Returns an unregister function that
// must be called exactly once.
func (c *Connection) OnInboundMessage(handler func(text string)) (unregister func()) {
	c.Lock()
	if c.inboundHandlers == nil {
		c.inboundHandlers = make(map[uint64]func(text string))
	}
	id := c.nextHandlerID
	c.nextHandlerID++
	c.inboundHandlers[id] = handler
	c.Unlock()

	return func() {
		c.Lock()
		delete(c.inboundHandlers, id)
		c.Unlock()
	}
}

func (c *Connection) From() model.ImEndpoint {
	return c.from
}

func (c *Connection) To() model.ImEndpoint {
	return c.to
}

func (c *Connection) ThreadId() string {
	return c.threadId
}

func (c *Connection) LastMessage() model.Message {
	return c.lastMsg
}

func (c *Connection) SendMessage(ctx context.Context, msg model.ChatMessageOutbound) (model.Response, *model.AppError) {
	var docs []*p.ImageInput

	if msg.File != nil {
		f := msg.File
		docs = append(docs, &p.ImageInput{
			Name:     f.Name,
			Link:     f.Url,
			MimeType: f.MimeType,
		})
	}

	_, err := c.srv.client.API.SendImage(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendImageRequest{
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
	_, err := c.srv.client.API.SendText(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendTextRequest{
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
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendTextMessage", "conv.msg", nil, err.Error(), http.StatusInternalServerError)
	}
	return model.CallResponseOK, nil
}

func (c *Connection) SendImageMessage(ctx context.Context, msg model.ChatMessageOutbound) (model.Response, *model.AppError) {
	var images []*p.ImageInput
	if msg.File != nil {
		f := msg.File
		images = append(images, &p.ImageInput{
			Id:       strconv.Itoa(f.Id),
			Name:     f.Name,
			Link:     f.Url,
			MimeType: f.MimeType,
		})
	}
	_, err := c.srv.client.API.SendImage(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendImageRequest{
		To: &p.Peer{Kind: &p.Peer_Contact{Contact: &p.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Image: &p.ImageRequest{Images: images, Body: msg.Text},
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendImageMessage", "conv.msg", nil, err.Error(), http.StatusInternalServerError)
	}
	return model.CallResponseOK, nil
}

func (c *Connection) SendDocumentMessage(ctx context.Context, msg model.ChatMessageOutbound) (model.Response, *model.AppError) {
	var docs []*p.DocumentInput
	if msg.File != nil {
		f := msg.File
		docs = append(docs, &p.DocumentInput{
			FileName:  f.Name,
			Url:       f.Url,
			MimeType:  f.MimeType,
			SizeBytes: &f.Size,
		})
	}
	_, err := c.srv.client.API.SendDocument(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendDocumentRequest{
		To: &p.Peer{Kind: &p.Peer_Contact{Contact: &p.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Document: &p.DocumentRequest{Documents: docs, Body: msg.Text},
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendDocumentMessage", "conv.msg", nil, err.Error(), http.StatusInternalServerError)
	}
	return model.CallResponseOK, nil
}

func (c *Connection) SendFile(ctx context.Context, text string, f *model.File, kind string) (model.Response, *model.AppError) {
	var docs []*p.DocumentInput
	if f != nil {
		docs = append(docs, &p.DocumentInput{
			FileName:  f.Name,
			Url:       f.Url,
			MimeType:  f.MimeType,
			SizeBytes: &f.Size,
		})
	}
	_, err := c.srv.client.API.SendDocument(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendDocumentRequest{
		To: &p.Peer{Kind: &p.Peer_Contact{Contact: &p.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Document: &p.DocumentRequest{Documents: docs, Body: text},
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendFile", "conv.msg", nil, err.Error(), http.StatusInternalServerError)
	}
	return model.CallResponseOK, nil
}

func (c *Connection) SendMenu(ctx context.Context, menu *model.ChatMenuArgs) (model.Response, *model.AppError) {
	rows := buildKeyboardRows(menu.Buttons)
	if menu.Type == "inline" {
		rows = buildKeyboardRows(menu.Inline)
	}

	_, err := c.srv.client.API.SendInteractive(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendInteractiveMessageRequest{
		To: &p.Peer{Kind: &p.Peer_Contact{Contact: &p.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Body: &menu.Text,
		Interactive: &p.Interactive{
			Kind: &p.Interactive_Markup{
				Markup: &p.KeyboardMarkup{Rows: rows},
			},
		},
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("Connection.SendMenu", "conv.send.menu.app_err", nil, err.Error(), http.StatusInternalServerError)
	}
	return model.CallResponseOK, nil
}

func buildKeyboardRows(src [][]model.ChatButton) []*p.KeyboardRow {
	rows := make([]*p.KeyboardRow, 0, len(src))
	for _, row := range src {
		buttons := make([]*p.KeyboardButton, 0, len(row))
		for _, btn := range row {
			kb := &p.KeyboardButton{Label: btn.Caption}
			switch {
			case btn.Type == "url":
				kb.Kind = &p.KeyboardButton_Url{Url: &p.KeyboardButtonURL{Url: btn.Url}}
			case btn.Code != "":
				kb.Kind = &p.KeyboardButton_Callback{Callback: &p.KeyboardButtonCallback{Data: btn.Code}}
			default:
				kb.Kind = &p.KeyboardButton_Callback{Callback: &p.KeyboardButtonCallback{Data: btn.Text}}
			}
			buttons = append(buttons, kb)
		}
		rows = append(rows, &p.KeyboardRow{Buttons: buttons})
	}
	return rows
}

func (c *Connection) Export(ctx context.Context, vars []string) (model.Response, *model.AppError) {
	exp := make(map[string]any)
	for _, v := range vars {
		tmp, _ := c.Get(v)
		tmp = strings.ToValidUTF8(tmp, "")
		exp[fmt.Sprintf("usr_%s", v)] = tmp
	}

	c.Lock()
	c.exportVariables = append(c.exportVariables, vars...)
	c.Unlock()

	if len(exp) > 0 {
		return c.Set(ctx, exp)
	}

	return model.CallResponseOK, nil
}

func (c *Connection) UnSet(_ context.Context, varKeys []string) (model.Response, *model.AppError) {
	c.Lock()
	defer c.Unlock()
	for _, k := range varKeys {
		delete(c.variables, k)
	}
	return model.CallResponseOK, nil
}

func (c *Connection) LastMessages(limit int) []model.ChatMessage {
	c.RLock()
	msgs := c.messages
	c.RUnlock()

	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	result := make([]model.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, model.ChatMessage{Text: m.Message.Text})
	}
	return result
}

func (c *Connection) GetQueueKey() *model.InQueueKey {
	c.RLock()
	defer c.RUnlock()
	return c.queueKey
}

func (c *Connection) SetQueue(key *model.InQueueKey) bool {
	c.Lock()
	defer c.Unlock()

	if c.queueKey == key {
		return false
	}

	c.queueKey = key
	return true
}

func (c *Connection) DumpExportVariables() map[string]string {
	c.RLock()
	defer c.RUnlock()
	out := make(map[string]string, len(c.exportVariables))
	for _, k := range c.exportVariables {
		if v, ok := c.variables[k]; ok {
			out[k] = v
		}
	}
	return out
}

func (c *Connection) IsTransfer() bool {
	return false
}

func (c *Connection) Stop(err error) {
	c.srv.stopConnection(c)
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
		c.variables[k] = fmt.Sprintf("%v", v)
	}

	return model.CallResponseOK, nil
}

func (c *Connection) ParseText(text string, ops ...model.ParseOption) string {
	return model.ParseText(c, text, ops...)
}

func (c *Connection) Close() error {
	return nil
}

func (c *Connection) Variables() map[string]string {
	c.RLock()
	defer c.RUnlock()

	return maps.Clone(c.variables)
}
