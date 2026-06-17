package clog

import (
	"io"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

func init() {
	const (
		envJaegerSamplerType           = "JAEGER_SAMPLER_TYPE"
		envJaegerSamplerParam          = "JAEGER_SAMPLER_PARAM"
		envJaegerReporterMaxQueueSize  = "JAEGER_REPORTER_MAX_QUEUE_SIZE"
		envJaegerReporterFlushInterval = "JAEGER_REPORTER_FLUSH_INTERVAL"
	)
	if os.Getenv(envJaegerSamplerType) == "" {
		os.Setenv(envJaegerSamplerType, "const")
	}
	if os.Getenv(envJaegerSamplerParam) == "" {
		os.Setenv(envJaegerSamplerParam, "1")
	}
	if os.Getenv(envJaegerReporterMaxQueueSize) == "" {
		os.Setenv(envJaegerReporterMaxQueueSize, "64")
	}
	if os.Getenv(envJaegerReporterFlushInterval) == "" {
		os.Setenv(envJaegerReporterFlushInterval, "10s")
	}
}

type Tracer struct {
	opentracing.Tracer
	io.Closer
}

// NewTracer creates a new Jaeger tracer with the given service name.
// The name is used as JAEGER_SERVICE_NAME. Additional config.Option values
// (e.g. config.Tag) can be passed to customise the tracer.
func NewTracer(name string, opts ...config.Option) (tracer *Tracer, err error) {
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
