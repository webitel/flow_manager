package builtin_test

import (
	"context"
	"testing"
	"time"

	"github.com/webitel/flow_manager/internal/runtime/ops/builtin"
	"github.com/webitel/flow_manager/internal/runtime/ops/testutil"
)

func TestGetTimeOp_RFC3339(t *testing.T) {
	before := time.Now().Truncate(time.Second)

	out, err := builtin.GetTime().Execute(context.Background(),
		testutil.MakeInput(
			map[string]any{"set": "result_t", "timezone": "Europe/Kyiv"},
			nil,
		),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, ok := out.SetVars["result_t"]
	if !ok {
		t.Fatal("SetVars[result_t] not set")
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("result %q is not RFC3339: %v", raw, err)
	}

	after := time.Now().Add(time.Second)
	if parsed.Before(before) || parsed.After(after) {
		t.Errorf("timestamp %q outside expected range [%s, %s]", raw, before, after)
	}

	// Verify offset matches Europe/Kyiv (+02:00 or +03:00 depending on DST).
	loc, _ := time.LoadLocation("Europe/Kyiv")
	_, kyivOffset := time.Now().In(loc).Zone()
	_, parsedOffset := parsed.Zone()
	if kyivOffset != parsedOffset {
		t.Errorf("timezone offset: got %d, want %d", parsedOffset, kyivOffset)
	}
}

func TestGetTimeOp_UTC(t *testing.T) {
	out, err := builtin.GetTime().Execute(context.Background(),
		testutil.MakeInput(map[string]any{"set": "ts"}, nil),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.SetVars["ts"]
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("result %q is not RFC3339: %v", raw, err)
	}

	_, offset := parsed.Zone()
	if offset != 0 {
		t.Errorf("expected UTC offset 0, got %d", offset)
	}
}

func TestGetTimeOp_FlowTimezone(t *testing.T) {
	// When op-level timezone is empty, fall back to in.Timezone.
	in := testutil.MakeInput(map[string]any{"set": "ts"}, nil)
	in.Timezone = "America/New_York"

	out, err := builtin.GetTime().Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.SetVars["ts"]
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("result %q is not RFC3339: %v", raw, err)
	}

	loc, _ := time.LoadLocation("America/New_York")
	_, nyOffset := time.Now().In(loc).Zone()
	_, parsedOffset := parsed.Zone()
	if nyOffset != parsedOffset {
		t.Errorf("timezone offset: got %d, want %d (America/New_York)", parsedOffset, nyOffset)
	}
}

func TestGetTimeOp_VariableExpansion(t *testing.T) {
	out, err := builtin.GetTime().Execute(context.Background(),
		testutil.MakeInput(
			map[string]any{"set": "ts", "timezone": "${tz}"},
			map[string]string{"tz": "Europe/Kyiv"},
		),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, out.SetVars["ts"]); err != nil {
		t.Errorf("result %q is not RFC3339: %v", out.SetVars["ts"], err)
	}
}

func TestGetTimeOp_InvalidTimezone(t *testing.T) {
	_, err := builtin.GetTime().Execute(context.Background(),
		testutil.MakeInput(
			map[string]any{"set": "ts", "timezone": "Not/ATimezone"},
			nil,
		),
	)
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

func TestGetTimeOp_NoSet(t *testing.T) {
	// set="" → op is a no-op, no error.
	out, err := builtin.GetTime().Execute(context.Background(),
		testutil.MakeInput(map[string]any{"timezone": "UTC"}, nil),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.SetVars) != 0 {
		t.Errorf("expected no vars, got %v", out.SetVars)
	}
}
