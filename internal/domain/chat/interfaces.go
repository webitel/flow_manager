package chat

// moved from model/chat.go and model/im.go

import (
	"context"

	ai_bots "github.com/webitel/flow_manager/api/gen/ai_bots"
	proto "github.com/webitel/flow_manager/api/gen/chat"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
)

// Conversation is the chat-specific connection interface.
type Conversation interface {
	flow.Connection
	ProfileId() int64
	Stop(err error, cause proto.CloseConversationCause)
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (flow.Response, error)
	SendTextMessage(ctx context.Context, text string) (flow.Response, error)
	SendMenu(ctx context.Context, menu *ChatMenuArgs) (flow.Response, error)
	SendImageMessage(ctx context.Context, url, name, text, kind string) (flow.Response, error)
	ReceiveMessage(ctx context.Context, name string, timeout, messageTimeout int) ([]string, error)
	Bridge(ctx context.Context, userId int64, timeout int) error
	Export(ctx context.Context, vars []string) (flow.Response, error)
	DumpExportVariables() map[string]string
	NodeName() string
	SchemaId() int32
	UserId() int64
	BreakCause() string
	IsTransfer() bool
	SendFile(ctx context.Context, text string, f *files.File, kind string) (flow.Response, error)

	SetQueue(*queue.InQueueKey) bool
	GetQueueKey() *queue.InQueueKey
	UnSet(ctx context.Context, varKeys []string) (flow.Response, error)
	LastMessages(limit int) []ChatMessage
	Bot(ctx context.Context, cli ai_bots.ConverseServiceClient, id string) (flow.Response, error)
}

// IMDialog is the IM-specific connection interface.
type IMDialog interface {
	flow.Connection
	ThreadId() string
	From() ImEndpoint
	To() ImEndpoint
	LastMessage() Message
	SchemaId() int
	Stop(err error)
	IsTransfer() bool
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (flow.Response, error)
	SendTextMessage(ctx context.Context, text string) (flow.Response, error)
	SendImageMessage(ctx context.Context, msg ChatMessageOutbound) (flow.Response, error)
	SendDocumentMessage(ctx context.Context, msg ChatMessageOutbound) (flow.Response, error)
	SendFile(ctx context.Context, text string, f *files.File, kind string) (flow.Response, error)
	SendMenu(ctx context.Context, menu *ChatMenuArgs) (flow.Response, error)
	Export(ctx context.Context, vars []string) (flow.Response, error)
	UnSet(ctx context.Context, varKeys []string) (flow.Response, error)
	LastMessages(limit int) []ChatMessage
	GetQueueKey() *queue.InQueueKey
	SetQueue(*queue.InQueueKey) bool
	DumpExportVariables() map[string]string
}
