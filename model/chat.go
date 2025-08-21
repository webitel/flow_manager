package model

import (
	"context"
	"github.com/webitel/flow_manager/gen/ai_bots"
	proto "github.com/webitel/flow_manager/gen/chat"
)

const (
	// TODO
	ConversationStartMessageVariable = "start_message"
	ConversationSessionId            = "uuid"
	ConversationProfileId            = "wbt_profile_id"

	BreakChatTransferCause = "TRANSFER"
)

type ChatAction string

const (
	ChatActionTyping ChatAction = "typing"
	ChatActionCancel ChatAction = "cancel"
)

type ChatButton struct {
	Caption string `json:"caption"`
	Text    string `json:"text"`
	Type    string `json:"type"`
	Url     string `json:"url"`
	Code    string `json:"code"`
}

type ChatMenuArgs struct {
	Type    string         `json:"type"` // type
	Buttons [][]ChatButton `json:"buttons"`
	Inline  [][]ChatButton `json:"inline"`
	Text    string         `json:"text"`
	NoInput bool           `json:"noInput"`
	Kind    string         `json:"kind"`
}

type ChatMessageOutbound struct {
	Type    string
	Text    string
	File    *File
	Server  string         `json:"server" db:"-"` // TODO
	Buttons [][]ChatButton `json:"buttons"`
	Inline  [][]ChatButton `json:"inline"`
	NoInput bool           `json:"noInput"`
	Kind    string         `json:"kind"`
}

type BroadcastPeer struct {
	Id   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Via  string `json:"via,omitempty"`
}

type BroadcastChat struct {
	Type    any
	Profile struct {
		Id int
	}
	Peer    []any
	Text    string
	File    *File
	Buttons [][]ChatButton `json:"buttons"`

	Variables    map[string]string
	DomainId     int64
	ResponseCode string `json:"responseCode"`
	// FailedReceivers used to set the variable name in which will be saved failed receivers. (if not set then info about failed receivers will not be saved)
	FailedReceivers string `json:"failedReceivers"`
	// Timeout determines how much time chat_manager is waiting (in secs) for the response from the host(telegram|whatsapp..) on our callback url.
	Timeout int64 `json:"timeout"`
}

type BroadcastChatResponse struct {
	Failed    []*FailedReceiver `json:"failed"`
	Variables map[string]string
}

type FailedReceiver struct {
	Id    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

type ChatMessage struct {
	Text       string `json:"text,omitempty" db:"msg"`
	CreatedAt  string `json:"created_at,omitempty" db:"created_at"`
	Type       string `json:"type,omitempty" db:"type"`
	User       string `json:"user,omitempty" db:"name"`
	IsInternal bool   `json:"isInternal" db:"internal"`
}

type Conversation interface {
	Connection
	ProfileId() int64
	Stop(err *AppError, cause proto.CloseConversationCause)
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
	SendTextMessage(ctx context.Context, text string) (Response, *AppError)
	SendMenu(ctx context.Context, menu *ChatMenuArgs) (Response, *AppError)
	SendImageMessage(ctx context.Context, url string, name string, text string, kind string) (Response, *AppError)
	ReceiveMessage(ctx context.Context, name string, timeout int, messageTimeout int) ([]string, *AppError)
	Bridge(ctx context.Context, userId int64, timeout int) *AppError
	Export(ctx context.Context, vars []string) (Response, *AppError)
	DumpExportVariables() map[string]string
	NodeName() string
	SchemaId() int32
	UserId() int64
	BreakCause() string
	IsTransfer() bool
	SendFile(ctx context.Context, text string, f *File, kind string) (Response, *AppError)

	SetQueue(*InQueueKey) bool
	GetQueueKey() *InQueueKey
	UnSet(ctx context.Context, varKeys []string) (Response, *AppError)
	LastMessages(limit int) []ChatMessage
	Bot(ctx context.Context, cli ai_bots.ConverseServiceClient, id string) (Response, *AppError)
}
