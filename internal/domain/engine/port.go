package engine

import (
	"context"

	"github.com/webitel/flow_manager/internal/domain/call"
)

type Client interface {
	MakeCall(ctx context.Context, req call.OutboundCallRequest) (string, error)
	GenerateFeedback(ctx context.Context, domainId int64, sourceId, source string, payload map[string]string) (string, error)
}
