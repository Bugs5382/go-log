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

## 🛠 Develop

```bash
task build    # go build ./...
task test     # go test ./...
task lint     # gofmt check + golangci-lint + yamllint
task license  # inject MIT headers (golic)
```

## ⚖️ License

MIT © 2026 Shane
