package clog

import (
	"context"

	"github.com/opentracing/opentracing-go"
)

// StartSpanFromContext starts a span from context and returns the span
// and new context with the span. If a parent span exists in the context,
// the new span is created as a child of it; otherwise a root span is
// started via the global tracer.
func StartSpanFromContext(ctx context.Context,
	name string,
	opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span := opentracing.SpanFromContext(ctx)
	var tracer opentracing.Tracer
	if span == nil {
		tracer = opentracing.GlobalTracer()
	} else {
		opts = append(opts, opentracing.ChildOf(span.Context()))
		tracer = span.Tracer()
	}
	span = tracer.StartSpan(name, opts...)
	ctx = opentracing.ContextWithSpan(ctx, span)
	return span, ctx
}
