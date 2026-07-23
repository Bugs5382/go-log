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

	"github.com/rs/zerolog"
)

// Field is a structured key/value pair attached to a Logger call. Build one
// with F, or construct it directly since both fields are exported.
type Field struct {
	Key string
	Val any
}

// F builds a Field for a neutral Logger call, e.g. log.F("order_id", id).
func F(key string, val any) Field {
	return Field{Key: key, Val: val}
}

// Logger is the neutral logging surface: no zerolog type appears in any
// method signature here, so a consumer can depend on this interface alone
// and never import zerolog directly. zerolog remains an internal
// implementation detail behind it. New/Ctx (which do return a concrete
// zerolog.Logger) are unchanged and continue to work side by side with this
// interface.
type Logger interface {
	// Debug logs msg at debug level with the given structured fields.
	Debug(msg string, fields ...Field)
	// Info logs msg at info level with the given structured fields.
	Info(msg string, fields ...Field)
	// Warn logs msg at warn level with the given structured fields.
	Warn(msg string, fields ...Field)
	// Error logs msg at error level, attaching err, with the given
	// structured fields.
	Error(err error, msg string, fields ...Field)
	// Fatal logs msg at fatal level, attaching err, with the given
	// structured fields, then terminates the process with a non-zero exit
	// code.
	Fatal(err error, msg string, fields ...Field)
	// With returns a child Logger that carries fields on every subsequent
	// line, in addition to whatever the receiver already carries.
	With(fields ...Field) Logger
	// Ctx returns a Logger correlated with the trace/span carried by ctx,
	// the neutral equivalent of the package-level Ctx function. Like Ctx, it
	// rebuilds from the service's base logger rather than the receiver, so
	// fields added via With are not carried over -- call With after Ctx if
	// both are needed.
	Ctx(ctx context.Context) Logger
}

// neutralLogger adapts a zerolog.Logger to Logger. zerolog is confined to
// this file; it never appears in the Logger interface above.
type neutralLogger struct {
	l zerolog.Logger
}

// NewLogger returns a neutral Logger for a service, honoring LOG_LEVEL and
// LOG_FORMAT exactly like New. Use this when the caller must not import
// zerolog directly; use New when a concrete zerolog.Logger is wanted instead.
func NewLogger(service string) Logger {
	return neutralLogger{l: New(service)}
}

// LoggerFromContext returns a neutral Logger correlated with the trace/span
// carried by ctx -- the neutral equivalent of Ctx.
func LoggerFromContext(ctx context.Context) Logger {
	return neutralLogger{l: Ctx(ctx)}
}

// withFields attaches each Field to e as an interface value and returns e for
// chaining into Msg.
func withFields(e *zerolog.Event, fields []Field) *zerolog.Event {
	for _, f := range fields {
		e = e.Interface(f.Key, f.Val)
	}
	return e
}

func (n neutralLogger) Debug(msg string, fields ...Field) {
	withFields(n.l.Debug(), fields).Msg(msg)
}

func (n neutralLogger) Info(msg string, fields ...Field) {
	withFields(n.l.Info(), fields).Msg(msg)
}

func (n neutralLogger) Warn(msg string, fields ...Field) {
	withFields(n.l.Warn(), fields).Msg(msg)
}

func (n neutralLogger) Error(err error, msg string, fields ...Field) {
	withFields(n.l.Error().Err(err), fields).Msg(msg)
}

func (n neutralLogger) Fatal(err error, msg string, fields ...Field) {
	withFields(n.l.Fatal().Err(err), fields).Msg(msg)
}

func (n neutralLogger) With(fields ...Field) Logger {
	c := n.l.With()
	for _, f := range fields {
		c = c.Interface(f.Key, f.Val)
	}
	return neutralLogger{l: c.Logger()}
}

func (n neutralLogger) Ctx(ctx context.Context) Logger {
	return neutralLogger{l: Ctx(ctx)}
}
