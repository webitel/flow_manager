package meeting

import (
	"context"
	"fmt"
	"testing"

	domainmeeting "github.com/webitel/flow_manager/internal/domain/meeting"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── fakeMeetingClient ─────────────────────────────────────────────────────────

type fakeMeetingClient struct {
	url string
	err error
}

func (f *fakeMeetingClient) Create(_ context.Context, _ int64, _ string, _ int, _ string, _ map[string]string) (string, error) {
	return f.url, f.err
}
func (f *fakeMeetingClient) Get(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}

var _ domainmeeting.Client = (*fakeMeetingClient)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func meetingInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreateMeeting_NoSetVar(t *testing.T) {
	op := New(&fakeMeetingClient{url: "https://meet.example.com/abc"})
	_, err := op.Execute(context.Background(), meetingInput(map[string]any{
		"title": "Team sync",
	}))
	if err == nil {
		t.Fatal("expected error when setVar is empty")
	}
}

func TestCreateMeeting_Success(t *testing.T) {
	op := New(&fakeMeetingClient{url: "https://meet.example.com/xyz"})
	out, err := op.Execute(context.Background(), meetingInput(map[string]any{
		"setVar":    "meeting_url",
		"title":     "Support call",
		"expireSec": 3600,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["meeting_url"] != "https://meet.example.com/xyz" {
		t.Errorf("meeting_url = %q, want https://meet.example.com/xyz", out.SetVars["meeting_url"])
	}
}

func TestCreateMeeting_DepError(t *testing.T) {
	op := New(&fakeMeetingClient{err: fmt.Errorf("service unavailable")})
	_, err := op.Execute(context.Background(), meetingInput(map[string]any{
		"setVar": "meeting_url",
	}))
	if err == nil {
		t.Fatal("expected error when Create fails")
	}
}

func TestCreateMeeting_WithVariables(t *testing.T) {
	op := New(&fakeMeetingClient{url: "https://meet.example.com/abc"})
	out, err := op.Execute(context.Background(), meetingInput(map[string]any{
		"setVar":    "link",
		"variables": map[string]any{"caller": "${caller_id_number}"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["link"] == "" {
		t.Error("expected link to be set")
	}
}
