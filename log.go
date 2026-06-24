// Package clog provides contextual logging with OpenTracing tracing integration.
//
// Two layers of API:
//
// Convenience — Debug, Info, Warn, Error accept key-value args (like slog):
//
//	clog.Info(ctx, "request handled", "method", r.Method, "status", 200)
//
// Structured — LogAttrs accepts typed slog.Attr for precise type control:
//
//	clog.LogAttrs(ctx, slog.LevelInfo, "request handled",
//	    slog.String("method", r.Method),
//	    slog.Int("status", 200),
//	    slog.Duration("latency", dur))
//
// Both write to the configured slog logger AND the active OpenTracing span
// (when present in the context). To write to spans without slog, use the
// NewSpanHandler with slog.NewMultiHandler.

package clog

import (
	"context"
	"log/slog"
)

// Debug logs at slog.LevelDebug via LogAttrs.
// The args are key-value pairs, following slog's convention:
//
//	clog.Debug(ctx, "processing order", "order_id", id, "amount", amt)
func Debug(ctx context.Context, msg string, args ...any) {
	LogAttrs(ctx, slog.LevelDebug, msg, argsToAttrs(args)...)
}

// Info logs at slog.LevelInfo via LogAttrs.
func Info(ctx context.Context, msg string, args ...any) {
	LogAttrs(ctx, slog.LevelInfo, msg, argsToAttrs(args)...)
}

// Warn logs at slog.LevelWarn via LogAttrs.
func Warn(ctx context.Context, msg string, args ...any) {
	LogAttrs(ctx, slog.LevelWarn, msg, argsToAttrs(args)...)
}

// Error logs at slog.LevelError via LogAttrs.
func Error(ctx context.Context, msg string, args ...any) {
	LogAttrs(ctx, slog.LevelError, msg, argsToAttrs(args)...)
}

// argsToAttrs converts a key-value pair slice to []slog.Attr.
//
// It follows slog's own convention:
//   - Even-length slices are paired as (key, value).
//   - Non-string keys and odd trailing values are marked with "!BADKEY".
func argsToAttrs(args []any) []slog.Attr {
	if len(args) == 0 {
		return nil
	}
	n := len(args)
	if n%2 != 0 {
		n-- // last element handled separately
	}
	attrs := make([]slog.Attr, n/2)
	for i := 0; i < n; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			key = "!BADKEY"
		}
		attrs[i/2] = slog.Any(key, args[i+1])
	}
	if n < len(args) {
		attrs = append(attrs, slog.Any("!BADKEY", args[n]))
	}
	return attrs
}
