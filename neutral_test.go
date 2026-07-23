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
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestNewLoggerAttachesServiceAndFields(t *testing.T) {
	out := captureStdout(t, func() {
		l := NewLogger("billing")
		l.Info("hello", F("order_id", "o-1"), F("count", 3))
	})
	if !strings.Contains(out, `"service":"billing"`) {
		t.Fatalf("expected service field, got %q", out)
	}
	if !strings.Contains(out, `"message":"hello"`) {
		t.Fatalf("expected message, got %q", out)
	}
	if !strings.Contains(out, `"order_id":"o-1"`) {
		t.Fatalf("expected order_id field, got %q", out)
	}
	if !strings.Contains(out, `"count":3`) {
		t.Fatalf("expected count field, got %q", out)
	}
}

func TestLoggerLevels(t *testing.T) {
	out := captureStdout(t, func() {
		l := NewLogger("billing")
		l.Debug("dbg")
		l.Info("nfo")
		l.Warn("wrn")
		l.Error(errors.New("boom"), "err")
	})
	if strings.Contains(out, "dbg") {
		t.Fatalf("debug should be filtered at the default info level, got %q", out)
	}
	for _, want := range []string{"nfo", "wrn", "err"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got %q", want, out)
		}
	}
	if !strings.Contains(out, `"error":"boom"`) {
		t.Fatalf("expected error field on Error, got %q", out)
	}
}

func TestLoggerWithChainsFields(t *testing.T) {
	out := captureStdout(t, func() {
		l := NewLogger("billing").With(F("request_id", "r-1"))
		l.Info("handled")
	})
	if !strings.Contains(out, `"request_id":"r-1"`) {
		t.Fatalf("expected request_id field from With, got %q", out)
	}
	if !strings.Contains(out, `"service":"billing"`) {
		t.Fatalf("expected service field preserved through With, got %q", out)
	}
}

func TestLoggerFromContextAttachesTraceAndSpanIDs(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	out := captureStdout(t, func() {
		NewLogger("billing")
		l := LoggerFromContext(ctx)
		l.Info("with-trace")
	})
	if !strings.Contains(out, `"trace_id":"`+traceID.String()+`"`) {
		t.Fatalf("expected trace_id, got %q", out)
	}
	if !strings.Contains(out, `"span_id":"`+spanID.String()+`"`) {
		t.Fatalf("expected span_id, got %q", out)
	}
}

func TestLoggerFromContextWithoutSpanOmitsTraceID(t *testing.T) {
	out := captureStdout(t, func() {
		NewLogger("billing")
		l := LoggerFromContext(context.Background())
		l.Info("no-trace")
	})
	if strings.Contains(out, "trace_id") {
		t.Fatalf("did not expect trace_id without a valid span, got %q", out)
	}
}

func TestLoggerCtxMethodMatchesPackageLevelCtx(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	out := captureStdout(t, func() {
		base := NewLogger("billing")
		l := base.Ctx(ctx)
		l.Info("via-method")
	})
	if !strings.Contains(out, `"trace_id":"`+traceID.String()+`"`) {
		t.Fatalf("expected trace_id via Logger.Ctx, got %q", out)
	}
}

func TestLoggerHonorsLevelFromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warn")
	out := captureStdout(t, func() {
		l := NewLogger("billing")
		l.Info("hidden")
		l.Warn("shown")
	})
	if strings.Contains(out, "hidden") {
		t.Fatalf("info line should be filtered at warn level, got %q", out)
	}
	if !strings.Contains(out, "shown") {
		t.Fatalf("warn line should be emitted, got %q", out)
	}
}

func TestLoggerHonorsConsoleFormat(t *testing.T) {
	t.Setenv("LOG_FORMAT", "console")
	out := captureStdout(t, func() {
		l := NewLogger("billing")
		l.Info("pretty")
	})
	if strings.Contains(out, `{"level"`) {
		t.Fatalf("console format should not emit JSON, got %q", out)
	}
	if !strings.Contains(out, "pretty") {
		t.Fatalf("expected the message text in console output, got %q", out)
	}
}

// TestNeutralFatalExits re-executes this test binary as a subprocess to
// exercise Fatal, since a real call terminates the process. The subprocess
// path is selected by GO_LOG_TEST_FATAL; the parent asserts a non-zero exit.
func TestNeutralFatalExits(t *testing.T) {
	if os.Getenv("GO_LOG_TEST_FATAL") == "1" {
		NewLogger("billing").Fatal(errors.New("boom"), "dying")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestNeutralFatalExits")
	cmd.Env = append(os.Environ(), "GO_LOG_TEST_FATAL=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.Success() {
		t.Fatalf("expected Fatal to exit non-zero, err=%v output=%q", err, out)
	}
	if !strings.Contains(string(out), `"error":"boom"`) {
		t.Fatalf("expected error field logged before exit, got %q", out)
	}
}
