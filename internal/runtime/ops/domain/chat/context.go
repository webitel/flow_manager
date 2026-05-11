package chat

import (
	"context"
	"strings"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/model"
)

func conversationFromContext(ctx context.Context) (model.Conversation, bool) {
	conn := connctx.ConnectionFromContext(ctx)
	if conn == nil {
		return nil, false
	}
	conv, ok := conn.(model.Conversation)
	return conv, ok
}

// rawStringSlice extracts []string from in.Node.RawArgs, expanding variables.
func rawStringSlice(in ops.OpInput) []string {
	switch v := in.Node.RawArgs.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, ops.ExpandStr(s, in.Variables, in.GlobalVar))
			}
		}
		return out
	case string:
		if s := ops.ExpandStr(v, in.Variables, in.GlobalVar); s != "" {
			return []string{s}
		}
	}
	return nil
}

// resolveServer picks a non-empty server string and strips a trailing slash.
func resolveServer(fileServer, fallback string) string {
	s := fileServer
	if s == "" {
		s = fallback
	}
	return strings.TrimSuffix(s, "/")
}
