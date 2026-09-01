package im

import (
	"testing"

	p "github.com/webitel/flow_manager/gen/im/api/gateway/v1"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

// Covers CW-60 Bug 1: on a bot.control.granted event the schema is started for
// the customer peer. This locks the participant-selection: pick the first human
// member that is not the granted bot.
func TestSelectCustomerPeer(t *testing.T) {
	member := func(id, sub, iss, name string, isBot bool, role p.ThreadRole) *p.ThreadMember {
		return &p.ThreadMember{
			Id:   id,
			Role: role,
			Contact: &p.Contact{
				Sub:   sub,
				Iss:   iss,
				Name:  name,
				IsBot: isBot,
			},
		}
	}

	const botSub = "42"

	tests := []struct {
		name    string
		members []*p.ThreadMember
		wantSub string // "" means no peer resolved
	}{
		{
			name: "picks the human customer over the granted bot",
			members: []*p.ThreadMember{
				member("m-bot", botSub, "bot", "QueueBot", true, p.ThreadRole_ROLE_OWNER),
				member("m-cust", "cust-1", "webitel", "Customer", false, p.ThreadRole_ROLE_OWNER),
			},
			wantSub: "cust-1",
		},
		{
			name: "skips a non-bot participant that is the granted bot by sub",
			members: []*p.ThreadMember{
				member("m-self", botSub, "webitel", "Self", false, p.ThreadRole_ROLE_OWNER),
				member("m-cust", "cust-2", "webitel", "Customer", false, p.ThreadRole_ROLE_MEMBER),
			},
			wantSub: "cust-2",
		},
		{
			name: "only bots present resolves nothing",
			members: []*p.ThreadMember{
				member("m-bot", botSub, "bot", "QueueBot", true, p.ThreadRole_ROLE_OWNER),
				member("m-bot2", "99", "bot", "OtherBot", true, p.ThreadRole_ROLE_OWNER),
			},
			wantSub: "",
		},
		{
			name:    "empty members resolves nothing",
			members: nil,
			wantSub: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectCustomerPeer(tt.members, botSub)
			if got.Sub != tt.wantSub {
				t.Fatalf("selectCustomerPeer sub = %q, want %q", got.Sub, tt.wantSub)
			}
		})
	}
}

// Covers CW-60 Bug 1: a schema must start ONLY on bot.control.granted, never from an
// inbound message. A bot receiver without a live connection (e.g. the owner suspended
// lower in the control stack while another bot holds the top) must be skipped by
// nodeMessage — otherwise it would run in parallel with the active controller.
func TestNodeMessage_DoesNotStartSchemaWithoutLiveConnection(t *testing.T) {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
	s := &server{
		log:             log,
		connectionStore: NewConnectionStore(log),
		consume:         make(chan model.Connection, 1),
	}

	msg := model.MessageWrapper[model.Message]{
		Type: model.IMEventTypeMessage,
		Message: model.Message{
			ThreadID: "thread-1",
			From:     model.ImEndpoint{Sub: "cust-1", Issuer: "webitel"},
			To:       []model.ImEndpoint{{Sub: "2614", Issuer: IMUserTypeBot}},
		},
	}

	if err := s.nodeMessage(msg); err != nil {
		t.Fatalf("nodeMessage returned error: %v", err)
	}

	if _, ok := s.connectionStore.Get("thread-1.2614"); ok {
		t.Fatal("nodeMessage started a schema for a bot without a live connection")
	}

	select {
	case <-s.consume:
		t.Fatal("nodeMessage dispatched a dialog to the schema runner; it must only deliver to live connections")
	default:
	}
}
