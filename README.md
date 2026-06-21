# clog

`clog` sends structured log events to an OpenTracing-compatible backend (e.g., Jaeger) by attaching them to active spans. It wraps [jaeger-client-go](https://github.com/jaegertracing/jaeger-client-go) with sensible defaults so you can start tracing with minimal setup.

## Install

```
go get github.com/nanjj/clog
```

## Quick Start

### 1. Initialise the tracer (usually in `main`)

```go
tracer, err := clog.NewTracer("my-service")
if err != nil {
    panic(err)
}
clog.SetGlobalTracer(tracer)
defer tracer.Close() // flush pending spans before exit
```

### 2. Create spans and attach logs

```go
span, ctx := clog.StartSpanFromContext(ctx, "CreateOrder")
defer span.Finish()

// Log structured key-value pairs on the span
span.LogKV("event", "order_created", "order_id", "12345")

// Or use the more type-safe log.Field API
span.LogFields(
    log.String("event", "payment_processed"),
    log.Int("amount_cents", 2999),
)
```

## Environment Variables

`clog` reads standard Jaeger environment variables via `config.FromEnv()`. Values set before importing clog take priority over the built-in defaults.

|Variable                        |Default                    |Description                                                      |
|--------------------------------|---------------------------|-----------------------------------------------------------------|
|`JAEGER_SERVICE_NAME`           |(set via `NewTracer(name)`)|Service name shown in traces                                     |
|`JAEGER_SAMPLER_TYPE`           |`const`                    |Sampler type (`const`, `probabilistic`, `ratelimiting`, `remote`)|
|`JAEGER_SAMPLER_PARAM`          |`1`                        |Sampler parameter (e.g. `1` = sample all)                        |
|`JAEGER_REPORTER_MAX_QUEUE_SIZE`|`64`                       |Max spans in the send queue                                      |
|`JAEGER_REPORTER_FLUSH_INTERVAL`|`10s`                      |How often to flush spans to the agent                            |
|`JAEGER_TRACEID_128BIT`         |`true`                     |128-bit trace IDs (W3C Trace Context standard)                   |
|`JAEGER_AGENT_HOST`             |`localhost`                |Jaeger agent UDP host                                            |
|`JAEGER_AGENT_PORT`             |`6831`                     |Jaeger agent UDP port                                            |
|`JAEGER_REPORTER_LOG_SPANS`     |(not set)                  |Log reporter activity when set to `true`                         |
|`JAEGER_DISABLED`               |(not set)                  |Disable tracing when set to `true`                               |
|`JAEGER_TAGS`                   |(not set)                  |Comma-separated `key=value` tags applied to all spans            |

### Customising the agent endpoint

By default spans are sent to `localhost:6831` (Jaeger agent UDP socket). To change:

```sh
export JAEGER_AGENT_HOST=jaeger.example.com
export JAEGER_AGENT_PORT=6831
```

### Additional tags at construction time

```go
tracer, err := clog.NewTracer("my-service",
    config.Tag("runner", "127.0.0.1:54321"),
    config.Tag("leader", "127.0.0.1:54312"),
)
```

## API

### `clog.NewTracer(name string, opts ...config.Option) (*Tracer, error)`

Creates a new Jaeger tracer. The `name` is used as the `JAEGER_SERVICE_NAME`. Additional `config.Option` values (e.g., `config.Tag(...)`) can be passed to customise the tracer further.

### `clog.NewTracerWithOptions(name string, opts ...Option) (*Tracer, error)`

Creates a new Jaeger tracer with programmatic option overrides. Options are applied before reading environment variables, so they take precedence over env vars.

```go
tracer, err := clog.NewTracerWithOptions("my-service",
    clog.With128Bit(true),
)
```

### `clog.With128Bit(enabled bool) Option`

Enables or disables 128-bit trace IDs. Enabled by default — pass `With128Bit(false)` to use legacy 64-bit trace IDs.

### `clog.CloseTracer(tracer *Tracer) error`

Closes a tracer, flushing any pending spans. Safe to call with `nil`.

```go
defer clog.CloseTracer(tracer)
```

### `clog.SetGlobalTracer(tracer *Tracer)`

Registers the tracer as the OpenTracing global tracer. Required before calling `StartSpanFromContext` without a parent span in context.

### `clog.GlobalTracer() *Tracer`

Returns the previously registered global tracer, or `nil` if none was set.

### `clog.StartSpanFromContext(ctx, name, opts...) (opentracing.Span, context.Context)`

Starts a new span. If `ctx` already contains a parent span, the new span becomes a child of it; otherwise a root span is created via the global tracer. Returns the span and an updated context that carries it.

### `clog.ExtractFromEnv(ctx) opentracing.SpanContext`

Extracts a parent span context from environment variables (`CLOG_TRACEPARENT`—W3C Trace Context, fallback `UBER_TRACE_ID`—Jaeger legacy). Used together with `InjectToEnv` or `TraceEnv` to propagate traces across process boundaries.

### `clog.InjectToEnv(span opentracing.Span) func()`

Serialises a span's context into environment variables (`CLOG_TRACEPARENT`, `CLOG_BAGGAGE_*`) so a child process can pick it up via `ExtractFromEnv`. Returns a **restore function** that reverts the environment to its previous state.

```go
// Parent process
span, ctx := clog.StartSpanFromContext(ctx, "parent")
restore := clog.InjectToEnv(span)
defer restore()

cmd := exec.Command("child-binary")
cmd.Env = os.Environ() // carries CLOG_TRACEPARENT
cmd.Run()
```

```go
// Child process (child-binary)
ctx := context.Background()
parentCtx := clog.ExtractFromEnv(ctx) // reads CLOG_TRACEPARENT
span, ctx := clog.StartSpanFromContext(ctx, "child")
// span is a child of the parent
defer span.Finish()
```

### `clog.TraceEnv(span opentracing.Span) []string`

Returns the span's trace context as `KEY=VALUE` strings suitable for appending to `os.Environ()` or `cmd.Env`. Unlike `InjectToEnv`, **does not modify the process environment** — ideal for constructing explicit env slices without global side effects.

```go
// Using TraceEnv with cmd.Env (no global side effects)
span, ctx := clog.StartSpanFromContext(ctx, "parent")
cmd := exec.Command("child-binary")
cmd.Env = append(os.Environ(), clog.TraceEnv(span)...)
cmd.Run()
```

```go
// Using TraceEnv with mvdan/sh (explicit env slice)
envVars := append(config.EnvVars, clog.TraceEnv(span)...)
r := interp.New(interp.Env(expand.ListEnviron(envVars...)))
```

## Important Notes

1. **128-bit trace IDs**: clog enables 128-bit trace IDs by default (`JAEGER_TRACEID_128BIT=true`). This aligns with the W3C Trace Context and OpenTelemetry standards. To use legacy 64-bit trace IDs, set `JAEGER_TRACEID_128BIT=false` before importing clog, or pass `clog.With128Bit(false)` to `NewTracerWithOptions`.

2. **UDP packet limits**: Jaeger's agent accepts spans over UDP. If you log many events in a single span, the combined payload may exceed the ~65 KB UDP packet limit. Use `JAEGER_REPORTER_MAX_QUEUE_SIZE` and `JAEGER_REPORTER_FLUSH_INTERVAL` to tune batching.
3. **Always close**: `defer tracer.Close()` in `main` and `defer span.Finish()` in each traced operation are required. Without them, buffered spans may never reach the backend. `clog.CloseTracer(tracer)` is a nil-safe convenience wrapper.

4. **Child span lifecycle**: A child span must be finished **before** its parent. The parent span only sends its logs and timing after `Finish()` is called.

5. **Lazy defaults**: Jaeger environment variable defaults (see table above) are only set on the first call to `NewTracer` or `NewTracerWithOptions`, and only when the corresponding env var is unset. Set any of these before calling `NewTracer` to override. No env vars are modified at import time.

6. **Cross-process propagation**: Use `InjectToEnv`/`ExtractFromEnv` or `TraceEnv` to propagate traces across process boundaries (e.g., `exec.Command`). `ExtractFromEnv` is also called internally by `StartSpanFromContext` as a fallback when no parent span exists in context, so child processes can create child spans without calling `ExtractFromEnv` explicitly. Prefer `TraceEnv` when you control the child's env slice (no global side effects); use `InjectToEnv` with `defer restore()` when modifying `os.Environ()` is more convenient.
