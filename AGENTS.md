# clog — Agent Guide

## Build

```sh
go build ./...
go test ./... -v -count=1
```

## Project Structure

| File | Purpose |
|------|---------|
| `tracer.go` | Package entry. `applyJaegerDefaults()` sets Jaeger env defaults. `Tracer`, `NewTracer`, `NewTracerWithOptions`, `SetGlobalTracer`, `GlobalTracer`, `CloseTracer`. |
| `span.go` | `StartSpanFromContext`, `SpanFromContext`, `LogKV`, `ExtractFromEnv`, `TraceEnv`, `TraceparentOf`, `InjectToEnv`. |
| `log.go` | Convenience functions: `Debug`, `Info`, `Warn`, `Error` — key-value args wrapping `LogAttrs`. |
| `slog.go` | slog integration. `LogAttrs` — writes to both slog and span. `NewSpanHandler` — `slog.Handler` for span-only writes. |
| `*_test.go` | Tests in `package clog_test` (external test package). Uses real Jaeger in-memory tracer (no backend needed). |

## Design Decisions

- **External test package**: Tests live in `clog_test` to enforce testing against the public API only.
- **init() env defaults**: Importing clog sets Jaeger defaults (`JAEGER_SAMPLER_TYPE=const`, `JAEGER_TRACEID_128BIT=true`, etc.). Only sets when env var is empty — user-set values always win. Moved to lazy `applyJaegerDefaults()` at first `NewTracer` call.
- **slog is stdlib**: `log/slog` (Go 1.21+) is part of the standard library. No new dependencies introduced.
- **Two-layer API**: `Debug/Info/Warn/Error` accept `...any` key-value pairs for daily use. `LogAttrs` accepts typed `slog.Attr` for precision. Signatures differ — compiler disambiguates automatically.
- **NewSpanHandler writes to span only**: It does not write to any output sink. Combine with `slog.NewMultiHandler` for dual output.
- **LogAttrs writes to both**: Always writes to slog (subject to handler level filtering) AND the span (when present in context).
## Code Style

- Go 1.26+, standard formatting (`gofmt`)
- Comments on every exported symbol (godoc style)
- Test sub-tests for related scenarios (`t.Run`)
- No external test dependencies beyond `jaeger-client-go` and `opentracing-go`
