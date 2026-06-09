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
	ThreadId() string
	From() ImEndpoint
	To() ImEndpoint
	LastMessage() Message
	SchemaId() int
	Stop(err error)
	IsTransfer() bool
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
	SendTextMessage(ctx context.Context, text string) (Response, *AppError)
	SendSystemMessage(ctx context.Context, msg SystemMessageOutbound) (Response, *AppError)
	SendImageMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
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
}

type ThreadMember struct {
	Type     string
	Name     string
	Iss      string
	Sub      string
	MemberId string
	Role     int
}

type ThreadInfo struct {
	Subject     string
	Description string
	Members     []ThreadMember
	LastMessage string
	Variables   map[string]string
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
	ID       string `json:"id"`
	Message  T      `json:"payload"`
	UserID   string `json:"user_id"`
	DomainID int64  `json:"domain_id"`
	Echo     bool   `json:"echo"`
	Type     string `json:"-"`
}

func (w MessageWrapper[T]) GetID() string       { return w.ID }
func (w MessageWrapper[T]) GetUserID() string   { return w.UserID }
func (w MessageWrapper[T]) GetDomainID() int64  { return w.DomainID }
func (w MessageWrapper[T]) IsEcho() bool        { return w.Echo }
func (w MessageWrapper[T]) GetPayload() IMEvent { return w.Message }
func (w MessageWrapper[T]) GetType() string     { return w.Type }

// Message описує вкладений об'єкт повідомлення
type Message struct {
	ID          string       `json:"id"`
	ThreadID    string       `json:"thread_id"`
	DomainID    int          `json:"domain_id"`
	From        ImEndpoint   `json:"from"`
	To          []ImEndpoint `json:"to"`
	Text        string       `json:"text"`
	CreatedAt   int64        `json:"created_at"` // Unix timestamp у мілісекундах
	Subject     string       `json:"subject"`
	Description string       `json:"description"`
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

type InteractiveCallback struct {
	ReactedBy    ImEndpoint `json:"reacted_by"`
	Receiver     ImEndpoint `json:"receiver"`
	InReplyTo    string     `json:"in_reply_to"`
	ThreadID     string     `json:"thread_id"`
	ButtonCode   string     `json:"button_code"`
	CallbackData string     `json:"callback_data"`
	ReactedAt    time.Time  `json:"reacted_at"`
	DomainID     int
}

func (c InteractiveCallback) GetThreadID() string     { return c.ThreadID }
func (c InteractiveCallback) MessageID() string       { return c.InReplyTo }
func (c InteractiveCallback) Sender() ImEndpoint      { return c.ReactedBy }
func (c InteractiveCallback) Message() Message        { return Message{Text: c.ButtonCode} }
func (c InteractiveCallback) Receivers() []ImEndpoint { return []ImEndpoint{c.Receiver} }
