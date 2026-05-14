package im

import (
	"context"
	"fmt"
	"testing"

	"github.com/webitel/wlog"

	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── sendStubDialog ────────────────────────────────────────────────────────────

// sendStubDialog satisfies chatdomain.IMDialog with recordable send methods.
// Fields OnSendMessage / OnSendTextMessage / etc. are called if set;
// otherwise they succeed silently and record the call.
type sendStubDialog struct {
	id       string
	domainId int64

	// Recorded calls
	sentMessages  []chatdomain.ChatMessageOutbound
	sentTexts     []string
	sentImages    []chatdomain.ChatMessageOutbound
	sentDocuments []chatdomain.ChatMessageOutbound

	// Optional error injection
	sendMessageErr  error
	sendTextErr     error
	sendImageErr    error
	sendDocumentErr error
}

// flow.Connection
func (d *sendStubDialog) Type() flow.ConnectionType     { return flow.ConnectionTypeIM }
func (d *sendStubDialog) Id() string                     { return d.id }
func (d *sendStubDialog) NodeId() string                 { return "" }
func (d *sendStubDialog) DomainId() int64                { return d.domainId }
func (d *sendStubDialog) Context() context.Context       { return context.Background() }
func (d *sendStubDialog) Close() error                   { return nil }
func (d *sendStubDialog) Log() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}
func (d *sendStubDialog) Variables() map[string]string      { return nil }
func (d *sendStubDialog) Get(_ string) (string, bool)       { return "", false }
func (d *sendStubDialog) Set(_ context.Context, _ flow.Variables) (flow.Response, error) {
	return nil, nil
}
func (d *sendStubDialog) ParseText(text string, _ ...flow.ParseOption) string { return text }

// chatdomain.IMDialog
func (d *sendStubDialog) ThreadId() string                     { return "" }
func (d *sendStubDialog) From() chatdomain.ImEndpoint          { return chatdomain.ImEndpoint{} }
func (d *sendStubDialog) To() chatdomain.ImEndpoint            { return chatdomain.ImEndpoint{} }
func (d *sendStubDialog) LastMessage() chatdomain.Message      { return chatdomain.Message{} }
func (d *sendStubDialog) SchemaId() int                        { return 0 }
func (d *sendStubDialog) Stop(_ error)                         {}
func (d *sendStubDialog) IsTransfer() bool                     { return false }
func (d *sendStubDialog) DumpExportVariables() map[string]string { return nil }
func (d *sendStubDialog) GetQueueKey() *queue.InQueueKey       { return nil }
func (d *sendStubDialog) SetQueue(_ *queue.InQueueKey) bool    { return false }
func (d *sendStubDialog) Export(_ context.Context, _ []string) (flow.Response, error) {
	return nil, nil
}
func (d *sendStubDialog) UnSet(_ context.Context, _ []string) (flow.Response, error) {
	return nil, nil
}
func (d *sendStubDialog) LastMessages(_ int) []chatdomain.ChatMessage { return nil }
func (d *sendStubDialog) SendMenu(_ context.Context, _ *chatdomain.ChatMenuArgs) (flow.Response, error) {
	return nil, nil
}

func (d *sendStubDialog) SendMessage(_ context.Context, msg chatdomain.ChatMessageOutbound) (flow.Response, error) {
	if d.sendMessageErr != nil {
		return nil, d.sendMessageErr
	}
	d.sentMessages = append(d.sentMessages, msg)
	return nil, nil
}

func (d *sendStubDialog) SendTextMessage(_ context.Context, text string) (flow.Response, error) {
	if d.sendTextErr != nil {
		return nil, d.sendTextErr
	}
	d.sentTexts = append(d.sentTexts, text)
	return nil, nil
}

func (d *sendStubDialog) SendImageMessage(_ context.Context, msg chatdomain.ChatMessageOutbound) (flow.Response, error) {
	if d.sendImageErr != nil {
		return nil, d.sendImageErr
	}
	d.sentImages = append(d.sentImages, msg)
	return nil, nil
}

func (d *sendStubDialog) SendDocumentMessage(_ context.Context, msg chatdomain.ChatMessageOutbound) (flow.Response, error) {
	if d.sendDocumentErr != nil {
		return nil, d.sendDocumentErr
	}
	d.sentDocuments = append(d.sentDocuments, msg)
	return nil, nil
}

func (d *sendStubDialog) SendFile(_ context.Context, _ string, _ *files.File, _ string) (flow.Response, error) {
	return nil, nil
}

var _ chatdomain.IMDialog = (*sendStubDialog)(nil)

// ── fakeSendDeps ──────────────────────────────────────────────────────────────

type fakeSendDeps struct {
	searchResult *files.File
	searchErr    error
	setupResult  *files.File
	setupErr     error
	actionErr    error

	// recorded
	searchCalls []files.SearchFile
	setupCalls  []files.File
	actionCalls []chatdomain.ChatAction
}

func (f *fakeSendDeps) SearchMediaFile(_ int64, search *files.SearchFile) (*files.File, error) {
	f.searchCalls = append(f.searchCalls, *search)
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if f.searchResult != nil {
		return f.searchResult, nil
	}
	return &files.File{Id: search.Id, Name: search.Name}, nil
}

func (f *fakeSendDeps) SetupPublicFileUrl(file *files.File, _ int64, _, _ string, _ int64) (*files.File, error) {
	f.setupCalls = append(f.setupCalls, *file)
	if f.setupErr != nil {
		return nil, f.setupErr
	}
	if f.setupResult != nil {
		return f.setupResult, nil
	}
	result := *file
	result.PublicUrl = "https://cdn.example.com/" + file.Name
	return &result, nil
}

func (f *fakeSendDeps) SenChatAction(_ context.Context, _ string, action chatdomain.ChatAction) error {
	f.actionCalls = append(f.actionCalls, action)
	return f.actionErr
}

var _ SendDeps = (*fakeSendDeps)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func ctxWithSendDialog(d chatdomain.IMDialog) context.Context {
	return connctx.WithConnection(context.Background(), d)
}

func sendInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

func sendInputWithVars(args map[string]any, vars map[string]string) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, Variables: vars, DomainID: 1}
}

// ── resolveServer tests ───────────────────────────────────────────────────────

func TestResolveServer(t *testing.T) {
	cases := []struct {
		file, fallback, want string
	}{
		{"https://files.local/", "", "https://files.local"},
		{"", "https://default.local/", "https://default.local"},
		{"https://files.local", "https://default.local", "https://files.local"},
		{"", "", ""},
	}
	for _, tc := range cases {
		got := resolveServer(tc.file, tc.fallback)
		if got != tc.want {
			t.Errorf("resolveServer(%q, %q) = %q, want %q", tc.file, tc.fallback, got, tc.want)
		}
	}
}

// ── sendText tests ────────────────────────────────────────────────────────────

func TestSendText_NoDialog(t *testing.T) {
	op := &sendTextOp{}
	_, err := op.Execute(context.Background(), ops.OpInput{Node: &tree.Node{RawArgs: "hi"}})
	if err == nil {
		t.Fatal("expected error when no dialog in context")
	}
}

func TestSendText_SendsText(t *testing.T) {
	dialog := &sendStubDialog{}
	op := &sendTextOp{}
	in := ops.OpInput{Node: &tree.Node{RawArgs: "hello world"}}
	_, err := op.Execute(ctxWithSendDialog(dialog), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dialog.sentTexts) != 1 || dialog.sentTexts[0] != "hello world" {
		t.Errorf("sentTexts = %v, want [hello world]", dialog.sentTexts)
	}
}

func TestSendText_VariableExpansion(t *testing.T) {
	dialog := &sendStubDialog{}
	op := &sendTextOp{}
	in := ops.OpInput{
		Node:      &tree.Node{RawArgs: "Hi ${name}!"},
		Variables: map[string]string{"name": "Alice"},
	}
	_, err := op.Execute(ctxWithSendDialog(dialog), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dialog.sentTexts) != 1 || dialog.sentTexts[0] != "Hi Alice!" {
		t.Errorf("sentTexts = %v, want [Hi Alice!]", dialog.sentTexts)
	}
}

func TestSendText_DepError(t *testing.T) {
	dialog := &sendStubDialog{sendTextErr: fmt.Errorf("channel closed")}
	op := &sendTextOp{}
	_, err := op.Execute(ctxWithSendDialog(dialog), ops.OpInput{Node: &tree.Node{RawArgs: "hi"}})
	if err == nil {
		t.Fatal("expected error when SendTextMessage fails")
	}
}

// ── sendMessage tests ─────────────────────────────────────────────────────────

func TestSendMessage_NoDialog(t *testing.T) {
	op := &sendMessageOp{deps: &fakeSendDeps{}}
	_, err := op.Execute(context.Background(), sendInput(nil))
	if err == nil {
		t.Fatal("expected error when no dialog in context")
	}
}

func TestSendMessage_NoFile(t *testing.T) {
	dialog := &sendStubDialog{}
	op := &sendMessageOp{deps: &fakeSendDeps{}}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"text": "plain message",
		"type": "text",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dialog.sentMessages) != 1 {
		t.Fatalf("expected 1 SendMessage call, got %d", len(dialog.sentMessages))
	}
	if dialog.sentMessages[0].Text != "plain message" {
		t.Errorf("text = %q, want %q", dialog.sentMessages[0].Text, "plain message")
	}
}

func TestSendMessage_FileWithURL_SkipsMediaLookup(t *testing.T) {
	// When File.Url is set and File.Id == 0, the op sets Id=1 and skips
	// SearchMediaFile — the URL is already public.
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{}
	op := &sendMessageOp{deps: deps}

	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"url": "https://cdn.example.com/img.png", "id": 0},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.searchCalls) != 0 {
		t.Error("SearchMediaFile should not be called when File.Url is set")
	}
	if len(dialog.sentMessages) != 1 || dialog.sentMessages[0].File.Id != 1 {
		t.Errorf("expected File.Id=1, got %+v", dialog.sentMessages)
	}
}

func TestSendMessage_FileWithoutURL_MediaLookup(t *testing.T) {
	// File without Url triggers SearchMediaFile + SetupPublicFileUrl.
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{
		setupResult: &files.File{Id: 5, Name: "doc.pdf", PublicUrl: "https://cdn/doc.pdf"},
	}
	op := &sendMessageOp{deps: deps}

	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"id": 5, "name": "doc.pdf"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.searchCalls) != 1 {
		t.Errorf("SearchMediaFile called %d times, want 1", len(deps.searchCalls))
	}
	if len(deps.setupCalls) != 1 {
		t.Errorf("SetupPublicFileUrl called %d times, want 1", len(deps.setupCalls))
	}
	// Type is forced to "file" when going through media lookup.
	if len(dialog.sentMessages) != 1 || dialog.sentMessages[0].Type != "file" {
		t.Errorf("expected Type=file, got %+v", dialog.sentMessages)
	}
}

func TestSendMessage_SearchMediaError(t *testing.T) {
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{searchErr: fmt.Errorf("storage unavailable")}
	op := &sendMessageOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"id": 1},
	}))
	if err == nil {
		t.Fatal("expected error when SearchMediaFile fails")
	}
}

func TestSendMessage_SetupURLError(t *testing.T) {
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{setupErr: fmt.Errorf("cdn error")}
	op := &sendMessageOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"id": 2},
	}))
	if err == nil {
		t.Fatal("expected error when SetupPublicFileUrl fails")
	}
}

func TestSendMessage_SendError(t *testing.T) {
	dialog := &sendStubDialog{sendMessageErr: fmt.Errorf("channel closed")}
	op := &sendMessageOp{deps: &fakeSendDeps{}}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{"text": "hi"}))
	if err == nil {
		t.Fatal("expected error when SendMessage fails")
	}
}

// ── sendImage tests ───────────────────────────────────────────────────────────

func TestSendImage_NoDialog(t *testing.T) {
	op := &sendImageOp{deps: &fakeSendDeps{}}
	_, err := op.Execute(context.Background(), sendInput(nil))
	if err == nil {
		t.Fatal("expected error when no dialog in context")
	}
}

func TestSendImage_NoFile(t *testing.T) {
	dialog := &sendStubDialog{}
	op := &sendImageOp{deps: &fakeSendDeps{}}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{"text": "caption"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dialog.sentImages) != 1 {
		t.Errorf("expected 1 SendImageMessage call, got %d", len(dialog.sentImages))
	}
}

func TestSendImage_FileWithURL_SkipsLookup(t *testing.T) {
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{}
	op := &sendImageOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"url": "https://cdn/img.jpg", "id": 0},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.searchCalls) != 0 {
		t.Error("SearchMediaFile should not be called when File.Url is set")
	}
}

func TestSendImage_SearchError(t *testing.T) {
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{searchErr: fmt.Errorf("not found")}
	op := &sendImageOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"id": 3},
	}))
	if err == nil {
		t.Fatal("expected error when SearchMediaFile fails")
	}
}

// ── sendFile tests ────────────────────────────────────────────────────────────

func TestSendFile_NoDialog(t *testing.T) {
	op := &sendFileOp{deps: &fakeSendDeps{}}
	_, err := op.Execute(context.Background(), sendInput(nil))
	if err == nil {
		t.Fatal("expected error when no dialog in context")
	}
}

func TestSendFile_FileWithURL_SkipsLookup(t *testing.T) {
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{}
	op := &sendFileOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"url": "https://cdn/doc.zip", "id": 0},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.searchCalls) != 0 {
		t.Error("SearchMediaFile should not be called when File.Url is set")
	}
	if len(dialog.sentDocuments) != 1 {
		t.Errorf("expected 1 SendDocumentMessage call, got %d", len(dialog.sentDocuments))
	}
}

func TestSendFile_MediaLookup(t *testing.T) {
	dialog := &sendStubDialog{}
	deps := &fakeSendDeps{}
	op := &sendFileOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"file": map[string]any{"id": 7, "name": "report.pdf"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.searchCalls) != 1 || deps.searchCalls[0].Id != 7 {
		t.Errorf("SearchMediaFile not called correctly: %v", deps.searchCalls)
	}
}

// ── sendAction tests ──────────────────────────────────────────────────────────

func TestSendAction_NoDialog(t *testing.T) {
	op := &sendActionOp{deps: &fakeSendDeps{}}
	_, err := op.Execute(context.Background(), sendInput(map[string]any{"action": "typing"}))
	if err == nil {
		t.Fatal("expected error when no dialog in context")
	}
}

func TestSendAction_Success(t *testing.T) {
	dialog := &sendStubDialog{id: "ch-42"}
	deps := &fakeSendDeps{}
	op := &sendActionOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{
		"action": "typing",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.actionCalls) != 1 || deps.actionCalls[0] != chatdomain.ChatActionTyping {
		t.Errorf("actionCalls = %v, want [typing]", deps.actionCalls)
	}
}

func TestSendAction_DepError(t *testing.T) {
	dialog := &sendStubDialog{id: "ch-1"}
	deps := &fakeSendDeps{actionErr: fmt.Errorf("grpc error")}
	op := &sendActionOp{deps: deps}
	_, err := op.Execute(ctxWithSendDialog(dialog), sendInput(map[string]any{"action": "typing"}))
	if err == nil {
		t.Fatal("expected error when SenChatAction fails")
	}
}

// ── rawStringSlice tests ──────────────────────────────────────────────────────

func TestRawStringSlice_Array(t *testing.T) {
	in := ops.OpInput{
		Node:      &tree.Node{RawArgs: []any{"a", "b", "c"}},
		Variables: map[string]string{},
	}
	got := rawStringSlice(in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRawStringSlice_String(t *testing.T) {
	in := ops.OpInput{Node: &tree.Node{RawArgs: "single"}}
	got := rawStringSlice(in)
	if len(got) != 1 || got[0] != "single" {
		t.Errorf("got %v, want [single]", got)
	}
}

func TestRawStringSlice_VariableExpansion(t *testing.T) {
	in := ops.OpInput{
		Node:      &tree.Node{RawArgs: []any{"Hello ${name}"}},
		Variables: map[string]string{"name": "World"},
	}
	got := rawStringSlice(in)
	if len(got) != 1 || got[0] != "Hello World" {
		t.Errorf("got %v, want [Hello World]", got)
	}
}

func TestRawStringSlice_Empty(t *testing.T) {
	in := ops.OpInput{Node: &tree.Node{RawArgs: nil}}
	got := rawStringSlice(in)
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}

// Suppress "imported and not used" for sendInputWithVars (used by caller).
var _ = sendInputWithVars
