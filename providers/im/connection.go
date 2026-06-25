package im

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/webitel/wlog"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/model"
)

var ErrWaitMessageTimeout = model.NewAppError("Dialog.WaitMessage", "dialog.timeout.msg", nil, "Timeout", http.StatusRequestTimeout)

var _ model.IMDialog = (*Connection)(nil)

type Connection struct {
	sync.RWMutex

	id       string
	threadId string
	ctx      context.Context
	domainId int64
	schemaId int
	srv      *server

	clientDeviceIDLock *sync.RWMutex
	clientDeviceID     string

	variables       map[string]string
	msg             model.Message
	lastMsg         model.Message
	from            model.ImEndpoint
	to              model.ImEndpoint
	log             *wlog.Logger
	waitMsgChan     chan model.IMEventWrapper
	hdrs            metadata.MD
	queueKey        *model.InQueueKey
	exportVariables []string
	messages        []model.IMEventWrapper
	info            model.ThreadInfo
}

func newConnection(s *server, id string, to model.ImEndpoint, msg model.IMEventWrapper) *Connection {
	schemaId, _ := strconv.Atoi(to.Sub)

	conn := &Connection{
		id:       id,
		threadId: msg.GetPayload().GetThreadID(),
		srv:      s,
		from:     msg.GetPayload().Sender(),
		to:       to,
		lastMsg:  msg.GetPayload().Message(),
		hdrs: metadata.New(map[string]string{
			"x-webitel-type":   "schema",
			"x-webitel-schema": fmt.Sprintf("%d.%d", msg.GetDomainID(), schemaId),
		}),
		msg:                msg.GetPayload().Message(),
		ctx:                context.Background(),
		domainId:           msg.GetDomainID(),
		variables:          toVariables(nil), // todo
		schemaId:           schemaId,
		RWMutex:            sync.RWMutex{},
		clientDeviceIDLock: new(sync.RWMutex),
		log: wlog.GlobalLogger().With(
			wlog.Namespace("context"),
			wlog.String("scope", "im"),
			wlog.String("id", id),
			wlog.String("thread_id", msg.GetPayload().GetThreadID()),
			wlog.Int64("domain_id", msg.GetDomainID()),
			wlog.Int("schema_id", schemaId),
		),
	}
	if conn.variables == nil {
		conn.variables = make(map[string]string)
	}

	if msg.JWTPayload() != "" {
		conn.variables[JWTPayloadVar] = msg.JWTPayload()
	}

	if msg.DeviceID() != "" { //first connection touch
		conn.clientDeviceID = msg.DeviceID()
	}

	return conn
}

func (c *Connection) IMOugoingContext(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, c.hdrs)
}

func (c *Connection) setDeviceID(device string) {
	if device == "" {
		return
	}

	c.clientDeviceIDLock.Lock()
	c.clientDeviceID = device
	c.clientDeviceIDLock.Unlock()
}

func (c *Connection) DeviceID() string {
	c.clientDeviceIDLock.RLock()
	defer c.clientDeviceIDLock.RUnlock()

	return c.clientDeviceID
}

func (c *Connection) setupVariables() {
	c.variables[model.ConversationStartMessageVariable] = c.msg.Text

	c.variables["uuid"] = c.id
	info, err := c.treadInfo(c.srv.client.ctx)
	if err != nil {
		c.log.Error("failed to get thread info", wlog.Err(err))
		return
	}
	c.msg.Subject = info.Subject
	c.msg.Description = info.Description
	raw, _ := json.Marshal(c.msg)
	c.variables["thread"] = string(raw)
	maps.Copy(c.variables, info.Variables)
	c.info = info
}

func (c *Connection) processLastMessage(lastMessage model.IMEventWrapper) {
	if lastMessage.GetType() != model.IMEventTypeMessage {
		return
	}

	c.Lock()
	defer c.Unlock()

	c.messages = append(c.messages, lastMessage)
	c.lastMsg = lastMessage.GetPayload().Message()
}

func (c *Connection) processLastInteractiveCallback(callback model.IMEventWrapper) {
	if callback.GetType() != model.IMEventTypeCallback {
		return
	}

	assertedCallback, ok := callback.GetPayload().(model.InteractiveCallback)
	if !ok {
		c.log.Warn("unsuccessfull assert from wrapper to callback")
		return
	}

	serializedCallback, err := json.Marshal(assertedCallback)
	if err != nil {
		c.log.Error("serializing interactive callback for variable setting", wlog.Err(err))
		return
	}

	c.Lock()
	c.variables["clicked_button"] = string(serializedCallback)
	c.Unlock()
}

func (c *Connection) pushMessageToWaitMessageChan(message model.IMEventWrapper) {
	if message.GetPayload().Message().Sender().Issuer == "bot" {
		return
	}

	c.RLock()
	ch := c.waitMsgChan
	c.RUnlock()

	if ch != nil {
		ch <- message
	}
}

func (c *Connection) updateJWTPayloadVariable(msg model.IMEventWrapper) {
	if msg.JWTPayload() == "" {
		return
	}

	c.Lock()
	c.variables[JWTPayloadVar] = msg.JWTPayload()
	c.Unlock()
}

func (c *Connection) OnMessage(msg model.IMEventWrapper) {
	log := c.log.With(
		wlog.String("operation", "OnMessage"),
		wlog.String("from_sub", msg.GetPayload().Sender().Sub),
		wlog.String("to_sub", c.to.Sub),
		wlog.String("thread_id", msg.GetPayload().GetThreadID()),
		wlog.String("message_id", msg.GetPayload().MessageID()),
		wlog.Int64("domain_id", msg.GetDomainID()),
	)

	if msg.GetPayload().Sender().Sub == c.to.Sub {
		log.Debug("message from sub changed")
		return
	}

	c.processLastMessage(msg)
	c.processLastInteractiveCallback(msg)
	c.pushMessageToWaitMessageChan(msg)
	c.updateJWTPayloadVariable(msg)
	c.setDeviceID(msg.DeviceID())

	log.Debug("processed on message event")
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

func (c *Connection) setStateWaitMessage(ch chan model.IMEventWrapper) error {
	c.Lock()
	defer c.Unlock()

	if c.waitMsgChan != nil && ch != nil {
		return errors.New("already set wait message chan")
	}

	c.waitMsgChan = ch
	return nil
}

func (c *Connection) SendMessage(ctx context.Context, msg model.ChatMessageOutbound) (model.Response, *model.AppError) {
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

	_, err := c.srv.client.messageService.Api.SendDocument(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendDocumentRequest{
		To: &p.Peer{
			Kind: &p.Peer_Contact{
				Contact: &p.PeerIdentity{
					Sub: c.msg.From.Sub,
					Iss: c.msg.From.Issuer,
				},
			},
		},
		Documents: docs,
		Body:      msg.Text,
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendMessage", "conv.msg", nil, err.Error(), model.ExtractHTPPStatusCodeFromGRPC(err))
	}

	return model.CallResponseOK, nil
}

func (c *Connection) GetAuthSession(ctx context.Context, deviceID string) (model.IMUserInfo, *model.AppError) {
	response, err := c.srv.client.accountService.Api.AccountGetAuthorizations(
		c.IMOugoingContext(ctx),
		&p.AccountGetAuthorizationsRequest{
			DeviceId: deviceID,
			Size:     1,
		},
	)

	if err != nil {
		return model.IMUserInfo{}, model.NewAppError(
			"GetAuthSession",
			"providers.im.connection",
			nil,
			err.Error(),
			model.ExtractHTPPStatusCodeFromGRPC(err),
		)
	}

	if len(response.Items) == 0 {
		return model.IMUserInfo{}, model.ErrAuthSesionNotFound
	}

	responsedUserInfo := response.GetItems()[0]

	userInfo := model.IMUserInfo{
		Session: model.IMUserSession{
			Date:          responsedUserInfo.GetDate(),
			Name:          responsedUserInfo.GetName(),
			ApplicationID: responsedUserInfo.GetAppId(),
			Current:       responsedUserInfo.GetCurrent(),
			Device: model.IMUserDevice{
				IP:   responsedUserInfo.GetDevice().GetIp(),
				Push: responsedUserInfo.GetDevice().GetPush(),
				App: model.IMUserAgent{
					Name:    responsedUserInfo.GetDevice().GetApp().GetName(),
					Version: responsedUserInfo.GetDevice().GetApp().GetVersion(),
					OS:      responsedUserInfo.GetDevice().GetApp().GetOsVersion(),
					Device:  responsedUserInfo.GetDevice().GetApp().GetDevice(),
					Mobile:  responsedUserInfo.GetDevice().GetApp().GetMobile(),
					Tablet:  responsedUserInfo.GetDevice().GetApp().GetTablet(),
					Desktop: responsedUserInfo.GetDevice().GetApp().GetTablet(),
					Bot:     responsedUserInfo.GetDevice().GetApp().GetBot(),
					String:  responsedUserInfo.GetDevice().GetApp().GetString_(),
				},
			},
		},
	}

	return userInfo, nil
}

func (c *Connection) SendTextMessage(ctx context.Context, text string) (model.Response, *model.AppError) {
	_, err := c.srv.client.messageService.Api.SendText(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendTextRequest{
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

func (c *Connection) SendSystemMessage(ctx context.Context, msg model.SystemMessageOutbound) (model.Response, *model.AppError) {
	meta, err := structpb.NewStruct(msg.Metadata)
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendSystemMessage", "conv.msg", nil, err.Error(), http.StatusBadRequest)
	}

	_, err = c.srv.client.messageService.Api.SendSystemMessage(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendSystemMessageRequest{
		To: &p.Peer{
			Kind: &p.Peer_Contact{
				Contact: &p.PeerIdentity{
					Sub: c.msg.From.Sub,
					Iss: c.msg.From.Issuer,
				},
			},
		},
		Type:     msg.Type,
		Body:     msg.Text,
		Metadata: meta,
	})
	if err != nil {
		return model.CallResponseError, model.NewAppError("SendSystemMessage", "conv.msg", nil, err.Error(), http.StatusInternalServerError)
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
	_, err := c.srv.client.messageService.Api.SendDocument(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendDocumentRequest{
		To: &p.Peer{Kind: &p.Peer_Contact{Contact: &p.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Documents: docs,
		Body:      msg.Text,
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
	_, err := c.srv.client.messageService.Api.SendDocument(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendDocumentRequest{
		To: &p.Peer{Kind: &p.Peer_Contact{Contact: &p.PeerIdentity{
			Sub: c.msg.From.Sub,
			Iss: c.msg.From.Issuer,
		}}},
		Documents: docs,
		Body:      text,
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

	_, err := c.srv.client.messageService.Api.SendInteractive(metadata.NewOutgoingContext(ctx, c.hdrs), &p.SendInteractiveMessageRequest{
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
			kb.Id = btn.Code
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

func (c *Connection) GetQueueKey() *model.InQueueKey {
	c.log.Info("GetQueueKey")
	c.RLock()
	defer c.RUnlock()
	return c.queueKey
}

func (c *Connection) SetQueue(key *model.InQueueKey) bool {
	c.log.Info("SetQueue called")
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

func (c *Connection) ReceiveMessage(ctx context.Context, name string, timeout, messageTimeout int) ([]string, *model.AppError) {
	msgs, err := c.receive(ctx, timeout)
	if err != nil {
		return nil, err
	}

	if messageTimeout > 0 {
		var m []model.IMEventWrapper
		for err == nil {
			m, err = c.receive(ctx, messageTimeout)
			msgs = append(msgs, m...)
		}
	}

	return messageToText(msgs...), nil
}

func (connection *Connection) SendInteractive(ctx context.Context, interactive model.SendInteractiveRequest) (model.Response, *model.AppError) {
	protoInteractive := convertToProtoInteractive(&interactive.Interactive)
	if protoInteractive == nil {
		return model.CallResponseError, model.NewRequestError("im.connection.send_interactive", "received nil pointer interactive proto after converting")
	}

	outCtx := metadata.NewOutgoingContext(ctx, connection.hdrs)
	sendMD, _ := structpb.NewStruct(interactive.Metadata)
	request := &p.SendInteractiveMessageRequest{
		To: &p.Peer{
			Kind: &p.Peer_Contact{
				Contact: &p.PeerIdentity{
					Sub: connection.msg.From.Sub,
					Iss: connection.msg.From.Issuer,
				},
			},
		},
		Interactive: protoInteractive,
		Body:        &interactive.Body,
		Metadata:    sendMD,
	}

	if _, err := connection.srv.client.messageService.Api.SendInteractive(outCtx, request); err != nil {
		return model.CallResponseError, model.NewAppError("connection.send_interactive", "im.connection.send_interactive", nil, err.Error(), model.ExtractHTPPStatusCodeFromGRPC(err))
	}

	return model.CallResponseOK, nil
}

func convertToProtoInteractive(src *model.InteractiveGeneric[model.KeyboardButton]) *p.Interactive {
	if src == nil {
		return nil
	}

	dst := &p.Interactive{
		SingleUse: src.SingleUse,
	}

	if src.Documents != nil {
		dst.Attachments = &p.Interactive_Documents{
			Documents: convertToProtoDocuments(src.Documents),
		}
	} else if src.Images != nil {
		dst.Attachments = &p.Interactive_Images{
			Images: convertToProtoImages(src.Images),
		}
	}

	if src.Markup != nil {
		dst.Kind = &p.Interactive_Markup{
			Markup: convertToProtoMarkup(src.Markup),
		}
	} else if src.ListReply != nil {
		dst.Kind = &p.Interactive_ListReply{
			ListReply: convertoToProtoListReply(src.ListReply),
		}
	}

	return dst
}

func convertToProtoImages(src *model.Images) *p.Images {
	if src == nil {
		return nil
	}

	imgs := make([]*p.ImageInput, len(src.Images))
	for i, f := range src.Images {
		imgs[i] = &p.ImageInput{
			Id:       strconv.Itoa(f.Id),
			Name:     f.Name,
			MimeType: f.MimeType,
		}
	}
	return &p.Images{Images: imgs}
}

func convertToProtoDocuments(src *model.Documents) *p.Documents {
	if src == nil {
		return nil
	}
	docs := make([]*p.DocumentInput, len(src.Documents))
	for i, f := range src.Documents {
		docs[i] = &p.DocumentInput{
			Id:        strconv.Itoa(f.Id),
			MimeType:  f.MimeType,
			FileName:  f.Name,
			SizeBytes: &f.Size,
		}
	}
	return &p.Documents{Documents: docs}
}

func convertToProtoMarkup(src *model.KeyboardMarkup) *p.KeyboardMarkup {
	if src == nil {
		return nil
	}
	rows := make([]*p.KeyboardRow, len(src.Rows))
	for i, r := range src.Rows {
		rows[i] = &p.KeyboardRow{
			Buttons: convertToProtoButtons(r.Buttons),
		}
	}
	return &p.KeyboardMarkup{Rows: rows}
}

func convertoToProtoListReply(src *model.KeyboardListReply) *p.KeyboardListReply {
	if src == nil {
		return nil
	}
	sections := make([]*p.KeyboardRowWithSection, len(src.Sections))
	for i, s := range src.Sections {
		sections[i] = &p.KeyboardRowWithSection{
			Section: s.Section,
			Buttons: convertToProtoButtons(s.Buttons),
		}
	}
	return &p.KeyboardListReply{
		MainButtonTitle: src.MainButtonTitle,
		Sections:        sections,
	}
}

func convertToProtoButtons(src []model.KeyboardButton) []*p.KeyboardButton {
	res := make([]*p.KeyboardButton, len(src))
	for i, b := range src {
		btn := &p.KeyboardButton{
			Id:    b.ID,
			Label: b.Label,
		}

		if b.Metadata != nil {
			if m, err := structpb.NewStruct(b.Metadata); err == nil {
				btn.Metadata = m
			}
		}

		if b.URL != nil {
			btn.Kind = &p.KeyboardButton_Url{
				Url: &p.KeyboardButtonURL{Url: b.URL.URL},
			}
		} else if b.Callback != nil {
			btn.Kind = &p.KeyboardButton_Callback{
				Callback: &p.KeyboardButtonCallback{Data: b.Callback.Data},
			}
		} else if b.Request != nil {
			btn.Kind = &p.KeyboardButton_Request{
				Request: &p.KeyboardButtonRequest{Action: b.Request.Action},
			}
		}
		res[i] = btn
	}
	return res
}

func (c *Connection) IsTransfer() bool {
	return false
}

func (c *Connection) Stop(err error) {
	c.srv.stopConnection(c)
}

func (c *Connection) receive(_ context.Context, timeout int) ([]model.IMEventWrapper, *model.AppError) {
	ch := make(chan model.IMEventWrapper)
	defer func() {
		c.setStateWaitMessage(nil)
	}()

	err := c.setStateWaitMessage(ch)
	if err != nil {
		c.log.Warn("failed to set wait message chan")
		return nil, model.NewAppError("Conversation.WaitMessage", "conv.timeout.msg", nil, "Timeout", http.StatusInternalServerError)
	}

	t := time.After(time.Second * time.Duration(timeout))

	c.log.Debug("wait message", wlog.Duration("wait_sec", time.Second*time.Duration(timeout)))

	select {
	case <-c.Context().Done():
		c.log.Debug("context cancelled", wlog.Err(c.Context().Err()))
		return nil, model.NewAppError("Conversation.WaitMessage", "conv.timeout.msg.app_err", nil, c.Context().Err().Error(), http.StatusInternalServerError)
	case <-t:
		c.log.Debug("waiting message timeout", wlog.Int("timeout_sec", timeout))
		return nil, ErrWaitMessageTimeout
	case msgsProto := <-ch:
		c.log.Debug(fmt.Sprintf("receive message: %v", msgsProto))
		return []model.IMEventWrapper{msgsProto}, nil
	}
}

func (c *Connection) Type() model.ConnectionType { return model.ConnectionTypeIM }
func (c *Connection) Log() *wlog.Logger          { return c.log }
func (c *Connection) Id() string                 { return c.id }
func (c *Connection) SchemaId() int              { return c.schemaId }
func (c *Connection) NodeId() string             { return "" }
func (c *Connection) DomainId() int64            { return c.domainId }
func (c *Connection) Context() context.Context   { return c.ctx }

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

func (c *Connection) TreadInfo() model.ThreadInfo { return c.info }

func (c *Connection) treadInfo(ctx context.Context) (model.ThreadInfo, *model.AppError) {
	var info model.ThreadInfo
	result, err := c.srv.client.threadService.Api.Search(metadata.NewOutgoingContext(ctx, c.hdrs), &p.ThreadSearchRequest{
		Fields: nil,
		Ids:    []string{c.threadId},
		Size:   1,
	})
	if err != nil {
		return info, model.NewAppError("Connection.TreadInfo", "conv.thread_info.app_err", nil, err.Error(), http.StatusInternalServerError)
	}

	if len(result.Items) == 0 {
		return info, model.NewAppError("Connection.TreadInfo", "conv.thread_info.not_found", nil, "result thread info set contains zero records", http.StatusNotFound)
	}

	infoResult := result.Items[0]
	info.Subject = infoResult.Subject
	info.Description = infoResult.Description
	if infoResult.LastMsg != nil {
		info.LastMessage = infoResult.LastMsg.Body
	}

	for _, v := range infoResult.Members {
		info.Members = append(info.Members, model.ThreadMember{
			Type:     v.GetContact().GetType(),
			Name:     v.GetContact().GetName(),
			Iss:      v.GetContact().GetIss(),
			Sub:      v.GetContact().GetSub(),
			Role:     int(v.GetRole()),
			MemberId: v.GetId(),
		})
	}

	if infoResult.Variables == nil {
		return info, nil
	}
	info.Variables = make(map[string]string)

	for k, v := range infoResult.Variables.Variables {
		if v.Value != nil {
			if raw, err := v.Value.MarshalJSON(); err != nil {
				info.Variables[k] = string(raw)
			}
		}
	}

	return info, nil
}

func toVariables(in map[string]json.RawMessage) map[string]string {
	vars := make(map[string]string)

	for k, v := range in {
		vars[k] = string(v)
	}

	return vars
}

func messageToText(messages ...model.IMEventWrapper) []string {
	msgs := make([]string, 0, len(messages))

	for _, m := range messages {
		msgs = append(msgs, m.GetPayload().Message().Text)
	}

	return msgs
}
