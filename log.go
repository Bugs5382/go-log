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
	"os"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// serviceName holds the name attached by New so Ctx can rebuild an equivalent
// base logger. It is set on the last call to New.
var serviceName string

// New returns a zerolog.Logger that writes JSON to stdout with a timestamp
// and the given service name attached to every line.
func New(service string) zerolog.Logger {
	serviceName = service
	return zerolog.New(os.Stdout).With().Timestamp().Str("service", service).Logger()
}

// Ctx returns a zerolog.Logger derived from the base logger with the active
// span's trace_id and span_id attached when ctx carries a valid span. This lets
// every log line be correlated with its trace in the tracing backend. When no
// valid span is present, it returns the plain base logger.
func Ctx(ctx context.Context) zerolog.Logger {
	base := zerolog.New(os.Stdout).With().Timestamp().Str("service", serviceName).Logger()

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return base
	}

	return base.With().
		Str("trace_id", sc.TraceID().String()).
		Str("span_id", sc.SpanID().String()).
		Logger()
}
