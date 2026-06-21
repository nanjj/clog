// Package clog provides structured log events via OpenTracing/Jaeger.
//
// Basic usage:
//
//	tracer, err := clog.NewTracer("my-service")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tracer.Close()
//	clog.SetGlobalTracer(tracer)
//
//	span, ctx := clog.StartSpanFromContext(context.Background(), "operation")
//	defer span.Finish()
//	span.LogKV("event", "work-done", "count", 42)
//
// Environment variables set by applyJaegerDefaults at first NewTracer call
// (only when unset):
//
//	JAEGER_SAMPLER_TYPE    const
//	JAEGER_SAMPLER_PARAM   1
//	JAEGER_TRACEID_128BIT  true
//
// Reporter env vars MaxQueueSize and FlushInterval are NOT set by
// applyJaegerDefaults; the Jaeger client's built-in defaults (100, 1s)
// are used instead.
//
// Set any of these before calling NewTracer to override the default.
// Use NewTracerWithOptions for programmatic control.
package clog

import (
	"io"
	"os"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

// Default environment variable keys set by applyJaegerDefaults.
const (
	envJaegerSamplerType  = "JAEGER_SAMPLER_TYPE"
	envJaegerSamplerParam = "JAEGER_SAMPLER_PARAM"
	envJaeger128Bit       = "JAEGER_TRACEID_128BIT"
)

var jaegerDefaultsOnce sync.Once

// applyJaegerDefaults sets sensible Jaeger defaults for env vars that are
// not already set. Called once at the first NewTracer call, so users can
// set any of these before calling NewTracer to override.
//
// Reporter defaults (MaxQueueSize=100, FlushInterval=1s) are NOT set here
// because the Jaeger client already provides well-tuned built-in values.
// Overriding them would risk queue overflow and span loss.
func applyJaegerDefaults() {
	jaegerDefaultsOnce.Do(func() {
		if os.Getenv(envJaegerSamplerType) == "" {
			os.Setenv(envJaegerSamplerType, "const")
		}
		if os.Getenv(envJaegerSamplerParam) == "" {
			os.Setenv(envJaegerSamplerParam, "1")
		}
		// 128-bit trace IDs are the modern standard (W3C Trace Context,
		// OpenTelemetry). Only override when unset so users can opt out
		// via export JAEGER_TRACEID_128BIT=false.
		if os.Getenv(envJaeger128Bit) == "" {
			os.Setenv(envJaeger128Bit, "true")
		}
	})
}

// Tracer wraps an OpenTracing tracer with its io.Closer.
// Call Close to flush pending spans before the process exits.
type Tracer struct {
	opentracing.Tracer
	io.Closer
}

// NewTracer creates a new Jaeger tracer with the given service name.
// The name is used as JAEGER_SERVICE_NAME. Additional config.Option values
// (e.g. config.Tag) can be passed to customise the tracer.
func NewTracer(name string, opts ...config.Option) (tracer *Tracer, err error) {
	applyJaegerDefaults()
	c, err := config.FromEnv()
	if err != nil {
		return
	}
	if name != "" {
		c.ServiceName = name
	}
	tr, closer, err := c.NewTracer(opts...)
	if err == nil {
		tracer = &Tracer{
			Tracer: tr,
			Closer: closer,
		}
	}
	return
}

// Option programmatically overrides Jaeger tracer configuration.
// Options are applied before config.FromEnv(), so they take precedence
// over environment variables (but not over values set explicitly before
// calling NewTracerWithOptions).
type Option func()

// With128Bit enables or disables 128-bit trace IDs.
// Enabled by default. Pass With128Bit(false) to use legacy
// 64-bit trace IDs.
func With128Bit(enabled bool) Option {
	return func() {
		if enabled {
			os.Setenv(envJaeger128Bit, "true")
		} else {
			os.Setenv(envJaeger128Bit, "false")
		}
	}
}

// NewTracerWithOptions creates a new Jaeger tracer with programmatic
// option overrides. Options are applied before reading environment
// variables, so they take precedence over env vars.
func NewTracerWithOptions(name string, opts ...Option) (*Tracer, error) {
	for _, opt := range opts {
		opt()
	}
	return NewTracer(name)
}

// CloseTracer closes a *Tracer, flushing any pending spans.
// Safe to call with nil — returns nil immediately.
func CloseTracer(tracer *Tracer) error {
	if tracer == nil {
		return nil
	}
	return tracer.Close()
}

// SetGlobalTracer registers the tracer as the OpenTracing global tracer.
// Must be called before StartSpanFromContext when there is no parent span.
func SetGlobalTracer(tracer *Tracer) {
	opentracing.SetGlobalTracer(tracer)
}

// GlobalTracer returns the previously registered global tracer,
// or nil if none was set or if the global tracer is not a clog.Tracer.
func GlobalTracer() (tracer *Tracer) {
	tracer, ok := opentracing.GlobalTracer().(*Tracer)
	if !ok {
		return nil
	}
	return tracer
}
