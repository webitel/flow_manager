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
// Kept here (not aliased) because it references *AppError — moving it would
// create an import cycle until AppError is extracted (Phase 5.2).
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
	SendImageMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
	SendDocumentMessage(ctx context.Context, msg ChatMessageOutbound) (Response, *AppError)
	SendFile(ctx context.Context, text string, f *File, kind string) (Response, *AppError)
	SendMenu(ctx context.Context, menu *ChatMenuArgs) (Response, *AppError)
	Export(ctx context.Context, vars []string) (Response, *AppError)
	UnSet(ctx context.Context, varKeys []string) (Response, *AppError)
	LastMessages(limit int) []ChatMessage
	GetQueueKey() *InQueueKey
	SetQueue(*InQueueKey) bool
	DumpExportVariables() map[string]string
}
