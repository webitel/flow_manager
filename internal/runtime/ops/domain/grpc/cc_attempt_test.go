package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/webitel/wlog"

	domgrpc "github.com/webitel/flow_manager/internal/domain/grpc"
	"github.com/webitel/flow_manager/internal/domain/flow"
	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/connctx"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── stubGRPCConn ──────────────────────────────────────────────────────────────

type stubGRPCConn struct {
	id       string
	results  []interface{}
	exported [][]string
	exportErr error
}

// flow.Connection
func (c *stubGRPCConn) Type() flow.ConnectionType { return flow.ConnectionTypeGrpc }
func (c *stubGRPCConn) Id() string                { return c.id }
func (c *stubGRPCConn) NodeId() string            { return "" }
func (c *stubGRPCConn) DomainId() int64           { return 1 }
func (c *stubGRPCConn) Context() context.Context  { return context.Background() }
func (c *stubGRPCConn) Close() error              { return nil }
func (c *stubGRPCConn) Log() *wlog.Logger {
	return wlog.NewLogger(&wlog.LoggerConfiguration{EnableConsole: false})
}
func (c *stubGRPCConn) Variables() map[string]string                         { return nil }
func (c *stubGRPCConn) Get(_ string) (string, bool)                          { return "", false }
func (c *stubGRPCConn) Set(_ context.Context, _ flow.Variables) (flow.Response, error) {
	return nil, nil
}
func (c *stubGRPCConn) ParseText(text string, _ ...flow.ParseOption) string { return text }

// domgrpc.GRPCConnection extras
func (c *stubGRPCConn) SchemaId() int { return 0 }
func (c *stubGRPCConn) Result(result interface{}) {
	c.results = append(c.results, result)
}
func (c *stubGRPCConn) Export(_ context.Context, vars []string) (flow.Response, error) {
	c.exported = append(c.exported, vars)
	return nil, c.exportErr
}
func (c *stubGRPCConn) DumpExportVariables() map[string]string { return nil }
func (c *stubGRPCConn) Scope() flow.Scope                     { return flow.Scope{} }

var _ domgrpc.GRPCConnection = (*stubGRPCConn)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func ctxWithGRPC(conn *stubGRPCConn) context.Context {
	return connctx.WithConnection(context.Background(), conn)
}

func grpcInput(args map[string]any) ops.OpInput {
	return ops.OpInput{Node: &tree.Node{Args: args}, DomainID: 1}
}

// ── exportVars (pure function) ────────────────────────────────────────────────

func TestExportVars_Listed(t *testing.T) {
	vars := map[string]string{"a": "1", "b": "2", "c": "3"}
	res := exportVars([]string{"a", "c"}, vars)
	if res["a"] != "1" || res["c"] != "3" {
		t.Errorf("got %v, want a=1 c=3", res)
	}
	if _, ok := res["b"]; ok {
		t.Error("unlisted key b should not be included")
	}
}

func TestExportVars_Empty(t *testing.T) {
	res := exportVars(nil, map[string]string{"a": "1"})
	if res != nil {
		t.Errorf("expected nil for empty vars list, got %v", res)
	}
}

func TestExportVars_MissingKey(t *testing.T) {
	res := exportVars([]string{"missing"}, map[string]string{})
	if len(res) != 0 {
		t.Errorf("expected empty map when key missing, got %v", res)
	}
}

// ── cancel ────────────────────────────────────────────────────────────────────

func TestCancel_NoConn(t *testing.T) {
	_, err := cancelOp{}.Execute(context.Background(), grpcInput(nil))
	if err == nil {
		t.Fatal("expected error when no grpc connection")
	}
}

func TestCancel_Success(t *testing.T) {
	conn := &stubGRPCConn{}
	_, err := cancelOp{}.Execute(ctxWithGRPC(conn), grpcInput(map[string]any{
		"description": "no answer", "stop": true,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conn.results) != 1 {
		t.Errorf("Result called %d times, want 1", len(conn.results))
	}
}

// ── confirm ───────────────────────────────────────────────────────────────────

func TestConfirm_NoConn(t *testing.T) {
	_, err := confirmOp{}.Execute(context.Background(), grpcInput(nil))
	if err == nil {
		t.Fatal("expected error when no grpc connection")
	}
}

func TestConfirm_Success(t *testing.T) {
	conn := &stubGRPCConn{}
	_, err := confirmOp{}.Execute(ctxWithGRPC(conn), grpcInput(map[string]any{
		"destination": "+380XXXXXXXXX",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conn.results) != 1 {
		t.Errorf("Result called %d times, want 1", len(conn.results))
	}
}

// ── abandoned ─────────────────────────────────────────────────────────────────

func TestAbandoned_NoConn(t *testing.T) {
	_, err := abandonedOp{}.Execute(context.Background(), grpcInput(nil))
	if err == nil {
		t.Fatal("expected error when no grpc connection")
	}
}

func TestAbandoned_Success(t *testing.T) {
	conn := &stubGRPCConn{}
	_, err := abandonedOp{}.Execute(ctxWithGRPC(conn), grpcInput(map[string]any{
		"status": "abandoned", "maxAttempts": 3, "waitBetweenRetries": 300,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conn.results) != 1 {
		t.Errorf("Result called %d times, want 1", len(conn.results))
	}
}

// ── success ───────────────────────────────────────────────────────────────────

func TestSuccess_NoConn(t *testing.T) {
	_, err := successOp{}.Execute(context.Background(), grpcInput(nil))
	if err == nil {
		t.Fatal("expected error when no grpc connection")
	}
}

func TestSuccess_CallsResult(t *testing.T) {
	conn := &stubGRPCConn{}
	_, err := successOp{}.Execute(ctxWithGRPC(conn), grpcInput(map[string]any{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conn.results) != 1 {
		t.Errorf("Result called %d times, want 1", len(conn.results))
	}
}

// ── retry ─────────────────────────────────────────────────────────────────────

func TestRetry_NoConn(t *testing.T) {
	_, err := retryOp{}.Execute(context.Background(), grpcInput(nil))
	if err == nil {
		t.Fatal("expected error when no grpc connection")
	}
}

func TestRetry_Success(t *testing.T) {
	conn := &stubGRPCConn{}
	_, err := retryOp{}.Execute(ctxWithGRPC(conn), grpcInput(map[string]any{
		"nextResource": true, "sleep": 30,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conn.results) != 1 {
		t.Errorf("Result called %d times, want 1", len(conn.results))
	}
}

// ── export ────────────────────────────────────────────────────────────────────

func TestGRPCExport_NoConn(t *testing.T) {
	_, err := exportOp{}.Execute(context.Background(), ops.OpInput{Node: &tree.Node{RawArgs: []any{"v1"}}})
	if err == nil {
		t.Fatal("expected error when no grpc connection")
	}
}

func TestGRPCExport_Success(t *testing.T) {
	conn := &stubGRPCConn{}
	in := ops.OpInput{Node: &tree.Node{RawArgs: []any{"var1", "var2"}}}
	_, err := exportOp{}.Execute(ctxWithGRPC(conn), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conn.exported) != 1 || len(conn.exported[0]) != 2 {
		t.Errorf("exported = %v, want [[var1 var2]]", conn.exported)
	}
}

func TestGRPCExport_DepError(t *testing.T) {
	conn := &stubGRPCConn{exportErr: fmt.Errorf("grpc error")}
	in := ops.OpInput{Node: &tree.Node{RawArgs: []any{"v"}}}
	_, err := exportOp{}.Execute(ctxWithGRPC(conn), in)
	if err == nil {
		t.Fatal("expected error when Export fails")
	}
}
