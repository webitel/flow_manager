package model

import (
	"context"
	"time"
)

const (
	IMEventTypeMessage  string = "message"
	IMEventTypeCallback string = "callback"
)

type IMDialog interface {
	Connection

	Via() string
	DeviceID() string
	ThreadId() string
	From() ImEndpoint
	To() ImEndpoint
	LastMessage() Message
	SchemaId() int
	Stop(err error)
	Complete(id string)
	IsTransfer() bool
	TransferredSchema() (int, string)
	CompleteId() string
	NewContext() context.Context
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
	SendTextMessage(ctx context.Context, text string) (Response, *AppError)
	SendSystemMessage(ctx context.Context, msg SystemMessageOutbound) (Response, *AppError)
	SendDocumentMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
	SendFile(ctx context.Context, text string, f *File, kind string) (Response, *AppError)
	SendMenu(ctx context.Context, menu *ChatMenuArgs) (Response, *AppError)
	ReceiveMessage(ctx context.Context, name string, timeout, messageTimeout int) ([]string, *AppError)
	Export(ctx context.Context, vars []string) (Response, *AppError)
	UnSet(ctx context.Context, varKeys []string) (Response, *AppError)
	TreadInfo() ThreadInfo
	GetQueueKey() *InQueueKey
	SetQueue(*InQueueKey) bool
	DumpExportVariables() map[string]string
	SendInteractive(ctx context.Context, interactive SendInteractiveRequest) (Response, *AppError)
	GetAuthSession(ctx context.Context, deviceID string) (IMUserInfo, *AppError)
	HandleGateInfo(ctx context.Context, gateType IMGateType, id string) (*IMGate, *AppError)
}

type ThreadMember struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Iss      string `json:"iss"`
	Sub      string `json:"sub"`
	MemberId string `json:"member_id"`
	Role     int    `json:"role"`
}

type ThreadInfo struct {
	Subject     string            `json:"subject"`
	Description string            `json:"description"`
	Members     []ThreadMember    `json:"members"`
	LastMessage string            `json:"last_message"`
	Variables   map[string]string `json:"variables"`
}

type CCQueueEvent struct {
	AttemptId int64  `json:"attempt_id"`
	Event     string `json:"event"`
	Result    string `json:"result"`
}

type IMEventWrapper interface {
	GetID() string
	GetUserID() string
	GetDomainID() int64
	IsEcho() bool
	GetPayload() IMEvent
	GetType() string
	JWTPayload() string
	DeviceID() string
	Via() string
}

type IMEvent interface {
	GetThreadID() string
	MessageID() string
	Sender() ImEndpoint
	Receivers() []ImEndpoint
	Message() Message
}

// MessageWrapper представляє кореневий об'єкт
type MessageWrapper[T IMEvent] struct {
	ID         string `json:"id"`
	Message    T      `json:"payload"`
	UserID     string `json:"user_id"`
	DomainID   int64  `json:"domain_id"`
	Echo       bool   `json:"echo"`
	jwtPayload string `json:"-"`
	deviceID   string `json:"-"`
	Type       string `json:"-"`
	via        string `json:"-"`
}

type IMBotControlGrantedEvent struct {
	ThreadID    string `json:"thread_id"`
	DomainID    int    `json:"domain_id"`
	MemberID    string `json:"member_id"`
	AutoLeave   bool   `json:"auto_leave"`
	IsResume    bool   `json:"is_resume"`
	ReleasedSub int    `json:"released_sub"`
	Sub         int    `json:"sub"`
}

func (w MessageWrapper[T]) GetID() string                 { return w.ID }
func (w MessageWrapper[T]) GetUserID() string             { return w.UserID }
func (w MessageWrapper[T]) GetDomainID() int64            { return w.DomainID }
func (w MessageWrapper[T]) IsEcho() bool                  { return w.Echo }
func (w MessageWrapper[T]) GetPayload() IMEvent           { return w.Message }
func (w MessageWrapper[T]) GetType() string               { return w.Type }
func (w MessageWrapper[T]) JWTPayload() string            { return w.jwtPayload }
func (w *MessageWrapper[T]) SetJWTPayload(payload string) { w.jwtPayload = payload }
func (w MessageWrapper[T]) DeviceID() string              { return w.deviceID }
func (w *MessageWrapper[T]) SetDeviceID(deviceID string)  { w.deviceID = deviceID }
func (w *MessageWrapper[T]) SetVia(via string)            { w.via = via }
func (w MessageWrapper[T]) Via() string                   { return w.via }

// Message описує вкладений об'єкт повідомлення
type Message struct {
	ID          string        `json:"id"`
	ThreadID    string        `json:"thread_id"`
	DomainID    int           `json:"domain_id"`
	From        ImEndpoint    `json:"from"`
	To          []ImEndpoint  `json:"to"`
	Text        string        `json:"text"`
	CreatedAt   int64         `json:"created_at"` // Unix timestamp у мілісекундах
	Subject     string        `json:"subject"`
	Description string        `json:"description"`
	Type        string        `json:"type"`
	Contact     *Contact      `json:"contact,omitempty"`
	Location    *Location     `json:"location,omitempty"`
	Documents   []MessageFile `json:"documents,omitempty"`
  System      *SystemIMMessage `json:"system,omitempty"`
}

type Contact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
	Name      string  `json:"name"`
}

type MessageFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Mime string `json:"mime"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type SystemIMMessage struct {
	Type string `json:"type,omitempty"`
	Sub  int    `json:"sub"`
}

func (m Message) GetThreadID() string     { return m.ThreadID }
func (m Message) MessageID() string       { return m.ID }
func (m Message) Sender() ImEndpoint      { return m.From }
func (m Message) Receivers() []ImEndpoint { return m.To }
func (m Message) Message() Message        { return m }

type SystemMessageOutbound struct {
	Type     string         `json:"type"`
	Text     string         `json:"text"`
	Metadata map[string]any `json:"metadata"`
}

// From описує відправника
type ImEndpoint struct {
	ID       string `json:"id"`
	Type     int    `json:"type"`
	Sub      string `json:"sub"`
	Issuer   string `json:"issuer"`
	Name     string `json:"name"`
	MemberID string `json:"member_id"`
	Role     int    `json:"role"`
}

func (e *ImEndpoint) GateType() IMGateType { return IMGateTypeFromString(e.Issuer) }

type InteractiveCallback struct {
	DomainID     int
	ReactedBy    ImEndpoint `json:"reacted_by"`
	Receiver     ImEndpoint `json:"receiver"`
	InReplyTo    string     `json:"in_reply_to"`
	ThreadID     string     `json:"thread_id"`
	ButtonCode   string     `json:"button_code"`
	CallbackData string     `json:"callback_data"`
	ReactedAt    time.Time  `json:"reacted_at"`
}

func (c InteractiveCallback) GetThreadID() string     { return c.ThreadID }
func (c InteractiveCallback) MessageID() string       { return c.InReplyTo }
func (c InteractiveCallback) Sender() ImEndpoint      { return c.ReactedBy }
func (c InteractiveCallback) Message() Message        { return Message{Text: c.ButtonCode} }
func (c InteractiveCallback) Receivers() []ImEndpoint { return []ImEndpoint{c.Receiver} }
func (c InteractiveCallback) System() any             { return nil }
