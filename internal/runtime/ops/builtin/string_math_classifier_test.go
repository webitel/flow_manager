package builtin_test

import (
	"context"
	"strings"
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

// ── string helpers ────────────────────────────────────────────────────────────

func stringInput(fn, data, setVar string, args []any, vars map[string]string) ops.OpInput {
	a := map[string]any{"fn": fn, "data": data, "setVar": setVar}
	if len(args) > 0 {
		a["args"] = args
	}
	return ops.OpInput{Node: &tree.Node{Args: a}, Variables: vars}
}

func runString(t *testing.T, fn, data, setVar string, args []any, vars map[string]string) string {
	t.Helper()
	out, err := builtin.StringOp().Execute(context.Background(),
		stringInput(fn, data, setVar, args, vars))
	if err != nil {
		t.Fatalf("StringOp(%q): unexpected error: %v", fn, err)
	}
	return out.SetVars[setVar]
}

// ── string: Go-native functions ───────────────────────────────────────────────

func TestString_Reverse(t *testing.T) {
	got := runString(t, "reverse", "hello", "out", nil, nil)
	if got != "olleh" {
		t.Errorf("got %q, want olleh", got)
	}
}

func TestString_Length(t *testing.T) {
	got := runString(t, "length", "hello", "out", nil, nil)
	if got != "5" {
		t.Errorf("got %q, want 5", got)
	}
}

func TestString_CharAt(t *testing.T) {
	got := runString(t, "charAt", "hello", "out", []any{"1"}, nil)
	if got != "e" {
		t.Errorf("got %q, want e", got)
	}
}

func TestString_Base64Encode(t *testing.T) {
	got := runString(t, "base64", "hello", "out", []any{"encoder"}, nil)
	if got != "aGVsbG8=" {
		t.Errorf("got %q, want aGVsbG8=", got)
	}
}

func TestString_Base64Decode(t *testing.T) {
	got := runString(t, "base64", "aGVsbG8=", "out", []any{"decoder"}, nil)
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestString_MD5(t *testing.T) {
	got := runString(t, "MD5", "hello", "out", nil, nil)
	if got != "5d41402abc4b2a76b9719d911017c592" {
		t.Errorf("MD5(hello) = %q, want 5d41402abc4b2a76b9719d911017c592", got)
	}
}

func TestString_SHA256(t *testing.T) {
	got := runString(t, "SHA-256", "hello", "out", nil, nil)
	if len(got) != 64 {
		t.Errorf("SHA-256 should be 64 hex chars, got %d: %q", len(got), got)
	}
}

func TestString_SHA512(t *testing.T) {
	got := runString(t, "SHA-512", "hello", "out", nil, nil)
	if len(got) != 128 {
		t.Errorf("SHA-512 should be 128 hex chars, got %d: %q", len(got), got)
	}
}

func TestString_GoMatch_FullMatch(t *testing.T) {
	got := runString(t, "gomatch", "380501234567", "out", []any{"^[0-9]{12}$"}, nil)
	if got == "" {
		t.Error("expected non-empty match for valid phone")
	}
}

func TestString_GoMatch_NoMatch(t *testing.T) {
	got := runString(t, "gomatch", "abc", "out", []any{"^[0-9]+$"}, nil)
	if got != "" {
		t.Errorf("expected empty for no-match, got %q", got)
	}
}

func TestString_GoMatch_CaptureGroup(t *testing.T) {
	got := runString(t, "gomatch", "hello world", "out", []any{"(\\w+) (\\w+)"}, nil)
	if got == "" {
		t.Error("expected captured groups")
	}
}

// ── string: JS-native functions ───────────────────────────────────────────────

func TestString_ToUpperCase(t *testing.T) {
	got := runString(t, "toUpperCase", "hello", "out", nil, nil)
	if got != "HELLO" {
		t.Errorf("got %q, want HELLO", got)
	}
}

func TestString_ToLowerCase(t *testing.T) {
	got := runString(t, "toLowerCase", "HELLO", "out", nil, nil)
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestString_Trim(t *testing.T) {
	got := runString(t, "trim", "  hello  ", "out", nil, nil)
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestString_Split(t *testing.T) {
	got := runString(t, "split", "a,b,c", "out", []any{","}, nil)
	// split result is joined with ","
	if got != "a,b,c" {
		t.Errorf("got %q, want a,b,c", got)
	}
}

func TestString_Includes_True(t *testing.T) {
	got := runString(t, "includes", "hello world", "out", []any{"world"}, nil)
	if got != "true" {
		t.Errorf("got %q, want true", got)
	}
}

func TestString_Includes_False(t *testing.T) {
	got := runString(t, "includes", "hello world", "out", []any{"xyz"}, nil)
	if got != "false" {
		t.Errorf("got %q, want false", got)
	}
}

func TestString_IndexOf(t *testing.T) {
	got := runString(t, "indexOf", "hello", "out", []any{"l"}, nil)
	if got != "2" {
		t.Errorf("got %q, want 2", got)
	}
}

func TestString_Slice(t *testing.T) {
	got := runString(t, "slice", "hello world", "out", []any{"6"}, nil)
	if got != "world" {
		t.Errorf("got %q, want world", got)
	}
}

func TestString_Replace(t *testing.T) {
	got := runString(t, "replace", "hello world", "out", []any{"world", "there"}, nil)
	if got != "hello there" {
		t.Errorf("got %q, want hello there", got)
	}
}

func TestString_VariableExpansion(t *testing.T) {
	got := runString(t, "toUpperCase", "${name}", "out", nil,
		map[string]string{"name": "alice"})
	if got != "ALICE" {
		t.Errorf("got %q, want ALICE", got)
	}
}

// ── math helpers ──────────────────────────────────────────────────────────────

func mathInput(fn, setVar string, data []any) ops.OpInput {
	args := map[string]any{"setVar": setVar, "data": data}
	if fn != "" {
		args["fn"] = fn
	}
	return ops.OpInput{Node: &tree.Node{Args: args}}
}

func runMath(t *testing.T, fn, setVar string, data []any) string {
	t.Helper()
	out, err := builtin.MathOp().Execute(context.Background(), mathInput(fn, setVar, data))
	if err != nil {
		t.Fatalf("MathOp(%q): unexpected error: %v", fn, err)
	}
	return out.SetVars[setVar]
}

// ── math: Go-native ───────────────────────────────────────────────────────────

func TestMath_Random_PicksFromPool(t *testing.T) {
	pool := []any{"a", "b", "c"}
	got := runMath(t, "random", "pick", pool)
	if got != "a" && got != "b" && got != "c" {
		t.Errorf("random pick %q not in pool", got)
	}
}

func TestMath_Random_SingleElement(t *testing.T) {
	got := runMath(t, "random", "pick", []any{"only"})
	if got != "only" {
		t.Errorf("got %q, want only", got)
	}
}

// ── math: JS-native ───────────────────────────────────────────────────────────

func TestMath_Round(t *testing.T) {
	got := runMath(t, "round", "r", []any{3.7})
	if got != "4" {
		t.Errorf("round(3.7) = %q, want 4", got)
	}
}

func TestMath_Floor(t *testing.T) {
	got := runMath(t, "floor", "r", []any{3.9})
	if got != "3" {
		t.Errorf("floor(3.9) = %q, want 3", got)
	}
}

func TestMath_Ceil(t *testing.T) {
	got := runMath(t, "ceil", "r", []any{3.1})
	if got != "4" {
		t.Errorf("ceil(3.1) = %q, want 4", got)
	}
}

func TestMath_Abs(t *testing.T) {
	got := runMath(t, "abs", "r", []any{-5})
	if got != "5" {
		t.Errorf("abs(-5) = %q, want 5", got)
	}
}

func TestMath_Max(t *testing.T) {
	got := runMath(t, "max", "r", []any{1, 5, 3})
	if got != "5" {
		t.Errorf("max(1,5,3) = %q, want 5", got)
	}
}

func TestMath_Min(t *testing.T) {
	got := runMath(t, "min", "r", []any{4, 1, 7})
	if got != "1" {
		t.Errorf("min(4,1,7) = %q, want 1", got)
	}
}

func TestMath_Sqrt(t *testing.T) {
	got := runMath(t, "sqrt", "r", []any{9})
	if got != "3" {
		t.Errorf("sqrt(9) = %q, want 3", got)
	}
}

func TestMath_PI(t *testing.T) {
	got := runMath(t, "PI", "r", nil)
	if !strings.HasPrefix(got, "3.14") {
		t.Errorf("PI = %q, expected to start with 3.14", got)
	}
}

func TestMath_Pow(t *testing.T) {
	got := runMath(t, "pow", "r", []any{2, 8})
	if got != "256" {
		t.Errorf("pow(2,8) = %q, want 256", got)
	}
}

// ── classifier ────────────────────────────────────────────────────────────────

func classifierInput(input, set, matchType string, phraseSearch bool, cluster map[string][]string) ops.OpInput {
	return ops.OpInput{
		Node: &tree.Node{Args: map[string]any{
			"input":        input,
			"set":          set,
			"matchType":    matchType,
			"phraseSearch": phraseSearch,
			"cluster":      toAnyCluster(cluster),
		}},
	}
}

func toAnyCluster(c map[string][]string) map[string]any {
	out := make(map[string]any, len(c))
	for k, v := range c {
		items := make([]any, len(v))
		for i, s := range v {
			items[i] = s
		}
		out[k] = items
	}
	return out
}

func runClassifier(t *testing.T, input, matchType string, phraseSearch bool, cluster map[string][]string) string {
	t.Helper()
	out, err := builtin.Classifier().Execute(context.Background(),
		classifierInput(input, "intent", matchType, phraseSearch, cluster))
	if err != nil {
		t.Fatalf("Classifier: unexpected error: %v", err)
	}
	return out.SetVars["intent"]
}

var intentCluster = map[string][]string{
	"billing": {"invoice", "payment", "bill"},
	"support": {"not working", "problem", "error"},
}

func TestClassifier_PartMatch_Substring(t *testing.T) {
	// phraseSearch=false + matchType=part: tokenizes input, checks if any
	// token or joined input contains the phrase.
	got := runClassifier(t, "I have a payment problem", "part", false, intentCluster)
	if got == "" {
		t.Error("expected a match ('payment' or 'problem' should hit a cluster), got empty")
	}
}

func TestClassifier_FullMatch_ExactToken(t *testing.T) {
	// phraseSearch=false + matchType=full: tokenizes input, checks if any
	// token equals the phrase exactly.
	got := runClassifier(t, "invoice", "full", false, intentCluster)
	if got != "billing" {
		t.Errorf("got %q, want billing", got)
	}
}

func TestClassifier_FullMatch_NoMatch(t *testing.T) {
	got := runClassifier(t, "hello there", "full", false, intentCluster)
	if got != "" {
		t.Errorf("expected empty for no match, got %q", got)
	}
}

func TestClassifier_MultiWordPhrase_PartMatch(t *testing.T) {
	// Multi-word phrase "not working": phraseSearch=false + matchType=part
	// checks if joined token string contains the phrase.
	got := runClassifier(t, "it is not working today", "part", false, intentCluster)
	if got != "support" {
		t.Errorf("got %q, want support", got)
	}
}

func TestClassifier_PhraseSearch_ExactMatch(t *testing.T) {
	// phraseSearch=true: checks if input (as-is) equals the phrase (full mode).
	got := runClassifier(t, "invoice", "full", true, intentCluster)
	if got != "billing" {
		t.Errorf("got %q, want billing for exact match", got)
	}
}

func TestClassifier_EmptyInput(t *testing.T) {
	got := runClassifier(t, "", "full", false, intentCluster)
	if got != "" {
		t.Errorf("expected empty for empty input, got %q", got)
	}
}

func TestClassifier_EmptyCluster(t *testing.T) {
	got := runClassifier(t, "hello", "full", false, map[string][]string{})
	if got != "" {
		t.Errorf("expected empty for empty cluster, got %q", got)
	}
}

// ── httpRequest: validation ───────────────────────────────────────────────────

func TestHttpRequest_EmptyURL_Error(t *testing.T) {
	op := builtin.HTTPRequestOp(nil)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"url": ""}},
	})
	if err == nil {
		t.Fatal("expected error when URL is empty")
	}
}

func TestHttpRequest_InvalidURL_Error(t *testing.T) {
	op := builtin.HTTPRequestOp(nil)
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"url": "not a url"}},
	})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// ── soft_sleep ────────────────────────────────────────────────────────────────

func TestSoftSleep_Suspends(t *testing.T) {
	out, err := builtin.SoftSleep().Execute(context.Background(), ops.OpInput{
		ConnID: "conn-1",
		Node:   &tree.Node{Args: map[string]any{"softSleep": 2000}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SuspendKey == "" {
		t.Error("expected SuspendKey to be set")
	}
	if out.Pending == nil {
		t.Error("expected Pending to be set")
	}
	if out.Pending.Args["wake_at"] == "" {
		t.Error("expected wake_at in Pending.Args")
	}
}

func TestSoftSleep_ZeroMs_ReturnsError(t *testing.T) {
	_, err := builtin.SoftSleep().Execute(context.Background(), ops.OpInput{
		ConnID: "conn-2",
		Node:   &tree.Node{Args: map[string]any{"softSleep": 0}},
	})
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
}

// ── log ───────────────────────────────────────────────────────────────────────

func TestLog_NoError(t *testing.T) {
	op := builtin.Log()
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{"log": "test log message"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLog_VariableExpansion(t *testing.T) {
	op := builtin.Log()
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node:      &tree.Node{Args: map[string]any{"log": "caller=${caller}"}},
		Variables: map[string]string{"caller": "380501234567"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLog_EmptyMessage_NoError(t *testing.T) {
	op := builtin.Log()
	_, err := op.Execute(context.Background(), ops.OpInput{
		Node: &tree.Node{Args: map[string]any{}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
