package model

import (
	"context"

	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
)

// Re-exports for backward compatibility.
type CCQueueEvent = chatdomain.CCQueueEvent
type MessageWrapper = chatdomain.MessageWrapper
type Message = chatdomain.Message
type ImEndpoint = chatdomain.ImEndpoint

// IMDialog is the IM-specific connection interface.
type IMDialog interface {
	Connection
	ThreadId() string
	From() ImEndpoint
	To() ImEndpoint
	LastMessage() Message
	SchemaId() int
	Stop(err error)
	IsTransfer() bool
	SendMessage(ctx context.Context, msg ChatMessageOutbound) (Response, error)
	SendTextMessage(ctx context.Context, text string) (Response, error)
	SendImageMessage(ctx context.Context, msg ChatMessageOutbound) (Response, error)
	SendDocumentMessage(ctx context.Context, msg ChatMessageOutbound) (Response, error)
	SendFile(ctx context.Context, text string, f *File, kind string) (Response, error)
	SendMenu(ctx context.Context, menu *ChatMenuArgs) (Response, error)
	Export(ctx context.Context, vars []string) (Response, error)
	UnSet(ctx context.Context, varKeys []string) (Response, error)
	LastMessages(limit int) []ChatMessage
	GetQueueKey() *InQueueKey
	SetQueue(*InQueueKey) bool
	DumpExportVariables() map[string]string
}
