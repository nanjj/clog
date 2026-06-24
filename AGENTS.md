# clog — Agent Guide

## Build

```sh
go build ./...
go test ./... -v -count=1
```

## Project Structure

| File | Purpose |
|------|---------|
| `tracer.go` | Package entry. `applyJaegerDefaults()` sets Jaeger env defaults. `Tracer` struct, `NewTracer`, `NewTracerWithOptions`, `SetGlobalTracer`, `GlobalTracer`, `CloseTracer`. |
| `span.go` | `StartSpanFromContext` — creates child/root spans from context. `SpanFromContext`, `LogKV`, `ExtractFromEnv`, `TraceEnv`, `TraceparentOf`, `InjectToEnv`. |
| `slog.go` | slog integration. `LogAttrs` — convenience for writing to both slog and span. `NewSpanHandler` — a `slog.Handler` that writes records to the active span. |
| `*_test.go` | Tests in `package clog_test` (external test package). Uses real Jaeger in-memory tracer (no backend needed). |

## Design Decisions

- **External test package**: Tests live in `clog_test` to enforce testing against the public API only.
- **init() env defaults**: Importing clog sets Jaeger defaults (`JAEGER_SAMPLER_TYPE=const`, `JAEGER_TRACEID_128BIT=true`, etc.). Only sets when env var is empty — user-set values always win. Moved to lazy `applyJaegerDefaults()` at first `NewTracer` call.
- **No Jaeger backend needed**: Tests use jaeger-client-go's in-memory transport — no agent required.
- **No panic**: `StartSpanFromContext` safely handles non-clog global tracers (returns noop spans).
- **slog is stdlib**: `log/slog` (Go 1.21+) is part of the standard library. No new dependencies introduced.
- **NewSpanHandler writes to span only**: It does not write to any output sink. Combine with `slog.NewMultiHandler` for dual output.
- **LogAttrs writes to both**: Always writes to slog (subject to handler level filtering) AND the span (when present in context).

## Code Style

- Go 1.26+, standard formatting (`gofmt`)
- Comments on every exported symbol (godoc style)
- Test sub-tests for related scenarios (`t.Run`)
- No external test dependencies beyond `jaeger-client-go` and `opentracing-go`
