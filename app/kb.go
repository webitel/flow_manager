package app

import (
	"context"

	"github.com/webitel/flow_manager/gen/kb"
)

// kbServiceName is the knowledge base service in the service discovery.
const kbServiceName = "webitel-kb"

// KBSemanticSearch searches the knowledge base under the token's rights.
func (fm *FlowManager) KBSemanticSearch(ctx context.Context, token string, req *kb.SemanticSearchRequest) (*kb.SemanticSearchResponse, error) {
	return fm.kbRetrieval.Api.SemanticSearch(fm.kbRetrieval.WithToken(ctx, token), req)
}
