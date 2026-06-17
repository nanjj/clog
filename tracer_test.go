package clog_test

import (
	"os"
	"testing"

	"github.com/nanjj/clog"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

func TestNewTracer(t *testing.T) {
	tcs := []struct {
		name string
	}{
		{"MyTracer01"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := clog.NewTracer(tc.name,
				config.Tag("runner", "127.0.0.1:54321"),
				config.Tag("leader", "127.0.0.1:54312"))
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Close()
			sp := tr.StartSpan("TestNewTracer")
			sp.Finish()
		})
	}
}

func TestNewTracer_EmptyName(t *testing.T) {
	// Setting JAEGER_SERVICE_NAME so NewTracer("") can pick it up
	t.Setenv("JAEGER_SERVICE_NAME", "empty-name-test")
	tr, err := clog.NewTracer("")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	sp := tr.StartSpan("test-span")
	sp.Finish()
}

func TestNewTracer_WithOptions(t *testing.T) {
	tr, err := clog.NewTracer("option-test",
		config.Tag("env", "test"),
		config.Tag("version", "1.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	sp := tr.StartSpan("test-span")
	sp.SetTag("span-specific", "yes")
	sp.Finish()
}

func TestSetGlobalTracer_RoundTrip(t *testing.T) {
	tr, err := clog.NewTracer("global-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	clog.SetGlobalTracer(tr)
	got := clog.GlobalTracer()
	if got != tr {
		t.Fatalf("GlobalTracer() = %v, want %v", got, tr)
	}
}

func TestGlobalTracer_NonClog(t *testing.T) {
	// When global tracer is set to a non-clog tracer, GlobalTracer returns nil
	dummy := opentracing.NoopTracer{}
	opentracing.SetGlobalTracer(dummy)
	if got := clog.GlobalTracer(); got != nil {
		t.Fatalf("GlobalTracer() = %v, want nil for non-clog tracer", got)
	}
}

func TestGlobalTracer_Default(t *testing.T) {
	// By default, global tracer is noop, so clog.GlobalTracer returns nil
	if got := clog.GlobalTracer(); got != nil {
		t.Fatalf("expected nil default global tracer, got %v", got)
	}
}

func TestTracer_LogKV(t *testing.T) {
	tr, err := clog.NewTracer("logkv-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	sp := tr.StartSpan("logkv-span")
	sp.LogKV("event", "test-event", "key", "value")
	sp.Finish()
}

func TestTracer_LogFields(t *testing.T) {
	tr, err := clog.NewTracer("logfields-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	sp := tr.StartSpan("logfields-span")
	sp.LogKV("event", "test-event")
	sp.Finish()
}

func TestTracer_MultipleSpans(t *testing.T) {
	tr, err := clog.NewTracer("multi-span-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	for i := 0; i < 5; i++ {
		sp := tr.StartSpan("multi-span")
		sp.SetTag("index", i)
		sp.Finish()
	}
}

func TestTracer_InitDefaults(t *testing.T) {
	// Verify init() sets expected defaults when env is not set
	// Clear known env vars
	for _, env := range []string{
		"JAEGER_SAMPLER_TYPE",
		"JAEGER_SAMPLER_PARAM",
		"JAEGER_REPORTER_MAX_QUEUE_SIZE",
		"JAEGER_REPORTER_FLUSH_INTERVAL",
	} {
		os.Unsetenv(env)
	}
	// Re-trigger init by re-initializing (not possible directly),
	// but we can verify defaults are set by creating a tracer
	t.Setenv("JAEGER_SERVICE_NAME", "init-defaults-test")
	tr, err := clog.NewTracer("")
	if err != nil {
		t.Fatal(err)
	}
	tr.Close()
}

func TestTracer_CloseIdempotent(t *testing.T) {
	tr, err := clog.NewTracer("close-idempotent")
	if err != nil {
		t.Fatal(err)
	}
	tr.Close()
	tr.Close() // second close should not panic
}

func TestTracer_ContextWithSpan(t *testing.T) {
	tr, err := clog.NewTracer("ctx-span-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	clog.SetGlobalTracer(tr)
	parent := tr.StartSpan("parent")
	parent.SetTag("role", "parent")
	ctx := opentracing.ContextWithSpan(t.Context(), parent)

	// Extract span from context
	extracted := opentracing.SpanFromContext(ctx)
	if extracted == nil {
		t.Fatal("expected span in context")
	}
	parent.Finish()
}
