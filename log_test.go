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
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// captureStdout redirects os.Stdout for the duration of f and returns what was
// written. Log lines are tiny, so the pipe buffer never blocks.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	f()
	_ = w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestNewAttachesService(t *testing.T) {
	out := captureStdout(t, func() {
		l := New("billing")
		l.Info().Msg("hello")
	})
	if !strings.Contains(out, `"service":"billing"`) {
		t.Fatalf("expected service field, got %q", out)
	}
	if !strings.Contains(out, `"message":"hello"`) {
		t.Fatalf("expected message, got %q", out)
	}
}

func TestCtxAttachesTraceAndSpanIDs(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	out := captureStdout(t, func() {
		New("billing")
		l := Ctx(ctx)
		l.Info().Msg("with-trace")
	})
	if !strings.Contains(out, `"trace_id":"`+traceID.String()+`"`) {
		t.Fatalf("expected trace_id, got %q", out)
	}
	if !strings.Contains(out, `"span_id":"`+spanID.String()+`"`) {
		t.Fatalf("expected span_id, got %q", out)
	}
}

func TestCtxWithoutSpanOmitsTraceID(t *testing.T) {
	out := captureStdout(t, func() {
		New("billing")
		l := Ctx(context.Background())
		l.Info().Msg("no-trace")
	})
	if strings.Contains(out, "trace_id") {
		t.Fatalf("did not expect trace_id without a valid span, got %q", out)
	}
}
