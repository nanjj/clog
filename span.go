package clog

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/opentracing/opentracing-go"
)

// StartSpanFromContext starts a span from context and returns the span
// and new context with the span. If a parent span exists in the context,
// the new span is created as a child of it. Otherwise, if a remote parent
// SpanContext has been propagated via environment variables (see SpanFromEnv),
// the new span is created as a child of that remote parent. Otherwise a
// root span is started via the global tracer.
func StartSpanFromContext(ctx context.Context,
	name string,
	opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {

	var tracer opentracing.Tracer

	// 1. Try parent span from context
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		opts = append(opts, opentracing.ChildOf(parent.Context()))
		tracer = parent.Tracer()
	} else {
		// 2. Try remote parent from environment variables
		tracer = opentracing.GlobalTracer()
		if spanCtx := SpanFromEnv(ctx); spanCtx != nil {
			opts = append(opts, opentracing.ChildOf(spanCtx))
		}
	}

	span := tracer.StartSpan(name, opts...)
	ctx = opentracing.ContextWithSpan(ctx, span)
	return span, ctx
}

// SpanFromContext returns a span from context if one exists.
func SpanFromContext(ctx context.Context) opentracing.Span {
	return opentracing.SpanFromContext(ctx)
}

// SpanFromEnv extracts a remote parent SpanContext from environment variables
// propagated by a parent process. Returns nil if no trace context is found.
//
// Supported env vars (checked in order):
//   - CLOG_TRACEPARENT  (W3C Trace Context, preferred)
//   - UBER_TRACE_ID     (Jaeger legacy, backward-compatible)
//
// CLOG_TRACEPARENT format (W3C):
//
//	00-{trace-id-32hex}-{span-id-16hex}-{flags-2hex}
//
// Example:
//
//	00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
//
// UBER_TRACE_ID format (Jaeger legacy):
//
//	{trace-id}:{span-id}:{parent-span-id}:{flags}
func SpanFromEnv(_ context.Context) opentracing.SpanContext {
	tracer := GlobalTracer()
	if tracer == nil {
		return nil
	}

	carrier := opentracing.TextMapCarrier{}

	switch {
	case os.Getenv("CLOG_TRACEPARENT") != "":
		uber := traceparentToJaeger(os.Getenv("CLOG_TRACEPARENT"))
		if uber == "" {
			return nil
		}
		carrier["uber-trace-id"] = uber
	case os.Getenv("UBER_TRACE_ID") != "":
		carrier["uber-trace-id"] = os.Getenv("UBER_TRACE_ID")
	default:
		return nil
	}

	spanCtx, err := tracer.Extract(opentracing.TextMap, carrier)
	if err != nil {
		return nil
	}
	return spanCtx
}

// traceparentToJaeger converts W3C traceparent to Jaeger uber-trace-id format.
//
// W3C:   00-{trace-id-32hex}-{span-id-16hex}-{flags-2hex}
// Jaeger: {trace-id}:{span-id}:0:{flags}
//
// Returns empty string on parse failure.
func traceparentToJaeger(tp string) string {
	// W3C: 00-{trace-id-32hex}-{span-id-16hex}-{flags-2hex}
	parts := strings.SplitN(tp, "-", 4)
	if len(parts) != 4 || parts[0] != "00" {
		return ""
	}
	// parts: [version, traceID, spanID, flags]
	if parts[1] == "" || parts[2] == "" {
		return ""
	}
	// Jaeger flags is 1 hex byte (01 = sampled, 00 = not).
	flags := parts[3]
	if len(flags) > 2 {
		flags = flags[:2]
	}
	if flags != "01" && flags != "00" {
		flags = "01" // assume sampled
	}
	return fmt.Sprintf("%s:%s:0:%s", parts[1], parts[2], flags)
}

// InjectToEnv injects the span's context into environment variables so that
// child processes can continue the trace via StartSpanFromContext / SpanFromEnv.
//
// Call before exec'ing a child process:
//
//	span, _ := clog.StartSpanFromContext(ctx, "parent-op")
//	defer span.Finish()
//	clog.InjectToEnv(span)
//	// exec child ...
//
// Sets CLOG_TRACEPARENT (W3C format) and CLOG_BAGGAGE_* for baggage items.
func InjectToEnv(span opentracing.Span) {
	tracer := span.Tracer()
	if tracer == nil {
		return
	}

	carrier := opentracing.TextMapCarrier{}
	if err := tracer.Inject(span.Context(), opentracing.TextMap, carrier); err != nil {
		return
	}

	// Core trace context as W3C Traceparent
	if uber, ok := carrier["uber-trace-id"]; ok {
		tp := jaegerToTraceparent(uber)
		if tp != "" {
			os.Setenv("CLOG_TRACEPARENT", tp)
		}
	}

	// Baggage items
	for k, v := range carrier {
		if !strings.HasPrefix(k, "uberctx-") {
			continue
		}
		key := strings.TrimPrefix(k, "uberctx-")
		osenv := fmt.Sprintf("CLOG_BAGGAGE_%s", strings.ToUpper(key))
		os.Setenv(osenv, v)
	}
}

// jaegerToTraceparent converts Jaeger uber-trace-id to W3C traceparent format.
//
// Jaeger: {trace-id}:{span-id}:{parent-span-id}:{flags}
// W3C:    00-{trace-id}-{span-id}-{flags}
//
// Returns empty string on parse failure.
func jaegerToTraceparent(uber string) string {
	parts := strings.SplitN(uber, ":", 4)
	if len(parts) != 4 {
		return ""
	}
	traceID, spanID := parts[0], parts[1]
	if traceID == "" || spanID == "" {
		return ""
	}

	// Pad 64-bit trace IDs to 32 hex chars for W3C
	if len(traceID) < 32 {
		pad := strings.Repeat("0", 32-len(traceID))
		traceID = pad + traceID
	}

	// Jaeger uses 1-byte flags, W3C uses 2 hex chars
	flags := parts[3]
	if len(flags) == 1 {
		flags = "0" + flags
	}

	return fmt.Sprintf("00-%s-%s-%s", traceID, spanID, flags)
}

