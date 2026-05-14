package chat

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/webitel/wlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	proto_chat "github.com/webitel/flow_manager/api/gen/chat"
	ai_bots "github.com/webitel/flow_manager/api/gen/ai_bots"
	"github.com/webitel/flow_manager/api/gen/cc"
	chatdomain "github.com/webitel/flow_manager/internal/domain/chat"
	"github.com/webitel/flow_manager/internal/domain/files"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/domain/queue"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/ops/domain/messaging"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── stubConversation ──────────────────────────────────────────────────────────

// stubConversation satisfies chatdomain.Conversation with recordable methods.
type stubConversation struct {
	id       string
	domainId int64
	vars     map[string]string

	// Recorded calls
	sentMessages  []chatdomain.ChatMessageOutbound
	sentTexts     []string
	sentImages    []string // URL args
	sentFiles     []string // text args
	sentMenus     []*chatdomain.ChatMenuArgs
	bridgeCalls   []int64 // userIds
	exportCalls   [][]string
	unSetCalls    [][]string
	setVarsCalls  []flow.Variables
	queueKey      *queue.InQueueKey
	setQueueCalls []*queue.InQueueKey

	// Error injection
	sendMessageErr error
	sendTextErr    error
	sendMenuErr    error
	sendImageErr   error
	sendFileErr    error
	bridgeErr      error
	exportErr      error
	unSetErr       error
}

// flow.Connection
func (c *stubConversation) Type() flow.ConnectionType { return flow.ConnectionTypeChat }
func (c *stubConversation) Id() string                { return c.id }
func (c *stubConversation) NodeId() string            { return "" }
func (c *stubConversation) DomainId() int64           { return c.domainId }
func (c *stubConversation) Context() context.Context  { return context.Background() }
func (c *stubConversation) Close() error              { return nil }
func (c *stubConversation) Log() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}
func (c *stubConversation) Variables() map[string]string { return c.vars }
func (c *stubConversation) Get(key string) (string, bool) {
	v, ok := c.vars[key]
	return v, ok
}
func (c *stubConversation) Set(_ context.Context, vars flow.Variables) (flow.Response, error) {
	c.setVarsCalls = append(c.setVarsCalls, vars)
	if c.vars == nil {
		c.vars = make(map[string]string)
	}
	for k, v := range vars {
		c.vars[k] = fmt.Sprintf("%v", v)
	}
	return nil, nil
}
func (c *stubConversation) ParseText(text string, _ ...flow.ParseOption) string { return text }

// Conversation-specific
func (c *stubConversation) ProfileId() int64  { return 0 }
func (c *stubConversation) NodeName() string   { return "" }
func (c *stubConversation) SchemaId() int32    { return 0 }
func (c *stubConversation) UserId() int64      { return 0 }
func (c *stubConversation) BreakCause() string { return "" }
func (c *stubConversation) IsTransfer() bool   { return false }
func (c *stubConversation) DumpExportVariables() map[string]string {
	out := make(map[string]string, len(c.vars))
	for k, v := range c.vars {
		out[k] = v
	}
	return out
}

func (c *stubConversation) Stop(_ error, _ proto_chat.CloseConversationCause) {}

func (c *stubConversation) GetQueueKey() *queue.InQueueKey { return c.queueKey }
func (c *stubConversation) SetQueue(k *queue.InQueueKey) bool {
	c.setQueueCalls = append(c.setQueueCalls, k)
	c.queueKey = k
	return true
}

func (c *stubConversation) SendMessage(_ context.Context, msg chatdomain.ChatMessageOutbound) (flow.Response, error) {
	if c.sendMessageErr != nil {
		return nil, c.sendMessageErr
	}
	c.sentMessages = append(c.sentMessages, msg)
	return nil, nil
}

func (c *stubConversation) SendTextMessage(_ context.Context, text string) (flow.Response, error) {
	if c.sendTextErr != nil {
		return nil, c.sendTextErr
	}
	c.sentTexts = append(c.sentTexts, text)
	return nil, nil
}

func (c *stubConversation) SendMenu(_ context.Context, menu *chatdomain.ChatMenuArgs) (flow.Response, error) {
	if c.sendMenuErr != nil {
		return nil, c.sendMenuErr
	}
	c.sentMenus = append(c.sentMenus, menu)
	return nil, nil
}

func (c *stubConversation) SendImageMessage(_ context.Context, imgURL, _, _, _ string) (flow.Response, error) {
	if c.sendImageErr != nil {
		return nil, c.sendImageErr
	}
	c.sentImages = append(c.sentImages, imgURL)
	return nil, nil
}

func (c *stubConversation) SendFile(_ context.Context, text string, _ *files.File, _ string) (flow.Response, error) {
	if c.sendFileErr != nil {
		return nil, c.sendFileErr
	}
	c.sentFiles = append(c.sentFiles, text)
	return nil, nil
}

func (c *stubConversation) Bridge(_ context.Context, userId int64, _ int) error {
	if c.bridgeErr != nil {
		return c.bridgeErr
	}
	c.bridgeCalls = append(c.bridgeCalls, userId)
	return nil
}

func (c *stubConversation) Export(_ context.Context, vars []string) (flow.Response, error) {
	if c.exportErr != nil {
		return nil, c.exportErr
	}
	c.exportCalls = append(c.exportCalls, vars)
	return nil, nil
}

func (c *stubConversation) UnSet(_ context.Context, keys []string) (flow.Response, error) {
	if c.unSetErr != nil {
		return nil, c.unSetErr
	}
	c.unSetCalls = append(c.unSetCalls, keys)
	return nil, nil
}

func (c *stubConversation) LastMessages(_ int) []chatdomain.ChatMessage { return nil }
func (c *stubConversation) ReceiveMessage(_ context.Context, _ string, _, _ int) ([]string, error) {
	panic("not called in these tests")
}
func (c *stubConversation) Bot(_ context.Context, _ ai_bots.ConverseServiceClient, _ string) (flow.Response, error) {
	panic("not called in these tests")
}

var _ chatdomain.Conversation = (*stubConversation)(nil)

// ── fakeQueueStream ───────────────────────────────────────────────────────────

type fakeChatQueueStream struct {
	events []*cc.QueueEvent
	idx    int
	err    error
}

func (f *fakeChatQueueStream) RecvMsg(m interface{}) error {
	if f.err != nil {
		return f.err
	}
	if f.idx >= len(f.events) {
		return io.EOF
	}
	ev := f.events[f.idx]
	f.idx++
	proto.Merge(m.(proto.Message), ev)
	return nil
}
func (f *fakeChatQueueStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeChatQueueStream) Trailer() metadata.MD         { return nil }
func (f *fakeChatQueueStream) CloseSend() error             { return nil }
func (f *fakeChatQueueStream) Context() context.Context     { return context.Background() }
func (f *fakeChatQueueStream) SendMsg(_ interface{}) error  { return nil }
func (f *fakeChatQueueStream) Recv() (*cc.QueueEvent, error) {
	var ev cc.QueueEvent
	if err := f.RecvMsg(&ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

var _ grpc.ClientStream = (*fakeChatQueueStream)(nil)
var _ cc.MemberService_ChatJoinToQueueClient = (*fakeChatQueueStream)(nil)

// ── fakeQueueDeps ─────────────────────────────────────────────────────────────

type fakeChatQueueDeps struct {
	cancelErr    error
	findQueueId  int32
	findQueueErr error
	agentId      *int32
	joinStream   cc.MemberService_ChatJoinToQueueClient
	joinErr      error
}

func (f *fakeChatQueueDeps) CancelAttempt(_ context.Context, _ queue.InQueueKey, _ string) error {
	return f.cancelErr
}
func (f *fakeChatQueueDeps) FindQueueByName(_ int64, _ string) (int32, error) {
	return f.findQueueId, f.findQueueErr
}
func (f *fakeChatQueueDeps) GetAgentIdByExtension(_ int64, _ string) (*int32, error) {
	return f.agentId, nil
}
func (f *fakeChatQueueDeps) JoinChatToInboundQueue(_ context.Context, _ *cc.ChatJoinToQueueRequest) (cc.MemberService_ChatJoinToQueueClient, error) {
	return f.joinStream, f.joinErr
}

var _ QueueDeps = (*fakeChatQueueDeps)(nil)

// ── fakeChatDeps ──────────────────────────────────────────────────────────────

type fakeChatDeps struct {
	profileType      string
	profileTypeErr   error
	broadcastResp    *chatdomain.BroadcastChatResponse
	broadcastErr     error
	messagesResult   *[]chatdomain.ChatMessage
	messagesErr      error
	parsedText       string
	parseErr         error
}

func (f *fakeChatDeps) ChatProfileType(_ int64, _ int) (string, error) {
	return f.profileType, f.profileTypeErr
}
func (f *fakeChatDeps) BroadcastChatMessage(_ context.Context, _ int64, _ chatdomain.BroadcastChat, _ []chatdomain.BroadcastPeer) (*chatdomain.BroadcastChatResponse, error) {
	return f.broadcastResp, f.broadcastErr
}
func (f *fakeChatDeps) GetChatMessagesByConversationId(_ context.Context, _ int64, _ string, _ int64) (*[]chatdomain.ChatMessage, error) {
	return f.messagesResult, f.messagesErr
}
func (f *fakeChatDeps) ParseChatMessages(_ *[]chatdomain.ChatMessage, _ string) (string, error) {
	return f.parsedText, f.parseErr
}

var _ ChatDeps = (*fakeChatDeps)(nil)

// ── fakeSendDeps ──────────────────────────────────────────────────────────────

type fakeChatSendDeps struct {
	searchResult *files.File
	searchErr    error
	setupResult  *files.File
	setupErr     error
	actionErr    error
	ttsURI       string
	ttsErr       error

	searchCalls []files.SearchFile
}

func (f *fakeChatSendDeps) SearchMediaFile(_ int64, search *files.SearchFile) (*files.File, error) {
	f.searchCalls = append(f.searchCalls, *search)
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if f.searchResult != nil {
		return f.searchResult, nil
	}
	return &files.File{Id: search.Id, Name: search.Name}, nil
}
func (f *fakeChatSendDeps) SetupPublicFileUrl(file *files.File, _ int64, _, _ string, _ int64) (*files.File, error) {
	if f.setupErr != nil {
		return nil, f.setupErr
	}
	if f.setupResult != nil {
		return f.setupResult, nil
	}
	result := *file
	result.PublicUrl = "https://cdn/" + file.Name
	return &result, nil
}
func (f *fakeChatSendDeps) SenChatAction(_ context.Context, _ string, _ chatdomain.ChatAction) error {
	return f.actionErr
}
func (f *fakeChatSendDeps) GenerateTTSLink(_ context.Context, _ string, _ int64, _ int, _, _, _ string) (string, error) {
	return f.ttsURI, f.ttsErr
}

var _ SendDeps = (*fakeChatSendDeps)(nil)

// ── fakeSTTDeps ───────────────────────────────────────────────────────────────

type fakeSTTDeps struct {
	text string
	err  error
}

func (f *fakeSTTDeps) GetFileTranscription(_ context.Context, _, _, _ int64, _ string) (string, error) {
	return f.text, f.err
}

var _ STTDeps = (*fakeSTTDeps)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func ctxWithConv(conv chatdomain.Conversation) context.Context {
	return connctx.WithConnection(context.Background(), conv)
}

func chatInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

func chatInputWithVars(args map[string]any, vars map[string]string) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, Variables: vars, DomainID: 1}
}

// ── misc.go: bridge ───────────────────────────────────────────────────────────

func TestBridge_NoConv(t *testing.T) {
	_, err := bridgeOp{}.Execute(context.Background(), chatInput(nil))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBridge_Error(t *testing.T) {
	conv := &stubConversation{bridgeErr: fmt.Errorf("timeout")}
	_, err := bridgeOp{}.Execute(ctxWithConv(conv), chatInput(map[string]any{"userId": 7, "timeout": 5000}))
	if err == nil {
		t.Fatal("expected error when Bridge fails")
	}
}

func TestBridge_Success(t *testing.T) {
	conv := &stubConversation{}
	_, err := bridgeOp{}.Execute(ctxWithConv(conv), chatInput(map[string]any{"userId": 42, "timeout": 3000}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.bridgeCalls) != 1 || conv.bridgeCalls[0] != 42 {
		t.Errorf("bridgeCalls = %v, want [42]", conv.bridgeCalls)
	}
}

// ── misc.go: export ───────────────────────────────────────────────────────────

func TestExport_NoConv(t *testing.T) {
	_, err := exportOp{}.Execute(context.Background(), ops.OpInput{Node: &tree.Node{RawArgs: []any{"k"}}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExport_Success(t *testing.T) {
	conv := &stubConversation{}
	in := ops.OpInput{Node: &tree.Node{RawArgs: []any{"var1", "var2"}}}
	_, err := exportOp{}.Execute(ctxWithConv(conv), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.exportCalls) != 1 || len(conv.exportCalls[0]) != 2 {
		t.Errorf("exportCalls = %v", conv.exportCalls)
	}
}

// ── misc.go: menu ─────────────────────────────────────────────────────────────

func TestMenu_NoConv(t *testing.T) {
	_, err := menuOp{}.Execute(context.Background(), chatInput(nil))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMenu_UnsupportedType(t *testing.T) {
	conv := &stubConversation{}
	_, err := menuOp{}.Execute(ctxWithConv(conv), chatInput(map[string]any{"type": "carousel"}))
	if err == nil {
		t.Fatal("expected error for unsupported menu type")
	}
}

func TestMenu_Success(t *testing.T) {
	conv := &stubConversation{}
	_, err := menuOp{}.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"type": "inline",
		"text": "pick one",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.sentMenus) != 1 || conv.sentMenus[0].Type != "inline" {
		t.Errorf("sentMenus = %v", conv.sentMenus)
	}
}

func TestMenu_SendError(t *testing.T) {
	conv := &stubConversation{sendMenuErr: fmt.Errorf("closed")}
	_, err := menuOp{}.Execute(ctxWithConv(conv), chatInput(map[string]any{"type": "buttons"}))
	if err == nil {
		t.Fatal("expected error when SendMenu fails")
	}
}

// ── misc.go: unSet ────────────────────────────────────────────────────────────

func TestUnSet_NoConv(t *testing.T) {
	_, err := unSetOp{}.Execute(context.Background(), ops.OpInput{Node: &tree.Node{RawArgs: []any{"k"}}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnSet_EmptyKeys(t *testing.T) {
	conv := &stubConversation{}
	_, err := unSetOp{}.Execute(ctxWithConv(conv), ops.OpInput{Node: &tree.Node{RawArgs: []any{}}})
	if err == nil {
		t.Fatal("expected error when no keys provided")
	}
}

func TestUnSet_Success(t *testing.T) {
	conv := &stubConversation{}
	_, err := unSetOp{}.Execute(ctxWithConv(conv), ops.OpInput{Node: &tree.Node{RawArgs: []any{"a", "b"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.unSetCalls) != 1 || len(conv.unSetCalls[0]) != 2 {
		t.Errorf("unSetCalls = %v", conv.unSetCalls)
	}
}

// ── op.go: broadcastChatMessage ───────────────────────────────────────────────

func TestBroadcast_NoPeer(t *testing.T) {
	op := &broadcastChatMessageOp{deps: &fakeChatDeps{}}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{"peer": []any{}}))
	if err == nil {
		t.Fatal("expected error when peer is empty")
	}
}

func TestBroadcast_ProfileTypeError(t *testing.T) {
	deps := &fakeChatDeps{profileTypeErr: fmt.Errorf("not found")}
	op := &broadcastChatMessageOp{deps: deps}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{
		"peer":    []any{"user-1"},
		"profile": map[string]any{"id": 5},
	}))
	if err == nil {
		t.Fatal("expected error when ChatProfileType fails")
	}
}

func TestBroadcast_BroadcastError(t *testing.T) {
	deps := &fakeChatDeps{broadcastErr: fmt.Errorf("unavailable")}
	op := &broadcastChatMessageOp{deps: deps}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{
		"peer": []any{"user-1"},
	}))
	if err == nil {
		t.Fatal("expected error when BroadcastChatMessage fails")
	}
}

func TestBroadcast_Success_StringPeer(t *testing.T) {
	deps := &fakeChatDeps{
		broadcastResp: &chatdomain.BroadcastChatResponse{
			Variables: map[string]string{"msg_id": "abc"},
		},
	}
	op := &broadcastChatMessageOp{deps: deps}
	out, err := op.Execute(context.Background(), chatInput(map[string]any{
		"peer": []any{"ch-1", "ch-2"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["msg_id"] != "abc" {
		t.Errorf("SetVars = %v, want msg_id=abc", out.SetVars)
	}
}

func TestBroadcast_Success_MapPeer(t *testing.T) {
	deps := &fakeChatDeps{
		broadcastResp: &chatdomain.BroadcastChatResponse{},
	}
	op := &broadcastChatMessageOp{deps: deps}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{
		"peer": []any{
			map[string]any{"id": "ch-3", "type": "webitel", "via": "10"},
		},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBroadcast_FailedReceivers(t *testing.T) {
	deps := &fakeChatDeps{
		broadcastResp: &chatdomain.BroadcastChatResponse{
			Failed: []*chatdomain.FailedReceiver{{Id: "ch-1", Error: "timeout"}},
		},
	}
	op := &broadcastChatMessageOp{deps: deps}
	out, err := op.Execute(context.Background(), chatInput(map[string]any{
		"peer":            []any{"ch-1"},
		"responseCode":    "broadcast_status",
		"failedReceivers": "broadcast_failed",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["broadcast_status"] != "timeout" {
		t.Errorf("broadcast_status = %q, want timeout", out.SetVars["broadcast_status"])
	}
}

// ── op.go: chatHistory ────────────────────────────────────────────────────────

func TestChatHistory_MessagesError(t *testing.T) {
	deps := &fakeChatDeps{messagesErr: fmt.Errorf("db error")}
	op := &chatHistoryOp{deps: deps}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{"variable": "history"}))
	if err == nil {
		t.Fatal("expected error when GetChatMessages fails")
	}
}

func TestChatHistory_ParseError(t *testing.T) {
	msgs := &[]chatdomain.ChatMessage{}
	deps := &fakeChatDeps{messagesResult: msgs, parseErr: fmt.Errorf("bad format")}
	op := &chatHistoryOp{deps: deps}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{"variable": "history"}))
	if err == nil {
		t.Fatal("expected error when ParseChatMessages fails")
	}
}

func TestChatHistory_Success(t *testing.T) {
	msgs := &[]chatdomain.ChatMessage{{Text: "hello"}}
	deps := &fakeChatDeps{messagesResult: msgs, parsedText: "hello"}
	op := &chatHistoryOp{deps: deps}
	out, err := op.Execute(context.Background(), chatInput(map[string]any{"variable": "hist"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["hist"] != "hello" {
		t.Errorf("SetVars = %v, want hist=hello", out.SetVars)
	}
}

// ── queue.go: cancelQueue ─────────────────────────────────────────────────────

func TestChatCancelQueue_NoConv(t *testing.T) {
	op := &cancelQueueOp{deps: &fakeChatQueueDeps{}}
	_, err := op.Execute(context.Background(), chatInput(nil))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatCancelQueue_NoKey(t *testing.T) {
	conv := &stubConversation{}
	op := &cancelQueueOp{deps: &fakeChatQueueDeps{}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(nil))
	if err == nil {
		t.Fatal("expected error when no queue key")
	}
}

func TestChatCancelQueue_DepError(t *testing.T) {
	conv := &stubConversation{queueKey: &queue.InQueueKey{AttemptId: 1}}
	op := &cancelQueueOp{deps: &fakeChatQueueDeps{cancelErr: fmt.Errorf("cc down")}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(nil))
	if err == nil {
		t.Fatal("expected error when CancelAttempt fails")
	}
}

func TestChatCancelQueue_Success(t *testing.T) {
	conv := &stubConversation{queueKey: &queue.InQueueKey{AttemptId: 7}}
	op := &cancelQueueOp{deps: &fakeChatQueueDeps{}}
	out, err := op.Execute(ctxWithConv(conv), chatInput(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["cc_cancel"] != "true" {
		t.Errorf("cc_cancel = %q, want true", out.SetVars["cc_cancel"])
	}
}

// ── queue.go: joinQueue ───────────────────────────────────────────────────────

func TestChatJoinQueue_NoConv(t *testing.T) {
	op := &joinQueueOp{deps: &fakeChatQueueDeps{}}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{"queue": map[string]any{"id": 1}}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatJoinQueue_FindByNameError(t *testing.T) {
	conv := &stubConversation{}
	deps := &fakeChatQueueDeps{findQueueErr: fmt.Errorf("not found")}
	op := &joinQueueOp{deps: deps}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"queue": map[string]any{"name": "support"},
	}))
	if err == nil {
		t.Fatal("expected error when FindQueueByName fails")
	}
}

func TestChatJoinQueue_JoinFail_ReturnsEmpty(t *testing.T) {
	// When JoinChatToInboundQueue fails, the op returns empty output (no error).
	conv := &stubConversation{}
	deps := &fakeChatQueueDeps{joinErr: fmt.Errorf("grpc error")}
	op := &joinQueueOp{deps: deps}
	out, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Break || len(out.SetVars) > 0 {
		t.Errorf("unexpected output on join fail: %+v", out)
	}
}

func TestChatJoinQueue_LeavingExitsWithResult(t *testing.T) {
	conv := &stubConversation{}
	stream := &fakeChatQueueStream{
		events: []*cc.QueueEvent{
			{Data: &cc.QueueEvent_Leaving{Leaving: &cc.QueueEvent_LeavingData{Result: "answered"}}},
		},
	}
	deps := &fakeChatQueueDeps{joinStream: stream}
	op := &joinQueueOp{deps: deps}

	out, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["cc_result"] != "answered" {
		t.Errorf("cc_result = %q, want answered", out.SetVars["cc_result"])
	}
}

func TestChatJoinQueue_OfferingSetsAgentVars(t *testing.T) {
	conv := &stubConversation{}
	stream := &fakeChatQueueStream{
		events: []*cc.QueueEvent{
			{Data: &cc.QueueEvent_Offering{Offering: &cc.QueueEvent_OfferingData{AgentName: "bob", AgentId: 5}}},
		},
	}
	deps := &fakeChatQueueDeps{joinStream: stream}
	op := &joinQueueOp{deps: deps}

	out, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["cc_agent_name"] != "bob" {
		t.Errorf("cc_agent_name = %q, want bob", out.SetVars["cc_agent_name"])
	}
}

func TestChatJoinQueue_JoinedSetsQueueKey(t *testing.T) {
	conv := &stubConversation{}
	stream := &fakeChatQueueStream{
		events: []*cc.QueueEvent{
			{Data: &cc.QueueEvent_Joined{Joined: &cc.QueueEvent_JoinedData{AttemptId: 99}}},
		},
	}
	deps := &fakeChatQueueDeps{joinStream: stream}
	op := &joinQueueOp{deps: deps}

	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SetQueue should have been called at least once with a non-nil key.
	foundKey := false
	for _, k := range conv.setQueueCalls {
		if k != nil && k.AttemptId == 99 {
			foundKey = true
		}
	}
	if !foundKey {
		t.Errorf("SetQueue never called with AttemptId=99; calls=%v", conv.setQueueCalls)
	}
}

// ── recv.go: chatRecvMessage ──────────────────────────────────────────────────

func withConnAndWaitable(connID string, cw ChatWaitable) context.Context {
	ctx := messaging.WithConnID(context.Background(), connID)
	if cw != nil {
		ctx = WithChatWaitable(ctx, cw)
	}
	return ctx
}

type stubWaitable struct{ started []int }

func (s *stubWaitable) StartWaiting(timeout int) { s.started = append(s.started, timeout) }

func TestChatRecv_FreshPath_NoConnID(t *testing.T) {
	op := chatRecvMessageOp{}
	_, err := op.Execute(context.Background(), chatInput(nil))
	if err == nil {
		t.Fatal("expected error when no connID in context")
	}
}

func TestChatRecv_FreshPath_Suspends(t *testing.T) {
	waitable := &stubWaitable{}
	ctx := withConnAndWaitable("conn-1", waitable)
	op := chatRecvMessageOp{}
	out, err := op.Execute(ctx, chatInput(map[string]any{"timeout": 30}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey != "msg:conn-1" {
		t.Errorf("SuspendKey = %q, want msg:conn-1", out.SuspendKey)
	}
	if !out.ReenterOnResume {
		t.Error("expected ReenterOnResume=true")
	}
	if len(waitable.started) != 1 || waitable.started[0] != 30 {
		t.Errorf("StartWaiting calls = %v, want [30]", waitable.started)
	}
}

func TestChatRecv_FreshPath_NoWaitable(t *testing.T) {
	// StartWaiting is optional — should not panic when no ChatWaitable in ctx.
	ctx := messaging.WithConnID(context.Background(), "conn-2")
	op := chatRecvMessageOp{}
	out, err := op.Execute(ctx, chatInput(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey == "" {
		t.Error("expected SuspendKey to be set even without waitable")
	}
}

func TestChatRecv_Resume_Timeout(t *testing.T) {
	op := chatRecvMessageOp{}
	in := ops.OpInput{
		Node:          &tree.Node{Args: map[string]any{"timeoutSet": "timed_out"}},
		ResumePayload: map[string]string{"timeout": "true"},
	}
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["timed_out"] != "true" {
		t.Errorf("SetVars = %v, want timed_out=true", out.SetVars)
	}
}

func TestChatRecv_Resume_TriggerMatch(t *testing.T) {
	triggerNode := &tree.Node{ID: "cmd-node"}
	op := chatRecvMessageOp{}
	in := ops.OpInput{
		Node:          &tree.Node{Args: map[string]any{"set": "user_msg"}},
		ResumePayload: map[string]string{"msg": "help"},
		Triggers:      map[string]*tree.Node{"commands-help": triggerNode},
	}
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Branch != triggerNode {
		t.Error("expected trigger branch to be returned")
	}
	if !out.ReenterOnResume {
		t.Error("expected ReenterOnResume=true after trigger dispatch")
	}
}

func TestChatRecv_Resume_SetVar(t *testing.T) {
	op := chatRecvMessageOp{}
	in := ops.OpInput{
		Node:          &tree.Node{Args: map[string]any{"set": "user_reply"}},
		ResumePayload: map[string]string{"msg": "hello"},
	}
	out, err := op.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["user_reply"] != "hello" {
		t.Errorf("SetVars = %v, want user_reply=hello", out.SetVars)
	}
}

// ── send.go: sendText ─────────────────────────────────────────────────────────

func TestChatSendText_NoConv(t *testing.T) {
	_, err := sendTextOp{}.Execute(context.Background(), ops.OpInput{Node: &tree.Node{RawArgs: "hi"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatSendText_SendsText(t *testing.T) {
	conv := &stubConversation{}
	_, err := sendTextOp{}.Execute(ctxWithConv(conv), ops.OpInput{Node: &tree.Node{RawArgs: "hello"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.sentTexts) != 1 || conv.sentTexts[0] != "hello" {
		t.Errorf("sentTexts = %v", conv.sentTexts)
	}
}

func TestChatSendText_VariableExpansion(t *testing.T) {
	conv := &stubConversation{}
	in := ops.OpInput{
		Node:      &tree.Node{RawArgs: "Hi ${name}"},
		Variables: map[string]string{"name": "Bob"},
	}
	_, err := sendTextOp{}.Execute(ctxWithConv(conv), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conv.sentTexts[0] != "Hi Bob" {
		t.Errorf("sentTexts[0] = %q, want Hi Bob", conv.sentTexts[0])
	}
}

// ── send.go: sendMessage ──────────────────────────────────────────────────────

func TestChatSendMessage_NoConv(t *testing.T) {
	op := &sendMessageOp{deps: &fakeChatSendDeps{}}
	_, err := op.Execute(context.Background(), chatInput(nil))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatSendMessage_NoFile(t *testing.T) {
	conv := &stubConversation{}
	op := &sendMessageOp{deps: &fakeChatSendDeps{}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{"text": "hi", "type": "text"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.sentMessages) != 1 {
		t.Fatalf("expected 1 SendMessage call, got %d", len(conv.sentMessages))
	}
}

func TestChatSendMessage_FileURLShortcut(t *testing.T) {
	conv := &stubConversation{}
	deps := &fakeChatSendDeps{}
	op := &sendMessageOp{deps: deps}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"file": map[string]any{"url": "https://cdn/img.png", "id": 0},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.searchCalls) != 0 {
		t.Error("SearchMediaFile should be skipped when File.Url is set")
	}
}

func TestChatSendMessage_MediaLookup(t *testing.T) {
	conv := &stubConversation{}
	deps := &fakeChatSendDeps{}
	op := &sendMessageOp{deps: deps}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"file": map[string]any{"id": 3, "name": "doc.pdf"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps.searchCalls) != 1 {
		t.Errorf("SearchMediaFile not called: searchCalls=%v", deps.searchCalls)
	}
	if conv.sentMessages[0].Type != "file" {
		t.Errorf("Type = %q, want file", conv.sentMessages[0].Type)
	}
}

// ── send.go: sendFile ─────────────────────────────────────────────────────────

func TestChatSendFile_NoConv(t *testing.T) {
	op := &sendFileOp{deps: &fakeChatSendDeps{}}
	_, err := op.Execute(context.Background(), chatInput(nil))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatSendFile_FileNotFound(t *testing.T) {
	conv := &stubConversation{}
	// SearchMediaFile returns (nil, nil) → "file not found" error.
	op := &sendFileOp{deps: &nilFileSearchDeps{&fakeChatSendDeps{}}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{"file": map[string]any{"id": 1}}))
	if err == nil {
		t.Fatal("expected error when file not found")
	}
}

func TestChatSendFile_Success(t *testing.T) {
	conv := &stubConversation{}
	deps := &fakeChatSendDeps{searchResult: &files.File{Id: 5, Name: "vid.mp4"}}
	op := &sendFileOp{deps: deps}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"file": map[string]any{"id": 5},
		"text": "caption",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.sentFiles) != 1 || conv.sentFiles[0] != "caption" {
		t.Errorf("sentFiles = %v, want [caption]", conv.sentFiles)
	}
}

// nilFileSearchDeps returns (nil, nil) for SearchMediaFile — triggers "file not found".
type nilFileSearchDeps struct{ *fakeChatSendDeps }

func (n *nilFileSearchDeps) SearchMediaFile(_ int64, _ *files.SearchFile) (*files.File, error) {
	return nil, nil
}

// ── send.go: sendImage ────────────────────────────────────────────────────────

func TestChatSendImage_NoConv(t *testing.T) {
	_, err := sendImageOp{}.Execute(context.Background(), chatInput(nil))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatSendImage_InvalidURL(t *testing.T) {
	conv := &stubConversation{}
	_, err := sendImageOp{}.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"url": "not-a-url",
	}))
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestChatSendImage_Success(t *testing.T) {
	conv := &stubConversation{}
	_, err := sendImageOp{}.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"url":  "https://cdn.example.com/img.jpg",
		"name": "photo",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.sentImages) != 1 {
		t.Errorf("sentImages = %v, want 1 call", conv.sentImages)
	}
}

// ── send.go: sendAction ───────────────────────────────────────────────────────

func TestChatSendAction_NoConv(t *testing.T) {
	op := &sendActionOp{deps: &fakeChatSendDeps{}}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{"action": "typing"}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatSendAction_Success(t *testing.T) {
	conv := &stubConversation{id: "ch-1"}
	deps := &fakeChatSendDeps{}
	op := &sendActionOp{deps: deps}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{"action": "typing"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── send.go: sendTts ──────────────────────────────────────────────────────────

func TestChatSendTts_NoConv(t *testing.T) {
	op := &sendTtsOp{deps: &fakeChatSendDeps{}}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{"message": "hello"}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChatSendTts_GenError(t *testing.T) {
	conv := &stubConversation{}
	deps := &fakeChatSendDeps{ttsErr: fmt.Errorf("tts unavailable")}
	op := &sendTtsOp{deps: deps}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{"message": "hi"}))
	if err == nil {
		t.Fatal("expected error when GenerateTTSLink fails")
	}
}

func TestChatSendTts_Success(t *testing.T) {
	conv := &stubConversation{}
	deps := &fakeChatSendDeps{ttsURI: "/audio/abc.mp3"}
	op := &sendTtsOp{deps: deps}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"message":    "Hello",
		"profileId":  1,
		"language":   "uk",
		"server":     "https://tts.example.com",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conv.sentFiles) != 1 {
		t.Errorf("SendFile not called; sentFiles=%v", conv.sentFiles)
	}
}

// ── stt.go: stt ───────────────────────────────────────────────────────────────

func TestSTT_NoConv(t *testing.T) {
	op := &sttOp{deps: &fakeSTTDeps{}}
	_, err := op.Execute(context.Background(), chatInput(map[string]any{
		"fileId": 1, "profileId": 1, "language": "uk", "setVar": "text",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSTT_InvalidFileId(t *testing.T) {
	conv := &stubConversation{}
	op := &sttOp{deps: &fakeSTTDeps{}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"fileId": 0, "profileId": 1, "language": "uk", "setVar": "text",
	}))
	if err == nil {
		t.Fatal("expected error for fileId <= 0")
	}
}

func TestSTT_InvalidProfileId(t *testing.T) {
	conv := &stubConversation{}
	op := &sttOp{deps: &fakeSTTDeps{}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"fileId": 1, "profileId": 0, "language": "uk", "setVar": "text",
	}))
	if err == nil {
		t.Fatal("expected error for profileId <= 0")
	}
}

func TestSTT_EmptyLanguage(t *testing.T) {
	conv := &stubConversation{}
	op := &sttOp{deps: &fakeSTTDeps{}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"fileId": 1, "profileId": 1, "setVar": "text",
	}))
	if err == nil {
		t.Fatal("expected error for empty language")
	}
}

func TestSTT_EmptySetVar(t *testing.T) {
	conv := &stubConversation{}
	op := &sttOp{deps: &fakeSTTDeps{}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"fileId": 1, "profileId": 1, "language": "uk",
	}))
	if err == nil {
		t.Fatal("expected error for empty setVar")
	}
}

func TestSTT_DepError(t *testing.T) {
	conv := &stubConversation{}
	op := &sttOp{deps: &fakeSTTDeps{err: fmt.Errorf("stt unavailable")}}
	_, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"fileId": 5, "profileId": 2, "language": "uk", "setVar": "transcript",
	}))
	if err == nil {
		t.Fatal("expected error when GetFileTranscription fails")
	}
}

func TestSTT_Success(t *testing.T) {
	conv := &stubConversation{}
	op := &sttOp{deps: &fakeSTTDeps{text: "hello world"}}
	out, err := op.Execute(ctxWithConv(conv), chatInput(map[string]any{
		"fileId": 5, "profileId": 2, "language": "uk", "setVar": "transcript",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SetVars["transcript"] != "hello world" {
		t.Errorf("SetVars = %v, want transcript=hello world", out.SetVars)
	}
	if conv.vars["transcript"] != "hello world" {
		t.Errorf("conv.vars = %v, want transcript=hello world", conv.vars)
	}
}
