# clog — Agent Guide

## Build

```sh
go build ./...
go test ./... -v -count=1
```

## Project Structure

| File | Purpose |
|------|---------|
| `tracer.go` | Package entry. `init()` sets Jaeger env defaults. `Tracer` struct, `NewTracer`, `NewTracerWithOptions`, `SetGlobalTracer`, `GlobalTracer`, `CloseTracer`. |
| `span.go` | `StartSpanFromContext` — creates child/root spans from context. |
| `*_test.go` | Tests in `package clog_test` (external test package). Uses real Jaeger in-memory tracer (no backend needed). |

## Design Decisions

- **External test package**: Tests live in `clog_test` to enforce testing against the public API only.
- **init() env defaults**: Importing clog sets Jaeger defaults (`JAEGER_SAMPLER_TYPE=const`, `JAEGER_TRACEID_128BIT=true`, etc.). Only sets when env var is empty — user-set values always win.
- **No Jaeger backend needed**: Tests use jaeger-client-go's in-memory transport — no agent required.
- **No panic**: `StartSpanFromContext` safely handles non-clog global tracers (returns noop spans).

## Code Style

- Go 1.26+, standard formatting (`gofmt`)
- Comments on every exported symbol (godoc style)
- Test sub-tests for related scenarios (`t.Run`)
- No external test dependencies beyond `jaeger-client-go` and `opentracing-go`
