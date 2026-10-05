package emailevent

import (
	"context"
	"testing"

	"github.com/webitel/wlog"
)

var testLog = wlog.NewLogger(&wlog.LoggerConfiguration{})

func TestConnectionVariables(t *testing.T) {
	event, err := Decode([]byte(`{
  "event_id": "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60", "event_type": "email.message.inbound",
  "schema_version": 1, "domain_id": 1, "profile_id": 2, "flow_id": 3, "thread_id": 4, "message_id": 5,
  "snapshot": {
    "internet_message_id": "<m@example.com>", "in_reply_to": "<p@example.com>", "subject": "Hi",
    "message_kind": "regular", "thread_kind": "regular", "contact_id": 6,
    "from": [{"address": "a@example.com", "name": "A"}],
    "to": [{"address": "b@example.com", "name": "B"}, {"address": "c@example.com", "name": ""}],
    "body_text": "text", "body_text_truncated": true,
    "attachments": [{"id": 7, "name": "f.pdf", "size": 8, "mime": "application/pdf", "view_name": null, "channel": "mail"}],
    "unknown": "ignored"
  }
}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	vars := NewConnection(context.Background(), "flow-1", event, testLog).Variables()
	want := map[string]string{
		"id": "5", "message_id": "<m@example.com>", "in_reply_to": "<p@example.com>", "subject": "Hi",
		"from": "a@example.com", "to": "b@example.com,c@example.com", "cc": "", "body": "text",
		"attachments":    `[{"id":7,"name":"f.pdf","size":8,"mime":"application/pdf","view_name":null,"server":"","channel":"mail"}]`,
		"wbt_contact_id": "6", "email_event_id": "0199b2a4-3c4e-7a10-8f2e-1b2c3d4e5f60",
		"email_profile_id": "2", "email_thread_id": "4", "email_message_id": "5", "body_truncated": "true",
	}
	for name, value := range want {
		if vars[name] != value {
			t.Errorf("%s = %q, want %q", name, vars[name], value)
		}
	}
	if _, ok := vars["body_html"]; ok {
		t.Error("body_html must not be set")
	}
}

func TestConnectionLaunchState(t *testing.T) {
	flowId := int64(1)
	newConn := func() *Connection {
		return NewConnection(context.Background(), "flow-1", &InboundMessage{FlowId: &flowId}, testLog)
	}

	abandoned := newConn()
	if !abandoned.Abandon() || abandoned.MarkStarted() {
		t.Error("an abandoned launch must not start")
	}

	started := newConn()
	if !started.MarkStarted() || started.Abandon() {
		t.Error("a started launch must not be abandoned")
	}
}
