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

`clog` reads standard Jaeger environment variables via `config.FromEnv()`. Values set before `NewTracer` is called take priority over the built-in defaults.

| Variable | Default | Description |
|----------|---------|-------------|
| `JAEGER_SERVICE_NAME` | (set via `NewTracer(name)`) | Service name shown in traces |
| `JAEGER_SAMPLER_TYPE` | `const` | Sampler type (`const`, `probabilistic`, `ratelimiting`, `remote`) |
| `JAEGER_SAMPLER_PARAM` | `1` | Sampler parameter (e.g. `1` = sample all) |
| `JAEGER_REPORTER_MAX_QUEUE_SIZE` | `64` | Max spans in the send queue |
| `JAEGER_REPORTER_FLUSH_INTERVAL` | `10s` | How often to flush spans to the agent |
| `JAEGER_AGENT_HOST` | `localhost` | Jaeger agent UDP host |
| `JAEGER_AGENT_PORT` | `6831` | Jaeger agent UDP port |
| `JAEGER_REPORTER_LOG_SPANS` | (not set) | Log reporter activity when set to `true` |
| `JAEGER_DISABLED` | (not set) | Disable tracing when set to `true` |
| `JAEGER_TAGS` | (not set) | Comma-separated `key=value` tags applied to all spans |

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

### `clog.SetGlobalTracer(tracer *Tracer)`

Registers the tracer as the OpenTracing global tracer. Required before calling `StartSpanFromContext` without a parent span in context.

### `clog.GlobalTracer() *Tracer`

Returns the previously registered global tracer, or `nil` if none was set.

### `clog.StartSpanFromContext(ctx, name, opts...) (opentracing.Span, context.Context)`

Starts a new span. If `ctx` already contains a parent span, the new span becomes a child of it; otherwise a root span is created via the global tracer. Returns the span and an updated context that carries it.

## Important Notes

1. **UDP packet limits**: Jaeger's agent accepts spans over UDP. If you log many events in a single span, the combined payload may exceed the ~65 KB UDP packet limit. Use `JAEGER_REPORTER_MAX_QUEUE_SIZE` and `JAEGER_REPORTER_FLUSH_INTERVAL` to tune batching.

2. **Always close**:  `defer tracer.Close()` in `main` and `defer span.Finish()` in each traced operation are required. Without them, buffered spans may never reach the backend.

3. **Child span lifecycle**: A child span must be finished **before** its parent. The parent span only sends its logs and timing after `Finish()` is called.

## Example: Full flow

```go
package main

import (
    "context"
    "log"

    "github.com/nanjj/clog"
)

func main() {
    tracer, err := clog.NewTracer("order-service")
    if err != nil {
        log.Fatal(err)
    }
    clog.SetGlobalTracer(tracer)
    defer tracer.Close()

    ctx := context.Background()
    processOrder(ctx, "ord_001")
}

func processOrder(ctx context.Context, orderID string) {
    span, ctx := clog.StartSpanFromContext(ctx, "processOrder")
    defer span.Finish()

    span.LogKV("order_id", orderID)

    chargeCustomer(ctx, orderID)
}

func chargeCustomer(ctx context.Context, orderID string) {
    span, ctx := clog.StartSpanFromContext(ctx, "chargeCustomer")
    defer span.Finish()

    span.LogKV("event", "payment_initiated", "order_id", orderID)
    // ... payment logic ...
}
```
