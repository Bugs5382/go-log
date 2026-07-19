package log

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// serviceName holds the name attached by New so Ctx can rebuild an equivalent
// base logger. It is set on the last call to New.
var serviceName string

// levelFromEnv resolves the minimum log level from the LOG_LEVEL environment
// variable (case-insensitive: trace, debug, info, warn, error, fatal, panic,
// disabled). An unset or unrecognized value falls back to info -- the safe
// default so a service is never accidentally silent or debug-noisy in
// production. Deployments raise verbosity by setting LOG_LEVEL (e.g. trace in
// dev) without any code change.
func levelFromEnv() zerolog.Level {
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		if lvl, err := zerolog.ParseLevel(strings.ToLower(v)); err == nil {
			return lvl
		}
	}
	return zerolog.InfoLevel
}

// writer resolves the output(s) from LOG_FORMAT. JSON is the default because
// OpenTelemetry/Loki trace correlation depends on structured fields
// (trace_id/span_id from Ctx) being machine-parseable:
//
//   - unset / "json": structured JSON on stdout (qa/prod; OTel-safe).
//   - "console" / "pretty": a human-readable, colorized ConsoleWriter on
//     stdout, for local dev where you read logs by eye (not aggregated).
//   - "both": JSON on stdout (kept parseable for OTel/Loki) AND a pretty
//     rendering on stderr for a human tailing the console -- both at once,
//     so trace correlation is never lost to get a readable console.
func writer() io.Writer {
	switch strings.ToLower(os.Getenv("LOG_FORMAT")) {
	case "console", "pretty":
		return zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	case "both":
		return zerolog.MultiLevelWriter(os.Stdout, zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	default:
		return os.Stdout
	}
}

// New returns a zerolog.Logger with a timestamp and the given service name
// attached to every line, filtered to the level from LOG_LEVEL (default info)
// and rendered per LOG_FORMAT (JSON by default, console/pretty for dev).
// Consumers no longer need to parse LOG_LEVEL or wire a writer themselves --
// calling New is enough.
func New(service string) zerolog.Logger {
	serviceName = service
	return zerolog.New(writer()).Level(levelFromEnv()).With().Timestamp().Str("service", service).Logger()
}

// Ctx returns a zerolog.Logger derived from the base logger with the active
// span's trace_id and span_id attached when ctx carries a valid span. This lets
// every log line be correlated with its trace in the tracing backend. When no
// valid span is present, it returns the plain base logger.
func Ctx(ctx context.Context) zerolog.Logger {
	base := zerolog.New(writer()).Level(levelFromEnv()).With().Timestamp().Str("service", serviceName).Logger()

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return base
	}

	return base.With().
		Str("trace_id", sc.TraceID().String()).
		Str("span_id", sc.SpanID().String()).
		Logger()
}
