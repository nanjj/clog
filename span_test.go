package clog_test

import (
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
