package model

import (
	"context"

	"github.com/webitel/flow_manager/api/gen/ai_bots"
	proto "github.com/webitel/flow_manager/api/gen/chat"
	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
)

const (
	// TODO
	ConversationStartMessageVariable = "start_message"
	ConversationSessionId            = "uuid"
	ConversationProfileId            = "wbt_profile_id"

	BreakChatTransferCause = "TRANSFER"
)

// Re-exports for backward compatibility.
type (
	ChatAction            = chatdomain.ChatAction
	ChatButton            = chatdomain.ChatButton
	ChatMenuArgs          = chatdomain.ChatMenuArgs
	ChatMessageOutbound   = chatdomain.ChatMessageOutbound
	BroadcastPeer         = chatdomain.BroadcastPeer
	BroadcastChat         = chatdomain.BroadcastChat
	BroadcastChatResponse = chatdomain.BroadcastChatResponse
	FailedReceiver        = chatdomain.FailedReceiver
	ChatMessage           = chatdomain.ChatMessage
)

const (
	ChatActionTyping = chatdomain.ChatActionTyping
	ChatActionCancel = chatdomain.ChatActionCancel
)

// Conversation is the chat-specific connection interface.
// Kept here (not aliased) because it references *AppError — moving it would
// create an import cycle until AppError is extracted (Phase 5.2).
type Conversation interface {
	Connection
	ProfileId() int64
	Stop(err error, cause proto.CloseConversationCause)
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (Response, error)
	SendTextMessage(ctx context.Context, text string) (Response, error)
	SendMenu(ctx context.Context, menu *ChatMenuArgs) (Response, error)
	SendImageMessage(ctx context.Context, url, name, text, kind string) (Response, error)
	ReceiveMessage(ctx context.Context, name string, timeout, messageTimeout int) ([]string, error)
	Bridge(ctx context.Context, userId int64, timeout int) error
	Export(ctx context.Context, vars []string) (Response, error)
	DumpExportVariables() map[string]string
	NodeName() string
	SchemaId() int32
	UserId() int64
	BreakCause() string
	IsTransfer() bool
	SendFile(ctx context.Context, text string, f *File, kind string) (Response, error)

	SetQueue(*InQueueKey) bool
	GetQueueKey() *InQueueKey
	UnSet(ctx context.Context, varKeys []string) (Response, error)
	LastMessages(limit int) []ChatMessage
	Bot(ctx context.Context, cli ai_bots.ConverseServiceClient, id string) (Response, error)
}
