package clog_test

import (
	"testing"

	"github.com/nanjj/clog"
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
