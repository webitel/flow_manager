package engine

import (
	"context"

	"github.com/webitel/flow_manager/model"
)

type Client interface {
	MakeCall(ctx context.Context, req model.OutboundCallRequest) (string, error)
	GenerateFeedback(ctx context.Context, domainId int64, sourceId, source string, payload map[string]string) (string, error)
}
