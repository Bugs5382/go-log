# go-log 🪵

> Tiny structured logging for Go services on [zerolog](https://github.com/rs/zerolog), with OpenTelemetry `trace_id`/`span_id` correlation baked in.

## 📦 Install

```bash
go get github.com/Bugs5382/go-log
```

## 🚀 Usage

```go
logger := log.New("my-service")
logger.Info().Msg("started")

// Inside a request/span, correlate logs with the active trace:
l := log.Ctx(ctx)
l.Info().Msg("handling request")
```

`New` writes JSON to stdout with a timestamp and a `service` field. `Ctx` derives
a logger that adds `trace_id` and `span_id` from the active OpenTelemetry span,
pairing with [go-otel](https://github.com/Bugs5382/go-otel). 🔗

### Neutral `Logger` (no zerolog dependency)

`New` and `Ctx` above return a concrete `zerolog.Logger`, so a wrapper package
that holds one is forced to import `github.com/rs/zerolog` just to type the
variable. `NewLogger` and `LoggerFromContext` return the neutral `Logger`
interface instead: no zerolog type appears in any of its method signatures,
so a consumer can depend on it (or an interface shaped like it) without ever
importing zerolog. zerolog stays an internal implementation detail behind
this path -- `LOG_LEVEL`, `LOG_FORMAT`, and OpenTelemetry trace correlation
all behave exactly as they do for `New`/`Ctx`.

```go
var logger log.Logger = log.NewLogger("my-service")
logger.Info("started", log.F("port", 8080))

child := logger.With(log.F("request_id", reqID))
child.Warn("slow downstream call", log.F("elapsed_ms", 420))
child.Error(err, "request failed")

// Inside a request/span, correlate logs with the active trace:
l := log.LoggerFromContext(ctx)
l.Info("handling request")
```

`Logger` covers `Debug`/`Info`/`Warn`/`Error`/`Fatal` with structured
`Field`s (build one with `log.F(key, val)`), a `With(fields...) Logger` for
child loggers, and a `Ctx(ctx) Logger` method mirroring the package-level
`Ctx`. `New`/`Ctx` are unchanged and continue to work side by side with the
neutral path.

## 🛠 Develop

```bash
task build    # go build ./...
task test     # go test ./...
task lint     # gofmt check + golangci-lint + yamllint
task license  # inject MIT headers (golic)
```

## ⚖️ License

MIT © 2026 Shane
