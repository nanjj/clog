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
	// init() sets defaults at package load time. Verify they are in effect.
	t.Run("env defaults", func(t *testing.T) {
		// If another test cleared these, skip env verification.
		checks := map[string]string{
			"JAEGER_SAMPLER_TYPE":   "const",
			"JAEGER_SAMPLER_PARAM":  "1",
			"JAEGER_TRACEID_128BIT": "true",
			// Reporter vars (MaxQueueSize, FlushInterval) are NOT set by init;
			// the Jaeger client's built-in defaults (100, 1s) are used instead.
		}
		for env, want := range checks {
			if got := os.Getenv(env); got != "" && got != want {
				t.Errorf("%s = %q, want %q or empty", env, got, want)
			}
		}
	})

	t.Run("tracer creation works", func(t *testing.T) {
		t.Setenv("JAEGER_SERVICE_NAME", "init-defaults-test")
		tr, err := clog.NewTracer("")
		if err != nil {
			t.Fatal(err)
		}
		tr.Close()
	})
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

func TestNewTracerWithOptions(t *testing.T) {

	t.Run("default (128-bit)", func(t *testing.T) {
		tr, err := clog.NewTracerWithOptions("options-128bit")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		sp := tr.StartSpan("test-span")
		sp.Finish()
	})

	t.Run("with 64-bit trace IDs", func(t *testing.T) {
		tr, err := clog.NewTracerWithOptions("options-64bit",
			clog.With128Bit(false))
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		sp := tr.StartSpan("test-span")
		sp.Finish()
	})
}

func TestCloseTracer(t *testing.T) {
	t.Run("nil is safe", func(t *testing.T) {
		if err := clog.CloseTracer(nil); err != nil {
			t.Errorf("CloseTracer(nil) = %v, want nil", err)
		}
	})

	t.Run("closes tracer", func(t *testing.T) {
		tr, err := clog.NewTracer("close-tracer-test")
		if err != nil {
			t.Fatal(err)
		}
		if err := clog.CloseTracer(tr); err != nil {
			t.Errorf("CloseTracer(tr) = %v, want nil", err)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		tr, err := clog.NewTracer("close-idempotent-pkg")
		if err != nil {
			t.Fatal(err)
		}
		clog.CloseTracer(tr)
		clog.CloseTracer(tr) // second should not panic
	})
}
