// Package chat provides native ops for chat-specific operations:
// broadcastChatMessage, chatHistory.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/model"
)

// ChatDeps is the narrow interface required by all chat native ops.
type ChatDeps interface {
	ChatProfileType(domainId int64, profileId int) (string, error)
	BroadcastChatMessage(ctx context.Context, domainId int64, req model.BroadcastChat, peers []model.BroadcastPeer) (*model.BroadcastChatResponse, error)
	GetChatMessagesByConversationId(ctx context.Context, domainId int64, conversationId string, limit int64) (*[]model.ChatMessage, error)
	ParseChatMessages(messages *[]model.ChatMessage, format string) (string, error)
}

// Register adds broadcastChatMessage and chatHistory to reg.
func Register(reg *ops.Registry, deps ChatDeps) {
	reg.Register("broadcastChatMessage", &broadcastChatMessageOp{deps: deps})
	reg.Register("chatHistory", &chatHistoryOp{deps: deps})
}

// ── broadcastChatMessage ──────────────────────────────────────────────────────

type broadcastChatMessageOp struct{ deps ChatDeps }

func (o *broadcastChatMessageOp) Kind() ops.OpKind { return ops.OpKindSync }

func (o *broadcastChatMessageOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	var argv model.BroadcastChat
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, fmt.Errorf("broadcastChatMessage: %w", err)
	}
	if len(argv.Peer) == 0 {
		return ops.OpOutput{}, fmt.Errorf("broadcastChatMessage: peer is required")
	}

	var typeProfile string
	if argv.Profile.Id > 0 {
		var err error
		typeProfile, err = o.deps.ChatProfileType(in.DomainID, argv.Profile.Id)
		if err != nil {
			return ops.OpOutput{}, fmt.Errorf("broadcastChatMessage: %w", err)
		}
	}

	peer := make([]model.BroadcastPeer, 0, len(argv.Peer))
	for _, v := range argv.Peer {
		switch p := v.(type) {
		case string:
			peer = append(peer, model.BroadcastPeer{
				Id:   p,
				Type: typeProfile,
				Via:  fmt.Sprintf("%d", argv.Profile.Id),
			})
		case map[string]any:
			expand := func(key string) string {
				s, _ := p[key].(string)
				return ops.ExpandStr(s, in.Variables, in.GlobalVar)
			}
			peer = append(peer, model.BroadcastPeer{
				Id:   expand("id"),
				Type: expand("type"),
				Via:  expand("via"),
			})
		}
	}

	resp, err := o.deps.BroadcastChatMessage(ctx, in.DomainID, argv, peer)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("broadcastChatMessage: %w", err)
	}

	setVars := make(map[string]string, len(resp.Variables)+2)

	if len(resp.Failed) != 0 && (argv.ResponseCode != "" || argv.FailedReceivers != "") {
		if argv.ResponseCode != "" {
			setVars[argv.ResponseCode] = resp.Failed[0].Error
		}
		if argv.FailedReceivers != "" {
			b, _ := json.Marshal(resp)
			setVars[argv.FailedReceivers] = string(b)
		}
	}

	for k, v := range resp.Variables {
		setVars[k] = v
	}

	return ops.OpOutput{SetVars: setVars}, nil
}

// ── chatHistory ───────────────────────────────────────────────────────────────

type chatHistoryOp struct{ deps ChatDeps }

func (o *chatHistoryOp) Kind() ops.OpKind { return ops.OpKindSync }

type chatHistoryArgs struct {
	ConversationId string `json:"conversationId,omitempty"`
	Variable       string `json:"variable,omitempty"`
	Format         string `json:"format,omitempty"`
	Timeout        int    `json:"timeout,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

func (o *chatHistoryOp) Execute(ctx context.Context, in ops.OpInput) (ops.OpOutput, error) {
	argv := chatHistoryArgs{
		ConversationId: in.ConnID,
	}
	if err := ops.DecodeArgs(in, &argv); err != nil {
		return ops.OpOutput{}, fmt.Errorf("chatHistory: %w", err)
	}
	if argv.Limit == 0 {
		argv.Limit = 300
	}
	if argv.Timeout == 0 {
		argv.Timeout = 3000
	}

	qCtx, cancel := context.WithTimeout(ctx, time.Millisecond*time.Duration(argv.Timeout))
	defer cancel()

	messages, err := o.deps.GetChatMessagesByConversationId(qCtx, in.DomainID, argv.ConversationId, int64(argv.Limit))
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("chatHistory: %w", err)
	}

	text, err := o.deps.ParseChatMessages(messages, argv.Format)
	if err != nil {
		return ops.OpOutput{}, fmt.Errorf("chatHistory: %w", err)
	}

	return ops.OpOutput{SetVars: map[string]string{argv.Variable: text}}, nil
}
