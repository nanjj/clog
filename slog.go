// Package clog — contextual logging with OpenTracing integration.
//
// Two paths from code to span:
//
//  1. Direct:  LogAttrs(ctx, level, msg, attrs...slog.Attr)
//     Writes to slog.Default() AND the active OpenTracing span.
//
//  2. Handler: NewSpanHandler() → slog.Handler
//     Plumbed via slog.SetDefault or slog.New.
//     Every slog call reaching that handler writes to the active span.
//
// Both are safe to call anywhere — they silently no-op when there is no
// active span in the context.
//
// # Span event naming convention
//
// The span event "event" field is always set to the log message (msg).
// This keeps event names human-readable and searchable in trace UIs.
// Callers should use descriptive messages that double as event names,
// e.g. "request started", "cache hit", "handleLogin".
//
// Attributes are preserved with their slog types mapped to the closest
// OpenTracing log.Field type (string, int64, float64, bool, duration).
// Groups are flattened with dot-separated keys (e.g. "http.method").
//
// Example:
//
//	clog.LogAttrs(ctx, slog.LevelInfo, "request handled",
//	    slog.String("method", r.Method),
//	    slog.Int("status", 200),
//	    slog.Duration("latency", dur),
//	)
//
// This writes a span event: event="request handled", method=GET, status=200, latency=1.2s.

package clog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

// LogAttrs writes a log record at the given level via slog, and also writes
// the message and attributes to the active OpenTracing span in ctx (if any).
//
// The span event uses the convention event=<msg> (see package doc).
// Attributes are preserved with their slog types mapped to the closest
// OpenTracing log.Field type.
//
// Example:
//
//	clog.LogAttrs(ctx, slog.LevelInfo, "request handled",
//	    slog.String("method", r.Method),
//	    slog.Int("status", 200),
//	    slog.Duration("latency", dur),
//	)
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	// Always write to slog. The default handler's level filtering determines
	// whether the output actually reaches the configured sink.
	slog.LogAttrs(ctx, level, msg, attrs...)

	// Always write to span when one exists in context — this is independent
	// of slog's level filtering. The caller explicitly chose to LogAttrs,
	// so we honour the intent.
	if span := opentracing.SpanFromContext(ctx); span != nil {
		fields := make([]log.Field, 0, 2+len(attrs))
		fields = append(fields, log.String("event", msg))
		fields = append(fields, expandAttrs(attrs, "")...)
		span.LogFields(fields...)
	}
}

// NewSpanHandler returns a slog.Handler that writes log records to the
// active OpenTracing span found in the context passed to Handle.
//
// The handler does not write to any output sink (stdout, stderr, file).
// To write to both the span and a sink, combine handlers via
// slog.NewMultiHandler:
//
//	slog.SetDefault(slog.New(
//	    slog.NewMultiHandler(
//	        clog.NewSpanHandler(),
//	        slog.NewTextHandler(os.Stderr, nil),
//	    ),
//	))
//
// WithAttrs and WithGroup are supported. Grouped attributes are flattened
// with dot-separated keys (e.g. "http.method") for OpenTracing span events.
func NewSpanHandler() slog.Handler {
	return &spanHandler{}
}

// spanHandler implements slog.Handler by forwarding records to an
// OpenTracing span obtained from the context at Handle-time.
type spanHandler struct {
	attrs  []slog.Attr   // accumulated via WithAttrs
	groups []string      // accumulated via WithGroup
}

// Enabled returns true when there is a span in ctx, so Handle gets a
// chance to write. Without the span the handler is a no-op, but we
// still report Enabled on best-effort: the caller's decision.
func (h *spanHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// We are always "enabled" at the handler level — Handle itself
	// gates on span presence. This ensures that even if the span
	// is not yet in context at Enabled-time (e.g. a middleware writes
	// the span into context after the logger is configured), Handle
	// will still fire when the span appears.
	return true
}

// Handle writes the slog record to the active OpenTracing span in ctx.
// No-op when ctx contains no span.
//
// The span event field is set to r.Message (consistent with LogAttrs).
// Level is written as a separate field. See package doc for the full
// naming convention.
func (h *spanHandler) Handle(ctx context.Context, r slog.Record) error {
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return nil
	}

	// Reserve reasonable capacity: 2 fixed fields + accumulated attrs +
	// attrs on the record.
	prefix := h.groupPrefix()

	// Count attrs for capacity estimate.
	attrCount := len(h.attrs) + r.NumAttrs()
	fields := make([]log.Field, 0, 3+attrCount)

	// Event message — consistent with LogAttrs: event=<msg>
	fields = append(fields, log.String("event", r.Message))

	// Level
	fields = append(fields, log.String("level", r.Level.String()))

	// Accumulated attrs (from WithAttrs)
	for _, attr := range h.attrs {
		fields = append(fields, expandAttr(attr, prefix)...)
	}

	// Per-record attrs
	r.Attrs(func(a slog.Attr) bool {
		fields = append(fields, expandAttr(a, prefix)...)
		return true
	})

	span.LogFields(fields...)
	return nil
}

// WithAttrs returns a new handler with the given attrs pre-attached.
func (h *spanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := &spanHandler{
		attrs:  make([]slog.Attr, 0, len(h.attrs)+len(attrs)),
		groups: h.groups,
	}
	h2.attrs = append(h2.attrs, h.attrs...)
	h2.attrs = append(h2.attrs, attrs...)
	return h2
}

// WithGroup returns a new handler with the given group name prepended
// to subsequent attribute keys. Groups are flattened with dot-separated
// keys for OpenTracing span events.
func (h *spanHandler) WithGroup(name string) slog.Handler {
	h2 := &spanHandler{
		attrs:  h.attrs,
		groups: make([]string, 0, len(h.groups)+1),
	}
	h2.groups = append(h2.groups, h.groups...)
	h2.groups = append(h2.groups, name)
	return h2
}

// groupPrefix returns the dot-separated prefix from accumulated groups,
// with a trailing dot. Returns "" when there are no groups.
func (h *spanHandler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

// expandAttrs converts a slice of slog.Attr to OpenTracing log.Fields,
// handling nested groups by flattening keys with dot-separated prefixes.
func expandAttrs(attrs []slog.Attr, prefix string) []log.Field {
	if len(attrs) == 0 {
		return nil
	}
	fields := make([]log.Field, 0, len(attrs))
	for _, attr := range attrs {
		fields = append(fields, expandAttr(attr, prefix)...)
	}
	return fields
}

// expandAttr converts a single slog.Attr to one or more log.Fields.
// Groups are recursively flattened with dot-separated keys.
func expandAttr(attr slog.Attr, prefix string) []log.Field {
	if attr.Key == "" {
		return nil
	}

	key := prefix + attr.Key
	val := attr.Value

	switch val.Kind() {
	case slog.KindGroup:
		children := val.Group()
		fields := make([]log.Field, 0, len(children))
		for _, child := range children {
			fields = append(fields, expandAttr(child, key+".")...)
		}
		return fields

	case slog.KindDuration:
		return []log.Field{log.String(key, val.Duration().String())}

	case slog.KindTime:
		return []log.Field{log.String(key, val.Time().Format(time.RFC3339Nano))}

	case slog.KindAny:
		return []log.Field{log.String(key, fmt.Sprintf("%v", val.Any()))}

	case slog.KindUint64:
		return []log.Field{log.Uint64(key, val.Uint64())}

	default:
		return []log.Field{slogValueToField(key, val)}
	}
}

// slogValueToField converts a non-nested, non-Any slog.Value to a log.Field.
func slogValueToField(key string, val slog.Value) log.Field {
	switch val.Kind() {
	case slog.KindString:
		return log.String(key, val.String())
	case slog.KindInt64:
		return log.Int64(key, val.Int64())
	case slog.KindFloat64:
		return log.Float64(key, val.Float64())
	case slog.KindBool:
		return log.Bool(key, val.Bool())
	default:
		return log.String(key, fmt.Sprintf("%v", val.Any()))
	}
}
