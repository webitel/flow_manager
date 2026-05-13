package im

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/wlog"

	"github.com/webitel/flow_manager/api/gen/im/api/gateway/v1"
	calldomain "github.com/webitel/flow_manager/internal/domain/call"
	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
)

var _ chatdomain.IMDialog = (*Connection)(nil)

type Connection struct {
	id       string
	threadId string
	ctx      context.Context
	domainId int64
	schemaId int
	srv      *server
	sync.RWMutex
	variables       map[string]string
	msg             chatdomain.Message
	lastMsg         chatdomain.Message
	from            chatdomain.ImEndpoint
	to              chatdomain.ImEndpoint
	log             *wlog.Logger
	hdrs            metadata.MD
	queueKey        *queue.InQueueKey
	exportVariables []string
	messages        []chatdomain.MessageWrapper

	inboundHandlers map[uint64]func(text string)
	nextHandlerID   uint64
}

func newConnection(s *server, id string, to chatdomain.ImEndpoint, msg chatdomain.MessageWrapper) *Connection {
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

	conn.variables[chatdomain.ConversationStartMessageVariable] = msg.Message.Text
	return conn
}

// OnMessage delivers an inbound message to all registered handlers.
func (c *Connection) OnMessage(msg chatdomain.MessageWrapper) {
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

func (c *Connection) From() chatdomain.ImEndpoint {
	return c.from
}

func (c *Connection) To() chatdomain.ImEndpoint {
	return c.to
}

func (c *Connection) ThreadId() string {
	return c.threadId
}

func (c *Connection) LastMessage() chatdomain.Message {
	return c.lastMsg
}

func (c *Connection) SendMessage(ctx context.Context, msg chatdomain.ChatMessageOutbound) (flow.Response, error) {
	var docs []*thread.ImageInput

	if msg.File != nil {
		f := msg.File
		docs = append(docs, &thread.ImageInput{
			Name:     f.Name,
			Link:     f.Url,
			MimeType: f.MimeType,
		})
	}

	_, err := c.srv.client.API.SendImage(metadata.NewOutgoingContext(ctx, c.hdrs), &thread.SendImageRequest{
		To: &thread.Peer{
			Kind: &thread.Peer_Contact{
				Contact: &thread.PeerIdentity{
					Sub: c.msg.From.Sub,
					Iss: c.msg.From.Issuer,
				},
			},
		},
		Image: &thread.ImageRequest{
			Images: docs,
			Body:   msg.Text,
		},
	})
	if err != nil {
		return calldomain.CallResponseError, fmt.Errorf("SendMessage: conv.msg: %w", err)
	}

	return calldomain.CallResponseOK, nil
}

func (c *Connection) SendTextMessage(ctx context.Context, text string) (flow.Response, error) {
	_, err := c.srv.client.API.SendText(metadata.NewOutgoingContext(ctx, c.hdrs), &thread.SendTextRequest{
		To: &thread.Peer{
			Kind: &thread.Peer_Contact{
				Contact: &thread.PeerIdentity{
					Sub: c.msg.From.Sub,
					Iss: c.msg.From.Issuer,
				},
			},
		},
		Body: text,
	})
	if err != nil {
		return calldomain.CallResponseError, fmt.Errorf("SendTextMessage: conv.msg: %w", err)
	}
	return calldomain.CallResponseOK, nil
}

func (c *Connection) SendImageMessage(ctx context.Context, msg chatdomain.ChatMessageOutbound) (flow.Response, error) {
	var images []*thread.ImageInput
	if msg.File != nil {
		f := msg.File
		images = append(images, &thread.ImageInput{
			Id:       strconv.Itoa(f.Id),
			Name:     f.Name,
			Link:     f.Url,
			MimeType: f.MimeType,
		})
	}
	_, err := c.srv.client.API.SendImage(metadata.NewOutgoingContext(ctx, c.hdrs), &thread.SendImageRequest{
		To: &thread.Peer{Kind: &thread.Peer_Contact{Contact: &thread.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Image: &thread.ImageRequest{Images: images, Body: msg.Text},
	})
	if err != nil {
		return calldomain.CallResponseError, fmt.Errorf("SendImageMessage: conv.msg: %w", err)
	}
	return calldomain.CallResponseOK, nil
}

func (c *Connection) SendDocumentMessage(ctx context.Context, msg chatdomain.ChatMessageOutbound) (flow.Response, error) {
	var docs []*thread.DocumentInput
	if msg.File != nil {
		f := msg.File
		docs = append(docs, &thread.DocumentInput{
			FileName:  f.Name,
			Url:       f.Url,
			MimeType:  f.MimeType,
			SizeBytes: &f.Size,
		})
	}
	_, err := c.srv.client.API.SendDocument(metadata.NewOutgoingContext(ctx, c.hdrs), &thread.SendDocumentRequest{
		To: &thread.Peer{Kind: &thread.Peer_Contact{Contact: &thread.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Document: &thread.DocumentRequest{Documents: docs, Body: msg.Text},
	})
	if err != nil {
		return calldomain.CallResponseError, fmt.Errorf("SendDocumentMessage: conv.msg: %w", err)
	}
	return calldomain.CallResponseOK, nil
}

func (c *Connection) SendFile(ctx context.Context, text string, f *files.File, kind string) (flow.Response, error) {
	var docs []*thread.DocumentInput
	if f != nil {
		docs = append(docs, &thread.DocumentInput{
			FileName:  f.Name,
			Url:       f.Url,
			MimeType:  f.MimeType,
			SizeBytes: &f.Size,
		})
	}
	_, err := c.srv.client.API.SendDocument(metadata.NewOutgoingContext(ctx, c.hdrs), &thread.SendDocumentRequest{
		To: &thread.Peer{Kind: &thread.Peer_Contact{Contact: &thread.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Document: &thread.DocumentRequest{Documents: docs, Body: text},
	})
	if err != nil {
		return calldomain.CallResponseError, fmt.Errorf("SendFile: conv.msg: %w", err)
	}
	return calldomain.CallResponseOK, nil
}

func (c *Connection) SendMenu(ctx context.Context, menu *chatdomain.ChatMenuArgs) (flow.Response, error) {
	rows := buildKeyboardRows(menu.Buttons)
	if menu.Type == "inline" {
		rows = buildKeyboardRows(menu.Inline)
	}

	_, err := c.srv.client.API.SendInteractive(metadata.NewOutgoingContext(ctx, c.hdrs), &thread.SendInteractiveMessageRequest{
		To: &thread.Peer{Kind: &thread.Peer_Contact{Contact: &thread.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Body: &menu.Text,
		Interactive: &thread.Interactive{
			Kind: &thread.Interactive_Markup{
				Markup: &thread.KeyboardMarkup{Rows: rows},
			},
		},
	})
	if err != nil {
		return calldomain.CallResponseError, fmt.Errorf("Connection.SendMenu: conv.send.menu.app_err: %w", err)
	}
	return calldomain.CallResponseOK, nil
}

func buildKeyboardRows(src [][]chatdomain.ChatButton) []*thread.KeyboardRow {
	rows := make([]*thread.KeyboardRow, 0, len(src))
	for _, row := range src {
		buttons := make([]*thread.KeyboardButton, 0, len(row))
		for _, btn := range row {
			kb := &thread.KeyboardButton{Label: btn.Caption}
			switch {
			case btn.Type == "url":
				kb.Kind = &thread.KeyboardButton_Url{Url: &thread.KeyboardButtonURL{Url: btn.Url}}
			case btn.Code != "":
				kb.Kind = &thread.KeyboardButton_Callback{Callback: &thread.KeyboardButtonCallback{Data: btn.Code}}
			default:
				kb.Kind = &thread.KeyboardButton_Callback{Callback: &thread.KeyboardButtonCallback{Data: btn.Text}}
			}
			buttons = append(buttons, kb)
		}
		rows = append(rows, &thread.KeyboardRow{Buttons: buttons})
	}
	return rows
}

func (c *Connection) Export(ctx context.Context, vars []string) (flow.Response, error) {
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

	return calldomain.CallResponseOK, nil
}

func (c *Connection) UnSet(_ context.Context, varKeys []string) (flow.Response, error) {
	c.Lock()
	defer c.Unlock()
	for _, k := range varKeys {
		delete(c.variables, k)
	}
	return calldomain.CallResponseOK, nil
}

func (c *Connection) LastMessages(limit int) []chatdomain.ChatMessage {
	c.RLock()
	msgs := c.messages
	c.RUnlock()

	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	result := make([]chatdomain.ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, chatdomain.ChatMessage{Text: m.Message.Text})
	}
	return result
}

func (c *Connection) GetQueueKey() *queue.InQueueKey {
	c.RLock()
	defer c.RUnlock()
	return c.queueKey
}

func (c *Connection) SetQueue(key *queue.InQueueKey) bool {
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

func (c *Connection) Type() flow.ConnectionType {
	return flow.ConnectionTypeIM
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

func (c *Connection) Set(ctx context.Context, vars flow.Variables) (flow.Response, error) {
	c.Lock()
	defer c.Unlock()

	for k, v := range vars {
		c.variables[k] = fmt.Sprintf("%v", v)
	}

	return calldomain.CallResponseOK, nil
}

func (c *Connection) ParseText(text string, ops ...flow.ParseOption) string {
	return flow.ParseText(c, text, ops...)
}

func (c *Connection) Close() error {
	return nil
}

func (c *Connection) Variables() map[string]string {
	c.RLock()
	defer c.RUnlock()

	return maps.Clone(c.variables)
}
