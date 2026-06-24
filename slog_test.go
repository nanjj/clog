package clog_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/nanjj/clog"
	"github.com/opentracing/opentracing-go"
)

// --- LogAttrs tests ---

func TestLogAttrs(t *testing.T) {
	t.Run("no-ops without span in context", func(t *testing.T) {
		// Should not panic when there is no span in context
		clog.LogAttrs(t.Context(), slog.LevelInfo, "test",
			slog.String("key", "value"))
	})

	t.Run("writes to both slog and span when span in context", func(t *testing.T) {
		tr, err := clog.NewTracer("logattrs-test")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "logattrs-span")
		defer span.Finish()

		// Should not panic
		clog.LogAttrs(ctx, slog.LevelInfo, "request handled",
			slog.String("method", "GET"),
			slog.Int("status", 200),
			slog.String("path", "/api/v1/test"),
		)
	})

	t.Run("writes with all attribute types", func(t *testing.T) {
		tr, err := clog.NewTracer("logattrs-types")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "logattrs-types-span")
		defer span.Finish()

		clog.LogAttrs(ctx, slog.LevelDebug, "all types",
			slog.Bool("flag", true),
			slog.Int("count", 42),
			slog.Int64("big", 1<<60),
			slog.Uint64("unsigned", 1<<63),
			slog.Float64("pi", 3.14159),
			slog.String("name", "test"),
			slog.Duration("latency", 1500000000), // 1.5s
			slog.Time("timestamp", mustParseTime(t, "2026-06-24T12:00:00Z")),
			slog.Group("http",
				slog.String("method", "POST"),
				slog.Int("status", 201),
			),
		)
	})

	t.Run("various log levels", func(t *testing.T) {
		tr, err := clog.NewTracer("logattrs-levels")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "logattrs-levels-span")
		defer span.Finish()

		clog.LogAttrs(ctx, slog.LevelDebug, "debug message", slog.Int("x", 1))
		clog.LogAttrs(ctx, slog.LevelInfo, "info message", slog.Int("x", 2))
		clog.LogAttrs(ctx, slog.LevelWarn, "warn message", slog.Int("x", 3))
		clog.LogAttrs(ctx, slog.LevelError, "error message", slog.Int("x", 4))
	})
}

// --- NewSpanHandler tests ---

func TestNewSpanHandler(t *testing.T) {
	t.Run("no-ops when no span in context", func(t *testing.T) {
		h := clog.NewSpanHandler()
		logger := slog.New(h)

		// Should not panic when context has no span
		logger.Info("no span here", slog.String("key", "val"))
	})

	t.Run("writes span event when span in context", func(t *testing.T) {
		tr, err := clog.NewTracer("spanhandler-test")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "spanhandler-span")
		defer span.Finish()

		h := clog.NewSpanHandler()
		logger := slog.New(h)

		// Should write to the span via the handler
		logger.LogAttrs(ctx, slog.LevelInfo, "handler test",
			slog.String("key", "value"),
			slog.Int("count", 10),
		)
	})

	t.Run("with pre-attached attributes via WithAttrs", func(t *testing.T) {
		tr, err := clog.NewTracer("spanhandler-attrs")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "spanhandler-attrs-span")
		defer span.Finish()

		h := clog.NewSpanHandler().
			WithAttrs([]slog.Attr{
				slog.String("service", "test-svc"),
				slog.String("env", "testing"),
			})
		logger := slog.New(h)

		logger.LogAttrs(ctx, slog.LevelInfo, "with attrs",
			slog.Int("request_id", 42),
		)
	})

	t.Run("with group via WithGroup", func(t *testing.T) {
		tr, err := clog.NewTracer("spanhandler-group")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "spanhandler-group-span")
		defer span.Finish()

		h := clog.NewSpanHandler().
			WithGroup("http").
			WithAttrs([]slog.Attr{
				slog.String("method", "PUT"),
			})
		logger := slog.New(h)

		logger.LogAttrs(ctx, slog.LevelInfo, "grouped",
			slog.Int("status", 200),
		)
	})

	t.Run("multi-level grouping", func(t *testing.T) {
		tr, err := clog.NewTracer("spanhandler-multigroup")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "spanhandler-multigroup-span")
		defer span.Finish()

		h := clog.NewSpanHandler().
			WithGroup("request").
			WithGroup("http")
		logger := slog.New(h)

		logger.LogAttrs(ctx, slog.LevelInfo, "deep group",
			slog.String("method", "DELETE"),
		)
	})

	t.Run("works with slog.NewMultiHandler", func(t *testing.T) {
		tr, err := clog.NewTracer("spanhandler-multi")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "spanhandler-multi-span")
		defer span.Finish()

		// Combine span handler with a text handler (discard output)
		h := slog.NewMultiHandler(
			clog.NewSpanHandler(),
			slog.NewTextHandler(discardWriter{}, nil),
		)
		logger := slog.New(h)

		logger.LogAttrs(ctx, slog.LevelInfo, "multi handler works",
			slog.Int("n", 1),
		)
	})

	t.Run("without global tracer set", func(t *testing.T) {
		// NewSpanHandler should work even without a global clog tracer
		// as long as there's a span in context
		tr, err := clog.NewTracer("spanhandler-no-global")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		// Don't set as global — use span from tracer directly

		span, ctx := opentracing.StartSpanFromContextWithTracer(
			t.Context(), tr, "no-global-span",
		)
		defer span.Finish()

		h := clog.NewSpanHandler()
		logger := slog.New(h)

		logger.LogAttrs(ctx, slog.LevelInfo, "no global tracer works",
			slog.String("status", "ok"),
		)
	})
}

// --- Helpers ---

// discardWriter implements io.Writer by discarding all writes.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// mustParseTime parses an RFC3339 time string, panicking on failure.
// Only used in test helpers where invalid input is a test bug.
func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("invalid test time %q: %v", s, err)
	}
	return ts
}
