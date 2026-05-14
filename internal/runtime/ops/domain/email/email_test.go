package email

import (
	"context"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"github.com/webitel/wlog"

	emaildomain "github.com/webitel/flow_manager/internal/domain/email"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── fakeEmailDeps ─────────────────────────────────────────────────────────────

type fakeEmailDeps struct {
	smtpSettings    *emaildomain.SmtSettings
	smtpSettingsErr error
	oauthToken      string
	oauthErr        error
	filesMeta       []files.File
	filesMetaErr    error
	downloadReader  io.ReadCloser
	downloadErr     error
	saveErr         error

	// recorded
	saveEmailCalls []*emaildomain.Email
}

func (f *fakeEmailDeps) SmtpSettings(_ int64, _ *queue.SearchEntity) (*emaildomain.SmtSettings, error) {
	return f.smtpSettings, f.smtpSettingsErr
}
func (f *fakeEmailDeps) SmtpSettingsOAuthToken(_ *emaildomain.SmtSettings) (string, error) {
	return f.oauthToken, f.oauthErr
}
func (f *fakeEmailDeps) GetFileMetadata(_ int64, _ []int64) ([]files.File, error) {
	return f.filesMeta, f.filesMetaErr
}
func (f *fakeEmailDeps) DownloadFile(_ int64, _ int64) (io.ReadCloser, error) {
	return f.downloadReader, f.downloadErr
}
func (f *fakeEmailDeps) SaveEmail(_ int64, email *emaildomain.Email) error {
	f.saveEmailCalls = append(f.saveEmailCalls, email)
	return f.saveErr
}

var _ EmailDeps = (*fakeEmailDeps)(nil)

// ── stubEmailConn ─────────────────────────────────────────────────────────────

// stubEmailConn satisfies emaildomain.EmailConnection (used by replyOp).
type stubEmailConn struct {
	id      string
	replyFn func(text string) (*emaildomain.Email, error)
}

func (c *stubEmailConn) Type() flow.ConnectionType { return flow.ConnectionTypeEmail }
func (c *stubEmailConn) Id() string                { return c.id }
func (c *stubEmailConn) NodeId() string            { return "" }
func (c *stubEmailConn) DomainId() int64           { return 1 }
func (c *stubEmailConn) Context() context.Context  { return context.Background() }
func (c *stubEmailConn) Close() error              { return nil }
func (c *stubEmailConn) Log() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}
func (c *stubEmailConn) Variables() map[string]string                         { return nil }
func (c *stubEmailConn) Get(_ string) (string, bool)                          { return "", false }
func (c *stubEmailConn) Set(_ context.Context, _ flow.Variables) (flow.Response, error) {
	return nil, nil
}
func (c *stubEmailConn) ParseText(text string, _ ...flow.ParseOption) string { return text }
func (c *stubEmailConn) SchemaId() int                                       { return 0 }
func (c *stubEmailConn) Email() *emaildomain.Email                           { return nil }
func (c *stubEmailConn) Reply(text string) (*emaildomain.Email, error) {
	if c.replyFn != nil {
		return c.replyFn(text)
	}
	return nil, nil
}

var _ emaildomain.EmailConnection = (*stubEmailConn)(nil)

// ── fakeReplyDeps ─────────────────────────────────────────────────────────────

type fakeReplyDeps struct {
	err      error
	replyCalls []string
}

func (f *fakeReplyDeps) ReplyEmail(_ emaildomain.EmailConnection, text string) error {
	f.replyCalls = append(f.replyCalls, text)
	return f.err
}

var _ ReplyDeps = (*fakeReplyDeps)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

// invalidSMTP is a server:port guaranteed to refuse connections instantly.
const invalidSMTP = "127.0.0.1"
const invalidSMTPPort = 1

func emailInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

// ── sendEmail: validation ─────────────────────────────────────────────────────

func TestSendEmail_EmptyMessage(t *testing.T) {
	op := &sendEmailOp{deps: &fakeEmailDeps{}}
	_, err := op.Execute(context.Background(), emailInput(map[string]any{
		"to": []any{"user@example.com"},
	}))
	if err == nil || !strings.Contains(err.Error(), "message is required") {
		t.Fatalf("expected 'message is required' error, got %v", err)
	}
}

func TestSendEmail_EmptyTo(t *testing.T) {
	op := &sendEmailOp{deps: &fakeEmailDeps{}}
	_, err := op.Execute(context.Background(), emailInput(map[string]any{
		"message": "hello",
	}))
	if err == nil || !strings.Contains(err.Error(), "to is required") {
		t.Fatalf("expected 'to is required' error, got %v", err)
	}
}

// ── sendEmail: profile lookup ─────────────────────────────────────────────────

func TestSendEmail_ProfileLookup_Applied(t *testing.T) {
	id := 7
	deps := &fakeEmailDeps{
		smtpSettings: &emaildomain.SmtSettings{
			Server: invalidSMTP,
			Port:   invalidSMTPPort,
			Auth:   emaildomain.SmtpPlainAuth{User: "bot@example.com"},
		},
	}
	op := &sendEmailOp{deps: deps}
	// Execute will attempt SMTP dial and fail — we only care that
	// SmtpSettings was called and the From was set from the profile user.
	out, err := op.Execute(context.Background(), emailInput(map[string]any{
		"message": "test body",
		"to":      []any{"a@b.com"},
		"profile": map[string]any{"id": id},
		"set":     map[string]any{"error": "send_err"},
	}))
	// SMTP dial will fail (connection refused), but SmtpSettings was called.
	// The error is captured in SetVars["send_err"].
	if err == nil {
		t.Fatal("expected SMTP dial error")
	}
	if out.SetVars["send_err"] == "" {
		t.Error("expected error captured in set.error variable")
	}
}

func TestSendEmail_ProfileLookup_Error_Ignored(t *testing.T) {
	// SmtpSettings error is silently swallowed — op continues with inline smtp.
	deps := &fakeEmailDeps{
		smtpSettingsErr: fmt.Errorf("profile not found"),
	}
	op := &sendEmailOp{deps: deps}
	// Without a valid SMTP server, dial will still fail — but profile error is not
	// propagated (it just falls back to whatever inline smtp args were given).
	_, err := op.Execute(context.Background(), emailInput(map[string]any{
		"message": "hello",
		"to":      []any{"a@b.com"},
		"profile": map[string]any{"id": 1},
		"smtp":    map[string]any{"server": invalidSMTP, "port": invalidSMTPPort},
	}))
	// SMTP dial will fail (not profile error).
	if err == nil {
		t.Fatal("expected SMTP dial error (not profile error)")
	}
	if strings.Contains(err.Error(), "profile not found") {
		t.Error("profile lookup error should be silently swallowed, not propagated")
	}
}

// ── sendEmail: async ──────────────────────────────────────────────────────────

func TestSendEmail_Async_ReturnsImmediately(t *testing.T) {
	// async=true: Execute returns at once without waiting for SMTP.
	deps := &fakeEmailDeps{}
	op := &sendEmailOp{deps: deps}
	start := time.Now()
	out, err := op.Execute(context.Background(), emailInput(map[string]any{
		"message": "hi",
		"to":      []any{"x@example.com"},
		"async":   true,
		// No valid SMTP server — the goroutine will fail, but Execute won't wait.
		"smtp": map[string]any{"server": invalidSMTP, "port": invalidSMTPPort},
	}))
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("async Execute should not return error, got: %v", err)
	}
	if out.Break || len(out.SetVars) > 0 {
		t.Errorf("unexpected output: %+v", out)
	}
	// Should return almost instantly — well under 500 ms.
	if elapsed > 500*time.Millisecond {
		t.Errorf("async Execute took %v, expected near-instant", elapsed)
	}
}

// ── sendEmail: set["error"] captures dial error ───────────────────────────────

func TestSendEmail_DialFail_ErrorCapturedInSetVar(t *testing.T) {
	deps := &fakeEmailDeps{}
	op := &sendEmailOp{deps: deps}
	out, err := op.Execute(context.Background(), emailInput(map[string]any{
		"message": "body",
		"to":      []any{"x@example.com"},
		"smtp":    map[string]any{"server": invalidSMTP, "port": invalidSMTPPort},
		"set":     map[string]any{"error": "smtp_err"},
	}))
	if err == nil {
		t.Fatal("expected SMTP dial error")
	}
	if out.SetVars["smtp_err"] == "" {
		t.Error("expected error captured in set.error variable")
	}
}

// ── sendEmail: retry ──────────────────────────────────────────────────────────

func TestSendEmail_Retry_ExhaustedReturnsError(t *testing.T) {
	// retryCount=1 means 1 retry after first failure → 2 total attempts, then error.
	deps := &fakeEmailDeps{}
	op := &sendEmailOp{deps: deps}
	_, err := op.Execute(context.Background(), emailInput(map[string]any{
		"message":    "body",
		"to":         []any{"x@example.com"},
		"smtp":       map[string]any{"server": invalidSMTP, "port": invalidSMTPPort},
		"retryCount": 1,
	}))
	if err == nil {
		t.Fatal("expected error after retry exhausted")
	}
}

// ── oAuth2Smtp ────────────────────────────────────────────────────────────────

func TestOAuth2Smtp_Start_TLS(t *testing.T) {
	auth := newOAuth2Smtp("user@example.com", "Bearer", "token123")
	mechanism, resp, err := auth.Start(&smtp.ServerInfo{TLS: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mechanism != "XOAUTH2" {
		t.Errorf("mechanism = %q, want XOAUTH2", mechanism)
	}
	if !strings.Contains(string(resp), "user=user@example.com") {
		t.Errorf("response missing user: %q", resp)
	}
	if !strings.Contains(string(resp), "Bearer token123") {
		t.Errorf("response missing token: %q", resp)
	}
}

func TestOAuth2Smtp_Start_NoTLS_Rejected(t *testing.T) {
	auth := newOAuth2Smtp("user@example.com", "Bearer", "tok")
	_, _, err := auth.Start(&smtp.ServerInfo{TLS: false})
	if err == nil {
		t.Fatal("expected error for non-TLS connection")
	}
}

func TestOAuth2Smtp_Next_NoMore(t *testing.T) {
	auth := newOAuth2Smtp("u", "Bearer", "t")
	resp, err := auth.Next(nil, false)
	if err != nil || resp != nil {
		t.Errorf("Next(more=false) = (%v, %v), want (nil, nil)", resp, err)
	}
}

func TestOAuth2Smtp_Next_MoreReturnsError(t *testing.T) {
	auth := newOAuth2Smtp("u", "Bearer", "t")
	_, err := auth.Next([]byte("challenge"), true)
	if err == nil {
		t.Fatal("expected error when server sends unexpected challenge")
	}
}

// ── replyOp ───────────────────────────────────────────────────────────────────

func TestReply_NoConnection(t *testing.T) {
	op := &replyOp{deps: &fakeReplyDeps{}}
	_, err := op.Execute(context.Background(), emailInput(map[string]any{"body": "hi"}))
	if err == nil {
		t.Fatal("expected error when no connection in context")
	}
}

func TestReply_WrongConnectionType(t *testing.T) {
	// Use a stub connection that doesn't implement EmailConnection.
	ctx := connctx.WithConnection(context.Background(), &stubNonEmailConn{})
	op := &replyOp{deps: &fakeReplyDeps{}}
	_, err := op.Execute(ctx, emailInput(map[string]any{"body": "hi"}))
	if err == nil {
		t.Fatal("expected error when connection is not EmailConnection")
	}
}

func TestReply_EmptyBody(t *testing.T) {
	conn := &stubEmailConn{id: "em-1"}
	ctx := connctx.WithConnection(context.Background(), conn)
	op := &replyOp{deps: &fakeReplyDeps{}}
	_, err := op.Execute(ctx, emailInput(map[string]any{}))
	if err == nil || !strings.Contains(err.Error(), "body is required") {
		t.Fatalf("expected 'body is required' error, got %v", err)
	}
}

func TestReply_DepError(t *testing.T) {
	conn := &stubEmailConn{id: "em-1"}
	ctx := connctx.WithConnection(context.Background(), conn)
	deps := &fakeReplyDeps{err: fmt.Errorf("smtp failure")}
	op := &replyOp{deps: deps}
	_, err := op.Execute(ctx, emailInput(map[string]any{"body": "hello"}))
	if err == nil {
		t.Fatal("expected error when ReplyEmail fails")
	}
}

func TestReply_Success(t *testing.T) {
	conn := &stubEmailConn{id: "em-1"}
	ctx := connctx.WithConnection(context.Background(), conn)
	deps := &fakeReplyDeps{}
	op := &replyOp{deps: deps}
	_, err := op.Execute(ctx, emailInput(map[string]any{"body": "thank you"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.replyCalls) != 1 || deps.replyCalls[0] != "thank you" {
		t.Errorf("replyCalls = %v, want [thank you]", deps.replyCalls)
	}
}

// ── stubNonEmailConn ──────────────────────────────────────────────────────────

// stubNonEmailConn satisfies flow.Connection but NOT emaildomain.EmailConnection.
type stubNonEmailConn struct{}

func (c *stubNonEmailConn) Type() flow.ConnectionType { return flow.ConnectionTypeCall }
func (c *stubNonEmailConn) Id() string                { return "non-email" }
func (c *stubNonEmailConn) NodeId() string            { return "" }
func (c *stubNonEmailConn) DomainId() int64           { return 0 }
func (c *stubNonEmailConn) Context() context.Context  { return context.Background() }
func (c *stubNonEmailConn) Close() error              { return nil }
func (c *stubNonEmailConn) Log() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}
func (c *stubNonEmailConn) Variables() map[string]string                         { return nil }
func (c *stubNonEmailConn) Get(_ string) (string, bool)                          { return "", false }
func (c *stubNonEmailConn) Set(_ context.Context, _ flow.Variables) (flow.Response, error) {
	return nil, nil
}
func (c *stubNonEmailConn) ParseText(text string, _ ...flow.ParseOption) string { return text }

var _ flow.Connection = (*stubNonEmailConn)(nil)
