# go-log 🪵

> 🧵 Tiny structured logging for Go services on [zerolog](https://github.com/rs/zerolog), with OpenTelemetry `trace_id`/`span_id` correlation baked in.

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

> ⚠️ `Ctx` is **deprecated**. Having no receiver, it derives from whichever logger
> `New` created last, so a process with two loggers cannot get the right
> `service` from both. Prefer `NewLogger` and the `Logger.Ctx` method below,
> which derives from the logger it is called on.

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
l := logger.Ctx(ctx)
l.Info("handling request")
```

`Logger` covers `Debug`/`Info`/`Warn`/`Error`/`Fatal` with structured
`Field`s (build one with `log.F(key, val)`), a `With(fields...) Logger` for
child loggers, and a `Ctx(ctx) Logger` method. `Logger.Ctx` derives from the
receiver, so the `service` name and any fields already added with `With` are
carried onto the correlated logger.

The package-level `Ctx` and `LoggerFromContext` are **deprecated** in favour of
`Logger.Ctx` -- both read the logger most recently built by `New`, which is
ambiguous once a process has more than one.

### Options (output, default level and format)

`NewLoggerWithOptions` builds the same neutral `Logger` as `NewLogger`, tuned
by options. With no options it behaves exactly like `NewLogger`.

```go
level := log.LevelInfo
if verbose {
	level = log.LevelTrace
}

logger := log.NewLoggerWithOptions("my-cli",
	log.WithOutput(os.Stderr),                // logs never mix with stdout
	log.WithDefaultFormat(log.FormatConsole), // plain text unless LOG_FORMAT says otherwise
	log.WithDefaultLevel(level),              // used only when LOG_LEVEL is unset or invalid
)
logger.Trace("resolved config", log.F("path", path))
```

- `WithOutput(w)` sends every line to `w` instead of stdout, in any format.
  With `LOG_FORMAT=both` the JSON half goes to `w` and the console half still
  goes to stderr.
- `WithDefaultLevel(level)` replaces info as the fallback level. 🎛 **The
  environment still wins:** a valid `LOG_LEVEL` always overrides it. `Level` is
  a string type using the `LOG_LEVEL` names (`log.LevelTrace` ...
  `log.LevelDisabled`), so `log.Level(flagValue)` works too.
- `WithDefaultFormat(format)` replaces JSON as the fallback format
  (`log.FormatJSON`, `log.FormatConsole`, `log.FormatBoth`). A valid
  `LOG_FORMAT` always overrides it.

### Trace and the no-op Logger

`NewLoggerWithOptions` and `Nop` return a `TraceLogger`: a `Logger` plus a
`Trace(msg, fields...)` method. Every logger this package hands out (children
from `With` and `Ctx` too) has `Trace`; for a value typed as `Logger`, call
`log.Trace(logger, msg, fields...)`. `Trace` lives on its own interface
because adding a method to `Logger` would break anyone who implements it.

`log.Nop()` writes nothing, whatever `LOG_LEVEL` says, which suits tests and
quiet modes. Its `Fatal` still exits the process, as callers expect.

## 🛠 Develop

```bash
task build    # go build ./...
task test     # go test ./...
task lint     # gofmt check + golangci-lint + yamllint
task license  # inject MIT headers (golic)
```

## ⚖️ License

MIT © 2026 Shane
