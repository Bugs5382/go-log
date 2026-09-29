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
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// clearEnv makes a test independent of LOG_LEVEL/LOG_FORMAT in the caller's
// shell; an empty value is treated the same as unset.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")
}

func TestWithOutputWritesToWriterNotStdout(t *testing.T) {
	clearEnv(t)
	var buf bytes.Buffer
	stdout := captureStdout(t, func() {
		l := NewLoggerWithOptions("golic", WithOutput(&buf))
		l.Info("hello", F("file", "a.go"))
	})
	if stdout != "" {
		t.Fatalf("expected nothing on stdout, got %q", stdout)
	}
	got := buf.String()
	for _, want := range []string{`"service":"golic"`, `"message":"hello"`, `"file":"a.go"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %s in writer output, got %q", want, got)
		}
	}
}

func TestWithOutputConsoleFormatGoesToWriter(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_FORMAT", "console")
	var buf bytes.Buffer
	stdout := captureStdout(t, func() {
		NewLoggerWithOptions("golic", WithOutput(&buf)).Info("pretty")
	})
	if stdout != "" {
		t.Fatalf("expected nothing on stdout, got %q", stdout)
	}
	got := buf.String()
	if strings.Contains(got, `{"level"`) {
		t.Fatalf("console format should not emit JSON, got %q", got)
	}
	if !strings.Contains(got, "pretty") {
		t.Fatalf("expected the message text in console output, got %q", got)
	}
}

func TestWithOutputBothFormatKeepsJSONOffStdout(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_FORMAT", "both")
	var buf bytes.Buffer
	stdout := captureStdout(t, func() {
		NewLoggerWithOptions("golic", WithOutput(&buf)).Info("twice")
	})
	if stdout != "" {
		t.Fatalf("expected nothing on stdout, got %q", stdout)
	}
	if !strings.Contains(buf.String(), `"message":"twice"`) {
		t.Fatalf("expected the JSON half on the writer, got %q", buf.String())
	}
}

func TestWithDefaultLevelAppliesWhenLogLevelUnset(t *testing.T) {
	clearEnv(t)
	var buf bytes.Buffer
	l := NewLoggerWithOptions("golic", WithOutput(&buf), WithDefaultLevel(LevelTrace))
	l.Trace("step", F("n", 1))
	l.Debug("flow")
	got := buf.String()
	if !strings.Contains(got, `"level":"trace"`) || !strings.Contains(got, `"message":"step"`) {
		t.Fatalf("expected the trace line at the default trace level, got %q", got)
	}
	if !strings.Contains(got, `"message":"flow"`) {
		t.Fatalf("expected the debug line at the default trace level, got %q", got)
	}
}

func TestWithDefaultLevelAppliesWhenLogLevelInvalid(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_LEVEL", "loud")
	var buf bytes.Buffer
	NewLoggerWithOptions("golic", WithOutput(&buf), WithDefaultLevel(LevelDebug)).Debug("flow")
	if !strings.Contains(buf.String(), `"message":"flow"`) {
		t.Fatalf("expected the fallback level to apply for an invalid LOG_LEVEL, got %q", buf.String())
	}
}

func TestLogLevelWinsOverDefaultLevel(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_LEVEL", "warn")
	var buf bytes.Buffer
	l := NewLoggerWithOptions("golic", WithOutput(&buf), WithDefaultLevel(LevelTrace))
	l.Trace("hidden-trace")
	l.Info("hidden-info")
	l.Warn("shown")
	got := buf.String()
	if strings.Contains(got, "hidden") {
		t.Fatalf("LOG_LEVEL=warn should win over the default level, got %q", got)
	}
	if !strings.Contains(got, "shown") {
		t.Fatalf("expected the warn line, got %q", got)
	}
}

func TestNoDefaultLevelStaysInfo(t *testing.T) {
	clearEnv(t)
	var buf bytes.Buffer
	l := NewLoggerWithOptions("golic", WithOutput(&buf))
	l.Trace("hidden-trace")
	l.Debug("hidden-debug")
	l.Info("shown")
	got := buf.String()
	if strings.Contains(got, "hidden") {
		t.Fatalf("expected the info default without WithDefaultLevel, got %q", got)
	}
	if !strings.Contains(got, "shown") {
		t.Fatalf("expected the info line, got %q", got)
	}
}

func TestUnknownDefaultLevelFallsBackToInfo(t *testing.T) {
	clearEnv(t)
	var buf bytes.Buffer
	l := NewLoggerWithOptions("golic", WithOutput(&buf), WithDefaultLevel(Level("chatty")))
	l.Debug("hidden")
	l.Info("shown")
	got := buf.String()
	if strings.Contains(got, "hidden") || !strings.Contains(got, "shown") {
		t.Fatalf("expected an unknown default level to mean info, got %q", got)
	}
}

func TestWithDefaultFormatAppliesWhenLogFormatUnset(t *testing.T) {
	clearEnv(t)
	var buf bytes.Buffer
	NewLoggerWithOptions("golic", WithOutput(&buf), WithDefaultFormat(FormatConsole)).Info("pretty")
	got := buf.String()
	if strings.Contains(got, `{"level"`) || !strings.Contains(got, "pretty") {
		t.Fatalf("expected console output from the default format, got %q", got)
	}
}

func TestLogFormatWinsOverDefaultFormat(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_FORMAT", "json")
	var buf bytes.Buffer
	NewLoggerWithOptions("golic", WithOutput(&buf), WithDefaultFormat(FormatConsole)).Info("structured")
	if !strings.Contains(buf.String(), `"message":"structured"`) {
		t.Fatalf("LOG_FORMAT=json should win over the default format, got %q", buf.String())
	}
}

func TestNewLoggerWithOptionsDefaultsMatchNewLogger(t *testing.T) {
	clearEnv(t)
	out := captureStdout(t, func() {
		NewLoggerWithOptions("billing").Info("hello")
	})
	if !strings.Contains(out, `"service":"billing"`) || !strings.Contains(out, `"message":"hello"`) {
		t.Fatalf("expected JSON on stdout with no options, got %q", out)
	}
}

func TestLoggerFromContextDerivesFromNewLoggerWithOptions(t *testing.T) {
	clearEnv(t)
	var buf bytes.Buffer
	NewLoggerWithOptions("golic", WithOutput(&buf))
	stdout := captureStdout(t, func() {
		LoggerFromContext(context.Background()).Info("via-package")
	})
	if stdout != "" {
		t.Fatalf("expected nothing on stdout, got %q", stdout)
	}
	if !strings.Contains(buf.String(), `"service":"golic"`) {
		t.Fatalf("expected LoggerFromContext to use the most recent logger, got %q", buf.String())
	}
}

func TestTraceIsFilteredAtDefaultLevel(t *testing.T) {
	clearEnv(t)
	out := captureStdout(t, func() {
		l := NewLogger("billing")
		Trace(l, "hidden")
		l.Info("shown")
	})
	if strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Fatalf("expected trace filtered at the info default, got %q", out)
	}
}

func TestTraceHelperReachesChildLoggers(t *testing.T) {
	clearEnv(t)
	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))

	var buf bytes.Buffer
	root := NewLoggerWithOptions("golic", WithOutput(&buf), WithDefaultLevel(LevelTrace))
	child := root.With(F("file", "a.go")).Ctx(ctx)
	if _, ok := child.(TraceLogger); !ok {
		t.Fatalf("expected With/Ctx children to implement TraceLogger")
	}
	Trace(child, "deep")
	got := buf.String()
	for _, want := range []string{`"level":"trace"`, `"message":"deep"`, `"file":"a.go"`, `"trace_id":"` + traceID.String() + `"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %s, got %q", want, got)
		}
	}
}

// debugOnly is a Logger from outside this package that has no Trace method.
type debugOnly struct{ calls int }

func (d *debugOnly) Debug(string, ...Field)        { d.calls++ }
func (d *debugOnly) Info(string, ...Field)         { d.calls++ }
func (d *debugOnly) Warn(string, ...Field)         { d.calls++ }
func (d *debugOnly) Error(error, string, ...Field) { d.calls++ }
func (d *debugOnly) Fatal(error, string, ...Field) { d.calls++ }
func (d *debugOnly) With(...Field) Logger          { return d }
func (d *debugOnly) Ctx(context.Context) Logger    { return d }

func TestTraceHelperSkipsLoggersWithoutTrace(t *testing.T) {
	d := &debugOnly{}
	Trace(d, "dropped")
	if d.calls != 0 {
		t.Fatalf("expected Trace to drop the line for a Logger without Trace, got %d calls", d.calls)
	}
}

func TestNopWritesNothing(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_LEVEL", "trace")
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	stdout := captureStdout(t, func() {
		l := Nop()
		l.Trace("t")
		l.Debug("d")
		l.Info("i")
		l.Warn("w")
		l.Error(errors.New("boom"), "e")
		child := l.With(F("k", "v")).Ctx(context.Background())
		child.Info("child")
		Trace(child, "child-trace")
	})
	_ = w.Close()
	os.Stderr = orig
	var stderr bytes.Buffer
	_, _ = stderr.ReadFrom(r)
	if stdout != "" || stderr.Len() != 0 {
		t.Fatalf("expected Nop to write nothing, stdout=%q stderr=%q", stdout, stderr.String())
	}
}

// TestNopFatalStillExits re-executes this test binary: Fatal keeps its
// contract of ending the process even when nothing is written.
func TestNopFatalStillExits(t *testing.T) {
	if os.Getenv("GO_LOG_TEST_NOP_FATAL") == "1" {
		Nop().Fatal(errors.New("boom"), "dying")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestNopFatalStillExits")
	cmd.Env = append(os.Environ(), "GO_LOG_TEST_NOP_FATAL=1")
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.Success() {
		t.Fatalf("expected Nop Fatal to exit non-zero, err=%v output=%q", err, out)
	}
	if strings.Contains(string(out), "boom") {
		t.Fatalf("expected Nop Fatal to write nothing, got %q", out)
	}
}
