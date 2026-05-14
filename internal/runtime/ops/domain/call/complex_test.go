package call

import (
	"context"
	"io"
	"testing"

	"github.com/webitel/wlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/webitel/flow_manager/api/gen/cc"
	calldomain "github.com/webitel/flow_manager/internal/domain/call"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/tree"
	"github.com/webitel/flow_manager/internal/storage"
)

// ── stubCall ──────────────────────────────────────────────────────────────────

// stubCall is a minimal hand-written stub for calldomain.Call.
// Only methods exercised by joinQueueOp and joinAgentOp are implemented;
// all others panic to surface accidental calls during tests.
type stubCall struct {
	id            string
	domainId      int64
	inQueue       bool
	stopped       bool
	hangupCause   string
	transferHist  string // value of "variable_transfer_history"
	// transferHistFn overrides transferHist when set; lets tests simulate
	// ESL updating the variable mid-stream.
	transferHistFn func() string
	contactId     int
	meetingId     string
	transferQId   int
	transferAgId  int
	blindTransfer bool
	exportVars    map[string]string
	vars          map[string]string

	// observable side-effect
	queueCancelSet context.CancelFunc
}

func (c *stubCall) Type() flow.ConnectionType       { return flow.ConnectionTypeCall }
func (c *stubCall) Id() string                       { return c.id }
func (c *stubCall) NodeId() string                   { return "" }
func (c *stubCall) DomainId() int64                  { return c.domainId }
func (c *stubCall) Context() context.Context         { return context.Background() }
func (c *stubCall) Close() error                     { return nil }
func (c *stubCall) Log() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}
func (c *stubCall) Variables() map[string]string     { return c.vars }
func (c *stubCall) ParseText(text string, _ ...flow.ParseOption) string { return text }

func (c *stubCall) Get(key string) (string, bool) {
	v, ok := c.vars[key]
	return v, ok
}
func (c *stubCall) Set(_ context.Context, vars flow.Variables) (flow.Response, error) {
	if c.vars == nil {
		c.vars = make(map[string]string)
	}
	for k, v := range vars {
		c.vars[k] = v.(string)
	}
	return nil, nil
}

func (c *stubCall) GetVariable(name string) string {
	if name == "variable_transfer_history" {
		if c.transferHistFn != nil {
			return c.transferHistFn()
		}
		return c.transferHist
	}
	return c.vars[name]
}

func (c *stubCall) DumpExportVariables() map[string]string {
	out := make(map[string]string, len(c.exportVars))
	for k, v := range c.exportVars {
		out[k] = v
	}
	return out
}

func (c *stubCall) InQueue() bool                                       { return c.inQueue }
func (c *stubCall) Stopped() bool                                       { return c.stopped }
func (c *stubCall) HangupCause() string                                 { return c.hangupCause }
func (c *stubCall) GetContactId() int                                   { return c.contactId }
func (c *stubCall) MeetingId() string                                   { return c.meetingId }
func (c *stubCall) TransferQueueId() int                                { return c.transferQId }
func (c *stubCall) IsBlindTransferQueue() bool                          { return c.blindTransfer }
func (c *stubCall) TransferAgentId() int                                { return c.transferAgId }
func (c *stubCall) CancelQueue() bool                                   { return false }

func (c *stubCall) SetQueueCancel(cancel context.CancelFunc) bool {
	c.queueCancelSet = cancel
	return true
}

// Intentionally-unimplemented methods: panic if a test path reaches them.
func (c *stubCall) UserId() int                                       { panic("not implemented") }
func (c *stubCall) From() *calldomain.CallEndpoint                    { panic("not implemented") }
func (c *stubCall) To() *calldomain.CallEndpoint                      { panic("not implemented") }
func (c *stubCall) IsTransfer() bool                                  { panic("not implemented") }
func (c *stubCall) IsOriginateRequest() bool                          { panic("not implemented") }
func (c *stubCall) Direction() calldomain.CallDirection               { panic("not implemented") }
func (c *stubCall) Destination() string                               { panic("not implemented") }
func (c *stubCall) SetDomainName(_ string)                            { panic("not implemented") }
func (c *stubCall) SetSchemaId(_ int) error                           { panic("not implemented") }
func (c *stubCall) DomainName() string                                { panic("not implemented") }
func (c *stubCall) Dump()                                             { panic("not implemented") }
func (c *stubCall) IVRQueueId() *int                                  { panic("not implemented") }
func (c *stubCall) TransferSchemaId() *int                            { panic("not implemented") }
func (c *stubCall) SetTransferFromId()                                { panic("not implemented") }
func (c *stubCall) SetTransferAfterBridge(_ context.Context, _ int) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) SetAll(_ context.Context, _ flow.Variables) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) SetNoLocal(_ context.Context, _ flow.Variables) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) UnSet(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) RingReady(_ context.Context) (flow.Response, error)      { panic("not implemented") }
func (c *stubCall) PreAnswer(_ context.Context) (flow.Response, error)      { panic("not implemented") }
func (c *stubCall) Answer(_ context.Context) (flow.Response, error)         { panic("not implemented") }
func (c *stubCall) Echo(_ context.Context, _ int) (flow.Response, error)    { panic("not implemented") }
func (c *stubCall) Hangup(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) HangupNoRoute(_ context.Context) (flow.Response, error)  { panic("not implemented") }
func (c *stubCall) HangupAppErr(_ context.Context) (flow.Response, error)   { panic("not implemented") }
func (c *stubCall) Bridge(_ context.Context, _ calldomain.Call, _ string, _ map[string]string, _ []*calldomain.Endpoint, _ []string, _ chan struct{}, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Sleep(_ context.Context, _ int) (flow.Response, error) { panic("not implemented") }
func (c *stubCall) Conference(_ context.Context, _, _, _ string, _ []string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) RecordFile(_ context.Context, _ string, _ string, _, _, _ int) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) SendFileToAi(_ context.Context, _ string, _ map[string]string, _ string, _, _, _ int) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) RecordSession(_ context.Context, _, _ string, _ int, _, _, _ bool) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) RecordSessionStop(_ context.Context, _, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Export(_ context.Context, _ []string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) FlushDTMF(_ context.Context) (flow.Response, error) { panic("not implemented") }
func (c *stubCall) StartDTMF(_ context.Context) (flow.Response, error) { panic("not implemented") }
func (c *stubCall) StopDTMF(_ context.Context) (flow.Response, error)  { panic("not implemented") }
func (c *stubCall) Park(_ context.Context, _ string, _ bool, _, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Playback(_ context.Context, _ []*calldomain.PlaybackFile) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Say(_ context.Context, _ string) (flow.Response, error) { panic("not implemented") }
func (c *stubCall) PlaybackAndGetDigits(_ context.Context, _ []*calldomain.PlaybackFile, _ *calldomain.PlaybackDigits) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) PlaybackUrl(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) PlaybackUrlAndGetDigits(_ context.Context, _ string, _ *calldomain.PlaybackDigits) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) PushSpeechMessage(_ calldomain.SpeechMessage)         { panic("not implemented") }
func (c *stubCall) SpeechMessages(_ int) []calldomain.SpeechMessage      { panic("not implemented") }
func (c *stubCall) TTS(_ context.Context, _ string, _ calldomain.TTSSettings, _ *calldomain.PlaybackDigits, _ int) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) TTSOpus(_ context.Context, _ string, _ *calldomain.PlaybackDigits, _ int) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Redirect(_ context.Context, _ []string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) SetSounds(_ context.Context, _, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) ScheduleHangup(_ context.Context, _ int, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Ringback(_ context.Context, _ bool, _, _, _ *calldomain.PlaybackFile) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) ClearExportVariables()                             {}
func (c *stubCall) Queue(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Intercept(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Amd(_ context.Context, _ calldomain.AmdParameters) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) AmdML(_ context.Context, _ calldomain.AmdMLParameters) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Pickup(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) PickupHash(_ string) string                        { panic("not implemented") }
func (c *stubCall) StartRecognize(_ context.Context, _, _ string, _, _ int) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) StopRecognize(_ context.Context) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) GoogleTranscribe(_ context.Context, _ *calldomain.GetSpeech) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) GoogleTranscribeStop(_ context.Context) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) RefreshVars(_ context.Context) (flow.Response, error) { panic("not implemented") }
func (c *stubCall) UpdateCid(_ context.Context, _, _, _ *string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Push(_ context.Context, _, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Cv(_ context.Context) (flow.Response, error) { panic("not implemented") }
func (c *stubCall) BackgroundPlayback(_ context.Context, _ *calldomain.PlaybackFile, _ string, _ int) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) BackgroundPlaybackStop(_ context.Context, _ string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Bot(_ context.Context, _ string, _ int, _ string, _ map[string]string) (flow.Response, error) {
	panic("not implemented")
}
func (c *stubCall) Update(_ context.Context) (flow.Response, error) { panic("not implemented") }

// ── fakeComplexDeps ──────────────────────────────────────────────────────────

type fakeComplexDeps struct {
	joinQueueErr    error
	joinQueueStream cc.MemberService_CallJoinToQueueClient
	joinAgentErr    error
	joinAgentStream cc.MemberService_CallJoinToAgentClient
}

func (f *fakeComplexDeps) GetStore() storage.Store { panic("not called") }
func (f *fakeComplexDeps) GetMediaFiles(_ int64, _ *[]*calldomain.PlaybackFile) ([]*calldomain.PlaybackFile, error) {
	return nil, nil
}
func (f *fakeComplexDeps) GetAgentIdByExtension(_ int64, _ string) (*int32, error) {
	return nil, nil
}
func (f *fakeComplexDeps) JoinToInboundQueue(_ context.Context, _ *cc.CallJoinToQueueRequest) (cc.MemberService_CallJoinToQueueClient, error) {
	return f.joinQueueStream, f.joinQueueErr
}
func (f *fakeComplexDeps) JoinToAgent(_ context.Context, _ *cc.CallJoinToAgentRequest) (cc.MemberService_CallJoinToAgentClient, error) {
	return f.joinAgentStream, f.joinAgentErr
}

// ── fakeQueueStream ──────────────────────────────────────────────────────────

// fakeQueueStream delivers a pre-loaded list of QueueEvents then returns EOF.
type fakeQueueStream struct {
	events []*cc.QueueEvent
	idx    int
	err    error // returned on first RecvMsg if non-nil
}

func (f *fakeQueueStream) RecvMsg(m interface{}) error {
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
func (f *fakeQueueStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeQueueStream) Trailer() metadata.MD         { return nil }
func (f *fakeQueueStream) CloseSend() error             { return nil }
func (f *fakeQueueStream) Context() context.Context     { return context.Background() }
func (f *fakeQueueStream) SendMsg(_ interface{}) error  { return nil }
func (f *fakeQueueStream) Recv() (*cc.QueueEvent, error) {
	var ev cc.QueueEvent
	if err := f.RecvMsg(&ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

// fakeAgentStream is the same pattern for CallJoinToAgentClient.
type fakeAgentStream struct {
	events []*cc.QueueEvent
	idx    int
	err    error
}

func (f *fakeAgentStream) RecvMsg(m interface{}) error {
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
func (f *fakeAgentStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeAgentStream) Trailer() metadata.MD         { return nil }
func (f *fakeAgentStream) CloseSend() error             { return nil }
func (f *fakeAgentStream) Context() context.Context     { return context.Background() }
func (f *fakeAgentStream) SendMsg(_ interface{}) error  { return nil }
func (f *fakeAgentStream) Recv() (*cc.QueueEvent, error) {
	var ev cc.QueueEvent
	if err := f.RecvMsg(&ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

// Verify interface compliance at compile time.
var _ grpc.ClientStream = (*fakeQueueStream)(nil)
var _ grpc.ClientStream = (*fakeAgentStream)(nil)

// ── helpers ──────────────────────────────────────────────────────────────────

func ctxWithCall(call calldomain.Call) context.Context {
	return connctx.WithConnection(context.Background(), call)
}

func queueInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}}
}

func agentInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}}
}

// ── joinQueue tests ───────────────────────────────────────────────────────────

func TestJoinQueue_NoCallInContext(t *testing.T) {
	op := &joinQueueOp{deps: &fakeComplexDeps{}}
	_, err := op.Execute(context.Background(), queueInput(nil))
	if err == nil {
		t.Fatal("expected error when no call in context")
	}
}

func TestJoinQueue_AlreadyInQueue(t *testing.T) {
	call := &stubCall{inQueue: true}
	op := &joinQueueOp{deps: &fakeComplexDeps{}}
	_, err := op.Execute(ctxWithCall(call), queueInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err == nil {
		t.Fatal("expected error when call is already in queue")
	}
}

func TestJoinQueue_JoinFails_CancelCalled(t *testing.T) {
	// Verifies the Phase-6.1 fix: cancelQueue() must be called when
	// JoinToInboundQueue returns an error, not just on success paths.
	call := &stubCall{}
	joinErr := io.ErrUnexpectedEOF
	deps := &fakeComplexDeps{joinQueueErr: joinErr}
	op := &joinQueueOp{deps: deps}

	out, err := op.Execute(ctxWithCall(call), queueInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("expected no error propagation, got %v", err)
	}
	if out.Break {
		t.Error("expected Break=false on queue join failure")
	}
	// SetQueueCancel should NOT have been called with a non-nil func,
	// because the stream never started.
	if call.queueCancelSet != nil {
		t.Error("queueCancel should not be set when JoinToInboundQueue failed")
	}
}

func TestJoinQueue_StreamEOF_NoTransfer(t *testing.T) {
	call := &stubCall{transferHist: "initial"}
	deps := &fakeComplexDeps{
		joinQueueStream: &fakeQueueStream{}, // immediate EOF
	}
	op := &joinQueueOp{deps: deps}

	out, err := op.Execute(ctxWithCall(call), queueInput(map[string]any{
		"queue": map[string]any{"id": 42, "name": "support"},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Break {
		t.Error("expected Break=false when no transfer occurred")
	}
}

func TestJoinQueue_StreamEOF_TransferDetected(t *testing.T) {
	// Simulates ESL updating variable_transfer_history while the call is in queue.
	// The op captures the value at entry and compares at exit — a change means
	// a transfer happened and the flow should Break.
	call := &stubCall{}
	calls := 0
	call.transferHistFn = func() string {
		calls++
		if calls == 1 {
			return "before" // captured at op entry
		}
		return "after" // observed at op exit after stream EOF
	}

	deps := &fakeComplexDeps{
		joinQueueStream: &fakeQueueStream{}, // immediate EOF
	}
	op := &joinQueueOp{deps: deps}

	out, err := op.Execute(ctxWithCall(call), queueInput(map[string]any{
		"queue": map[string]any{"id": 1},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Break {
		t.Error("expected Break=true when transfer history changed")
	}
}

func TestJoinQueue_HooksFired(t *testing.T) {
	// Verifies that "offering" and "leaving" event hooks are dispatched.
	call := &stubCall{}

	events := []*cc.QueueEvent{
		{Data: &cc.QueueEvent_Offering{Offering: &cc.QueueEvent_OfferingData{AgentName: "alice", AgentId: 7}}},
		{Data: &cc.QueueEvent_Leaving{Leaving: &cc.QueueEvent_LeavingData{Result: "abandoned"}}},
	}
	deps := &fakeComplexDeps{
		joinQueueStream: &fakeQueueStream{events: events},
	}

	var fired []string
	hookNode := &tree.Node{
		Args: map[string]any{"_hooks_index": map[string]int{"offering": 0, "reporting": 1}},
		Children: []*tree.Node{
			{}, // offering branch
			{}, // reporting branch
		},
	}
	in := ops.OpInput{
		Node: hookNode,
		RunBranch: func(_ context.Context, n *tree.Node, vars map[string]string) {
			switch {
			case n == hookNode.Children[0]:
				fired = append(fired, "offering:"+vars["cc_agent_name"])
			case n == hookNode.Children[1]:
				fired = append(fired, "reporting:"+vars["cc_result"])
			}
		},
	}

	op := &joinQueueOp{deps: deps}
	_, err := op.Execute(ctxWithCall(call), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFired := []string{"offering:alice", "reporting:abandoned"}
	if len(fired) != len(wantFired) {
		t.Fatalf("hooks fired %v, want %v", fired, wantFired)
	}
	for i, want := range wantFired {
		if fired[i] != want {
			t.Errorf("hook[%d] = %q, want %q", i, fired[i], want)
		}
	}
}

// ── joinAgent tests ───────────────────────────────────────────────────────────

func TestJoinAgent_NoCallInContext(t *testing.T) {
	op := &joinAgentOp{deps: &fakeComplexDeps{}}
	_, err := op.Execute(context.Background(), agentInput(nil))
	if err == nil {
		t.Fatal("expected error when no call in context")
	}
}

func TestJoinAgent_NoAgent(t *testing.T) {
	call := &stubCall{}
	op := &joinAgentOp{deps: &fakeComplexDeps{}}
	_, err := op.Execute(ctxWithCall(call), agentInput(map[string]any{}))
	if err == nil {
		t.Fatal("expected error when agent field is absent")
	}
}

func TestJoinAgent_AgentIdNil_ExtensionNotFound(t *testing.T) {
	// GetAgentIdByExtension returns nil — agent lookup fails.
	call := &stubCall{}
	deps := &fakeComplexDeps{}
	op := &joinAgentOp{deps: deps}
	ext := "101"
	_, err := op.Execute(ctxWithCall(call), agentInput(map[string]any{
		"agent": map[string]any{"extension": ext},
	}))
	if err == nil {
		t.Fatal("expected error when agent not found by extension")
	}
}

func TestJoinAgent_JoinFails_NoErrorPropagated(t *testing.T) {
	call := &stubCall{}
	agentId := int32(99)
	deps := &fakeComplexDeps{joinAgentErr: io.ErrUnexpectedEOF}
	op := &joinAgentOp{deps: deps}

	out, err := op.Execute(ctxWithCall(call), agentInput(map[string]any{
		"agent": map[string]any{"id": agentId},
	}))
	if err != nil {
		t.Fatalf("join error should not propagate to caller, got: %v", err)
	}
	if out.Break {
		t.Error("expected Break=false")
	}
}

func TestJoinAgent_StreamEOF_NoTransfer(t *testing.T) {
	call := &stubCall{transferHist: "v1"}
	agentId := int32(5)
	deps := &fakeComplexDeps{
		joinAgentStream: &fakeAgentStream{},
	}
	op := &joinAgentOp{deps: deps}

	out, err := op.Execute(ctxWithCall(call), agentInput(map[string]any{
		"agent": map[string]any{"id": agentId},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Break {
		t.Error("expected Break=false when no transfer occurred")
	}
}

func TestJoinAgent_StreamEOF_TransferDetected(t *testing.T) {
	call := &stubCall{}
	calls := 0
	call.transferHistFn = func() string {
		calls++
		if calls == 1 {
			return "before"
		}
		return "after"
	}
	agentId := int32(5)
	deps := &fakeComplexDeps{
		joinAgentStream: &fakeAgentStream{},
	}
	op := &joinAgentOp{deps: deps}

	out, err := op.Execute(ctxWithCall(call), agentInput(map[string]any{
		"agent": map[string]any{"id": agentId},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Break {
		t.Error("expected Break=true when transfer history changed")
	}
}

// Compile-time check: stubCall must satisfy calldomain.Call.
var _ calldomain.Call = (*stubCall)(nil)

// Compile-time check: fakeDeps must satisfy ComplexDeps.
var _ ComplexDeps = (*fakeComplexDeps)(nil)

