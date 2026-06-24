package clog_test

import (
	"os"
	"strings"
	"testing"

	"github.com/nanjj/clog"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

func TestStartSpanFromContext(t *testing.T) {
	t.Run("NoSpan", func(t *testing.T) {
		tracer, err := clog.NewTracer("NoSpan")
		if err != nil {
			t.Fatal(err)
		}
		clog.SetGlobalTracer(tracer)
		ctx := t.Context()
		if tracer != clog.GlobalTracer() {
			t.Fatal(tracer)
		}
		t.Cleanup(func() {
			tracer.Close()
		})
		span, ctx := clog.StartSpanFromContext(ctx, "TestStartSpanFromContext01")
		span.LogKV("key01", "value01")
		span.LogKV("key02", "value02")
		defer span.Finish()
	})

	t.Run("StartSpanFromContext", func(t *testing.T) {
		tracer, err := clog.NewTracer("StartSpanFromContext")
		if err != nil {
			t.Fatal(err)
		}
		clog.SetGlobalTracer(tracer)
		ctx := t.Context()
		if tracer != clog.GlobalTracer() {
			t.Fatal(tracer)
		}
		t.Cleanup(func() {
			tracer.Close()
		})
		span, ctx := clog.StartSpanFromContext(ctx, "TestStartSpanFromContext01")
		defer span.Finish()
		span.LogKV("key01", "value01")
		span.LogKV("key02", "value02")
		span, ctx = clog.StartSpanFromContext(ctx, "TestStartSpanFromContext02")
		defer span.Finish()
		span.LogFields(log.String("key03", "value03"), log.String("key04", "value04"))
	})
}

func TestStartSpanFromContext_NonClogGlobalTracer(t *testing.T) {
	// Set a non-clog tracer as global — this should NOT panic
	dummy := opentracing.NoopTracer{}
	opentracing.SetGlobalTracer(dummy)

	ctx := t.Context()
	span, ctx := clog.StartSpanFromContext(ctx, "with-non-clog-global")
	span.LogKV("event", "works-with-noop")
	span.Finish()
}

func TestStartSpanFromContext_ChildSpan(t *testing.T) {
	tracer, err := clog.NewTracer("child-span-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tracer.Close()
	clog.SetGlobalTracer(tracer)

	// Create parent span
	span1, ctx := clog.StartSpanFromContext(t.Context(), "parent")
	span1.LogKV("event", "parent-started")
	defer span1.Finish()

	// Create child span from context
	span2, ctx := clog.StartSpanFromContext(ctx, "child")
	span2.LogKV("event", "child-work")
	defer span2.Finish()

	// Create grandchild
	span3, _ := clog.StartSpanFromContext(ctx, "grandchild")
	span3.LogKV("event", "grandchild-work")
	span3.Finish()
}

func TestStartSpanFromContext_MultipleRoots(t *testing.T) {
	tracer, err := clog.NewTracer("multi-root-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tracer.Close()
	clog.SetGlobalTracer(tracer)

	// Multiple independent root spans
	span1, _ := clog.StartSpanFromContext(t.Context(), "root-1")
	span1.SetTag("root", "1")
	span1.Finish()

	span2, _ := clog.StartSpanFromContext(t.Context(), "root-2")
	span2.SetTag("root", "2")
	span2.Finish()
}

func TestStartSpanFromContext_LogTypes(t *testing.T) {
	tracer, err := clog.NewTracer("log-types-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tracer.Close()
	clog.SetGlobalTracer(tracer)

	span, _ := clog.StartSpanFromContext(t.Context(), "log-types")
	span.LogFields(
		log.String("string", "hello"),
		log.Int("int", 42),
		log.Int64("int64", 1234567890),
		log.Float32("float32", 3.14),
		log.Float64("float64", 2.71828),
		log.Bool("bool", true),
		log.String("event", "all-types-logged"),
	)
	span.Finish()
}

func TestStartSpanFromContext_WithTags(t *testing.T) {
	tracer, err := clog.NewTracer("tags-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tracer.Close()
	clog.SetGlobalTracer(tracer)

	span, _ := clog.StartSpanFromContext(t.Context(), "tags-span")
	span.SetTag("string-tag", "value")
	span.SetTag("int-tag", 100)
	span.SetTag("bool-tag", true)
	span.SetTag("service", "test-service")
	span.Finish()
}

func TestStartSpanFromContext_ErrorSpan(t *testing.T) {
	tracer, err := clog.NewTracer("error-span-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tracer.Close()
	clog.SetGlobalTracer(tracer)

	span, _ := clog.StartSpanFromContext(t.Context(), "error-span")
	span.SetTag("error", true)
	span.LogFields(
		log.String("event", "error"),
		log.String("error.kind", "exception"),
		log.String("message", "something went wrong"),
	)
	span.Finish()
}

// --- ExtractFromEnv tests ---

func TestExtractFromEnv_MissingEnv(t *testing.T) {
	t.Setenv("CLOG_TRACEPARENT", "")
	t.Setenv("UBER_TRACE_ID", "")

	tr, err := clog.NewTracer("span-env-missing")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	if got := clog.ExtractFromEnv(t.Context()); got != nil {
		t.Errorf("ExtractFromEnv() = %v, want nil (no env vars)", got)
	}
}

func TestExtractFromEnv_CLOG_TRACEPARENT(t *testing.T) {
	t.Setenv("CLOG_TRACEPARENT",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	tr, err := clog.NewTracer("span-env-clog")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	spanCtx := clog.ExtractFromEnv(t.Context())
	if spanCtx == nil {
		t.Fatal("ExtractFromEnv() returned nil, want non-nil SpanContext")
	}
}

func TestExtractFromEnv_UBER_TRACE_ID(t *testing.T) {
	t.Setenv("UBER_TRACE_ID",
		"4bf92f3577b34da6a3ce929d0e0e4736:00f067aa0ba902b7:0:01")

	tr, err := clog.NewTracer("span-env-uber")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	spanCtx := clog.ExtractFromEnv(t.Context())
	if spanCtx == nil {
		t.Fatal("ExtractFromEnv() returned nil, want non-nil SpanContext")
	}
}

func TestExtractFromEnv_NoGlobalTracer(t *testing.T) {
	t.Setenv("CLOG_TRACEPARENT",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	// Ensure global tracer is NOT a clog.Tracer (so GlobalTracer() returns nil).
	old := opentracing.GlobalTracer()
	defer opentracing.SetGlobalTracer(old)
	opentracing.SetGlobalTracer(opentracing.NoopTracer{})

	if got := clog.ExtractFromEnv(t.Context()); got != nil {
		t.Errorf("ExtractFromEnv() = %v, want nil (no clog global tracer)", got)
	}
}

func TestExtractFromEnv_InvalidFormat(t *testing.T) {
	t.Setenv("CLOG_TRACEPARENT", "invalid-format-data-here")

	tr, err := clog.NewTracer("span-env-invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	if got := clog.ExtractFromEnv(t.Context()); got != nil {
		t.Errorf("ExtractFromEnv() = %v, want nil (invalid format)", got)
	}
}

// --- InjectToEnv tests ---

func TestInjectToEnv(t *testing.T) {
	tr, err := clog.NewTracer("inject-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	// Set a pre-existing value to test restore
	t.Setenv("CLOG_TRACEPARENT", "pre-existing-value")

	span, _ := clog.StartSpanFromContext(t.Context(), "inject-span")
	defer span.Finish()

	restore := clog.InjectToEnv(span)

	tp := os.Getenv("CLOG_TRACEPARENT")
	if tp == "" {
		t.Fatal("CLOG_TRACEPARENT not set after InjectToEnv")
	}
	if tp == "pre-existing-value" {
		t.Fatal("CLOG_TRACEPARENT was not updated by InjectToEnv")
	}
	// W3C format: "00-" + 32 hex + "-" + 16 hex + "-" + 2 hex = 55 chars
	if len(tp) < 50 {
		t.Errorf("CLOG_TRACEPARENT = %q, looks too short (len=%d)", tp, len(tp))
	}

	// Verify restore reverts to pre-existing value
	restore()
	if got := os.Getenv("CLOG_TRACEPARENT"); got != "pre-existing-value" {
		t.Errorf("after restore, CLOG_TRACEPARENT = %q, want %q", got, "pre-existing-value")
	}
}

func TestInjectToEnv_ThenStartSpanFromContext(t *testing.T) {
	// Simulate full round-trip: parent injects, child uses via StartSpanFromContext.
	tr, err := clog.NewTracer("inject-roundtrip")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	// Parent
	parent, _ := clog.StartSpanFromContext(t.Context(), "parent")
	restore := clog.InjectToEnv(parent)
	parent.Finish()
	defer restore()

	// Child — uses env var set by InjectToEnv above
	child, _ := clog.StartSpanFromContext(t.Context(), "child")
	child.LogKV("event", "roundtrip-success")
	child.Finish()
}

// --- TraceEnv tests ---

func TestTraceEnv(t *testing.T) {
	tr, err := clog.NewTracer("trace-env-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	span, _ := clog.StartSpanFromContext(t.Context(), "trace-env-span")
	defer span.Finish()

	env := clog.TraceEnv(span)
	if len(env) == 0 {
		t.Fatal("TraceEnv() returned empty slice")
	}

	var foundTraceparent bool
	for _, kv := range env {
		if strings.HasPrefix(kv, "CLOG_TRACEPARENT=") {
			foundTraceparent = true
			tp := strings.TrimPrefix(kv, "CLOG_TRACEPARENT=")
			if len(tp) < 50 {
				t.Errorf("traceparent too short: %q", tp)
			}
		}
	}
	if !foundTraceparent {
		t.Error("CLOG_TRACEPARENT not found in TraceEnv result")
	}
}

func TestTraceEnv_NilSpan(t *testing.T) {
	if got := clog.TraceEnv(nil); got != nil {
		t.Errorf("TraceEnv(nil) = %v, want nil", got)
	}
}

func TestTraceEnv_NoGlobalTracer(t *testing.T) {
	// TraceEnv uses the span's own tracer, not the global tracer.
	tr, err := clog.NewTracer("trace-env-no-global")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	span, _ := clog.StartSpanFromContext(t.Context(), "no-global-span")
	defer span.Finish()

	// Don't set global tracer — should still work since span has its own
	env := clog.TraceEnv(span)
	if len(env) == 0 {
		t.Fatal("TraceEnv() returned empty even without global tracer")
	}
}

func TestTraceEnv_Baggage(t *testing.T) {
	tr, err := clog.NewTracer("trace-env-baggage")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	span, _ := clog.StartSpanFromContext(t.Context(), "baggage-span")
	span.SetBaggageItem("user.id", "42")
	span.SetBaggageItem("request.id", "abc-123")
	defer span.Finish()

	env := clog.TraceEnv(span)

	var foundUserID, foundRequestID bool
	for _, kv := range env {
		switch {
		case kv == "CLOG_BAGGAGE_USER.ID=42":
			foundUserID = true
		case kv == "CLOG_BAGGAGE_REQUEST.ID=abc-123":
			foundRequestID = true
		}
	}
	if !foundUserID {
		t.Errorf("CLOG_BAGGAGE_USER.ID=42 not found in env: %q", env)
	}
	if !foundRequestID {
		t.Errorf("CLOG_BAGGAGE_REQUEST.ID=abc-123 not found in env: %q", env)
	}
}

// --- TraceparentOf tests ---

func TestTraceparentOf_ValidSpan(t *testing.T) {
	tr, err := clog.NewTracer("traceparent-of-valid")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	span, _ := clog.StartSpanFromContext(t.Context(), "traceparent-of-span")
	defer span.Finish()

	tp := clog.TraceparentOf(span)
	if tp == "" {
		t.Fatal("TraceparentOf() returned empty string")
	}
	// W3C: "00-" + 32 hex + "-" + 16 hex + "-" + 2 hex = 55 chars
	if len(tp) < 50 {
		t.Errorf("TraceparentOf() = %q, looks too short (len=%d)", tp, len(tp))
	}
	// Must start with "00-"
	if !strings.HasPrefix(tp, "00-") {
		t.Errorf("TraceparentOf() = %q, want 00- prefix", tp)
	}
	// Must have 4 dash-separated parts
	if parts := strings.SplitN(tp, "-", 4); len(parts) != 4 {
		t.Errorf("TraceparentOf() = %q, want 4 dash-separated parts, got %d", tp, len(parts))
	}
}

func TestTraceparentOf_NilSpan(t *testing.T) {
	if got := clog.TraceparentOf(nil); got != "" {
		t.Errorf("TraceparentOf(nil) = %q, want empty string", got)
	}
}

func TestTraceparentOf_NoGlobalTracer(t *testing.T) {
	// TraceparentOf uses the span's own tracer, not the global one.
	tr, err := clog.NewTracer("traceparent-of-no-global")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	span, _ := clog.StartSpanFromContext(t.Context(), "no-global-span")
	defer span.Finish()

	// Don't set global tracer — should still work
	tp := clog.TraceparentOf(span)
	if tp == "" {
		t.Fatal("TraceparentOf() returned empty even without global tracer")
	}
}

// --- SpanFromContext tests ---

func TestSpanFromContext(t *testing.T) {
	tr, err := clog.NewTracer("span-from-ctx")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	clog.SetGlobalTracer(tr)

	t.Run("no span in context", func(t *testing.T) {
		if got := clog.SpanFromContext(t.Context()); got != nil {
			t.Errorf("SpanFromContext() = %v, want nil", got)
		}
	})

	t.Run("span in context", func(t *testing.T) {
		span, ctx := clog.StartSpanFromContext(t.Context(), "test-span")
		defer span.Finish()
		if got := clog.SpanFromContext(ctx); got == nil {
			t.Error("SpanFromContext() = nil, want non-nil")
		}
	})
}
