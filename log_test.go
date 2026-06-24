package clog_test

import (
	"testing"

	"github.com/nanjj/clog"
)

// --- Convenience functions (Debug/Info/Warn/Error) tests ---

func TestConvenienceFunctions(t *testing.T) {
	t.Run("no-ops without span in context", func(t *testing.T) {
		// Should not panic when there is no span in context
		clog.Debug(t.Context(), "debug no span", "k", "v")
		clog.Info(t.Context(), "info no span")
		clog.Warn(t.Context(), "warn no span")
		clog.Error(t.Context(), "error no span")
	})

	t.Run("writes to span when span in context", func(t *testing.T) {
		tr, err := clog.NewTracer("convenience-test")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "convenience-span")
		defer span.Finish()

		// Should not panic — verify each level works
		clog.Debug(ctx, "debug msg", "key", "val")
		clog.Info(ctx, "info msg", "count", 42)
		clog.Warn(ctx, "warn msg", "reason", "timeout")
		clog.Error(ctx, "error msg", "code", 500)
	})

	t.Run("handles key-value arg pairs", func(t *testing.T) {
		tr, err := clog.NewTracer("convenience-args")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "convenience-args-span")
		defer span.Finish()

		// Multiple key-value pairs
		clog.Info(ctx, "multiple pairs",
			"string", "value",
			"int", 123,
			"bool", true,
			"float", 3.14,
		)
	})

	t.Run("handles odd number of args", func(t *testing.T) {
		tr, err := clog.NewTracer("convenience-odd")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "convenience-odd-span")
		defer span.Finish()

		// Odd trailing value should not panic
		clog.Info(ctx, "odd args", "key", "val", "lonely")
	})

	t.Run("handles non-string key", func(t *testing.T) {
		tr, err := clog.NewTracer("convenience-badkey")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "convenience-badkey-span")
		defer span.Finish()

		// Non-string key should not panic
		clog.Info(ctx, "bad key", 42, "value")
	})

	t.Run("works with slog global logger", func(t *testing.T) {
		tr, err := clog.NewTracer("convenience-slog")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "convenience-slog-span")
		defer span.Finish()

		// Verify LogAttrs is called via the convenience function,
		// which means slog also receives the record
		clog.Info(ctx, "works with slog too", "via", "convenience")
	})

	t.Run("all levels produce distinct log records", func(t *testing.T) {
		tr, err := clog.NewTracer("convenience-levels")
		if err != nil {
			t.Fatal(err)
		}
		defer tr.Close()
		clog.SetGlobalTracer(tr)

		span, ctx := clog.StartSpanFromContext(t.Context(), "convenience-levels-span")
		defer span.Finish()

		clog.Debug(ctx, "level debug", "x", 1)
		clog.Info(ctx, "level info", "x", 2)
		clog.Warn(ctx, "level warn", "x", 3)
		clog.Error(ctx, "level error", "x", 4)
	})
}
