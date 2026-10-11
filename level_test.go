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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// syncBuffer is a bytes.Buffer safe to write from the watcher goroutine while
// the test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// isolateLevel gives a test a clean live level and puts it back afterwards,
// so a SetLevel in one test never leaks into the next.
func isolateLevel(t *testing.T) {
	t.Helper()
	clearEnv(t)
	reset := func() {
		levelMu.Lock()
		overridden = false
		startLevel = zerolog.InfoLevel
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		levelMu.Unlock()
	}
	reset()
	t.Cleanup(reset)
	orig := levelPollInterval
	levelPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { levelPollInterval = orig })
}

// watch starts a watcher that is stopped, and waited for, when the test ends.
func watch(t *testing.T, path string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := watchLevelFile(ctx, path)
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func writeLevelFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write level file: %v", err)
	}
}

func TestSetLevelChangesExistingLoggers(t *testing.T) {
	isolateLevel(t)
	var buf syncBuffer
	l := NewLoggerWithOptions("svc", WithOutput(&buf), WithDefaultLevel(LevelError))
	child := l.With(F("k", "v"))
	l.Debug("hidden-before")

	if err := SetLevel(LevelDebug); err != nil {
		t.Fatalf("SetLevel: %v", err)
	}
	l.Debug("root-after")
	child.Debug("child-after")
	got := buf.String()
	if strings.Contains(got, "hidden-before") {
		t.Fatalf("expected debug filtered at the error start level, got %q", got)
	}
	for _, want := range []string{"root-after", "child-after"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %s once the live level is debug, got %q", want, got)
		}
	}
	if CurrentLevel() != LevelDebug {
		t.Fatalf("expected CurrentLevel debug, got %q", CurrentLevel())
	}
}

func TestSetLevelReachesZerologSurface(t *testing.T) {
	isolateLevel(t)
	t.Setenv("LOG_LEVEL", "error")
	out := captureStdout(t, func() {
		l := New("svc")
		l.Info().Msg("hidden")
		if err := SetLevel(LevelInfo); err != nil {
			t.Errorf("SetLevel: %v", err)
		}
		l.Info().Msg("shown")
	})
	if strings.Contains(out, "hidden") || !strings.Contains(out, "shown") {
		t.Fatalf("expected the live level to apply to New loggers, got %q", out)
	}
}

func TestSetLevelRejectsUnknownLevel(t *testing.T) {
	isolateLevel(t)
	t.Setenv("LOG_LEVEL", "warn")
	NewLoggerWithOptions("svc", WithOutput(&syncBuffer{}))
	for _, bad := range []Level{"", "loud", "WARNING", "-1"} {
		if err := SetLevel(bad); err == nil {
			t.Fatalf("expected an error for level %q", bad)
		}
	}
	if CurrentLevel() != LevelWarn {
		t.Fatalf("expected the level to stay warn, got %q", CurrentLevel())
	}
}

func TestLogLevelIsTheStartLevel(t *testing.T) {
	isolateLevel(t)
	t.Setenv("LOG_LEVEL", "warn")
	NewLoggerWithOptions("svc", WithOutput(&syncBuffer{}), WithDefaultLevel(LevelTrace))
	if CurrentLevel() != LevelWarn {
		t.Fatalf("expected LOG_LEVEL as the start level, got %q", CurrentLevel())
	}
}

func TestNewLoggerKeepsALiveLevel(t *testing.T) {
	isolateLevel(t)
	t.Setenv("LOG_LEVEL", "error")
	if err := SetLevel(LevelDebug); err != nil {
		t.Fatalf("SetLevel: %v", err)
	}
	var buf syncBuffer
	NewLoggerWithOptions("late", WithOutput(&buf)).Debug("shown")
	if CurrentLevel() != LevelDebug || !strings.Contains(buf.String(), "shown") {
		t.Fatalf("expected a logger built later to keep the live level, level=%q out=%q", CurrentLevel(), buf.String())
	}
}

func TestWatchLevelFileAppliesAtStartAndOnChange(t *testing.T) {
	isolateLevel(t)
	t.Setenv("LOG_LEVEL", "error")
	NewLoggerWithOptions("svc", WithOutput(&syncBuffer{}))
	path := filepath.Join(t.TempDir(), "level")
	writeLevelFile(t, path, "Debug\n")

	watch(t, path)
	if CurrentLevel() != LevelDebug {
		t.Fatalf("expected the file level before WatchLevelFile returns, got %q", CurrentLevel())
	}
	writeLevelFile(t, path, "trace")
	eventually(t, "trace", func() bool { return CurrentLevel() == LevelTrace })
	writeLevelFile(t, path, "error")
	eventually(t, "error", func() bool { return CurrentLevel() == LevelError })
}

func TestWatchLevelFileFollowsASymlinkSwap(t *testing.T) {
	isolateLevel(t)
	dir := t.TempDir()
	for name, lvl := range map[string]string{"a": "debug", "b": "warn"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
		writeLevelFile(t, filepath.Join(dir, name, "level"), lvl)
	}
	link := filepath.Join(dir, "current")
	if err := os.Symlink(filepath.Join(dir, "a"), link); err != nil {
		t.Fatal(err)
	}
	watch(t, filepath.Join(link, "level"))
	if CurrentLevel() != LevelDebug {
		t.Fatalf("expected debug, got %q", CurrentLevel())
	}
	tmp := filepath.Join(dir, "next")
	if err := os.Symlink(filepath.Join(dir, "b"), tmp); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, link); err != nil {
		t.Fatal(err)
	}
	eventually(t, "warn after the swap", func() bool { return CurrentLevel() == LevelWarn })
}

func TestWatchLevelFileUnknownWordKeepsLevelAndWarns(t *testing.T) {
	isolateLevel(t)
	var buf syncBuffer
	NewLoggerWithOptions("svc", WithOutput(&buf), WithDefaultLevel(LevelError))
	path := filepath.Join(t.TempDir(), "level")
	writeLevelFile(t, path, "debug")
	watch(t, path)

	writeLevelFile(t, path, "loud")
	eventually(t, "the warning", func() bool { return strings.Contains(buf.String(), `"level":"warn"`) })
	got := buf.String()
	if !strings.Contains(got, "loud") {
		t.Fatalf("expected the warning to name the bad word, got %q", got)
	}
	if CurrentLevel() != LevelDebug {
		t.Fatalf("expected an unknown word to keep debug, got %q", CurrentLevel())
	}
}

func TestWatchLevelFileUnknownWordWarnsEvenAtErrorLevel(t *testing.T) {
	isolateLevel(t)
	var buf syncBuffer
	NewLoggerWithOptions("svc", WithOutput(&buf), WithDefaultLevel(LevelError))
	path := filepath.Join(t.TempDir(), "level")
	writeLevelFile(t, path, "chatty")
	watch(t, path)
	if !strings.Contains(buf.String(), "chatty") {
		t.Fatalf("expected the warning to be written while the level is error, got %q", buf.String())
	}
	if CurrentLevel() != LevelError {
		t.Fatalf("expected the start level to stay, got %q", CurrentLevel())
	}
}

func TestWatchLevelFileMissingFileMeansStartLevel(t *testing.T) {
	isolateLevel(t)
	t.Setenv("LOG_LEVEL", "warn")
	NewLoggerWithOptions("svc", WithOutput(&syncBuffer{}))
	dir := t.TempDir()
	path := filepath.Join(dir, "level")
	writeLevelFile(t, path, "debug")
	watch(t, path)
	if CurrentLevel() != LevelDebug {
		t.Fatalf("expected debug, got %q", CurrentLevel())
	}
	if err := os.Rename(path, filepath.Join(dir, "gone")); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the start level", func() bool { return CurrentLevel() == LevelWarn })
}

func TestWatchLevelFileEmptyFileMeansStartLevel(t *testing.T) {
	isolateLevel(t)
	t.Setenv("LOG_LEVEL", "info")
	NewLoggerWithOptions("svc", WithOutput(&syncBuffer{}))
	path := filepath.Join(t.TempDir(), "level")
	writeLevelFile(t, path, " \n")
	watch(t, path)
	if CurrentLevel() != LevelInfo {
		t.Fatalf("expected the start level for an empty file, got %q", CurrentLevel())
	}
}

func TestWatchLevelFileStopsWithContext(t *testing.T) {
	isolateLevel(t)
	path := filepath.Join(t.TempDir(), "level")
	writeLevelFile(t, path, "debug")
	ctx, cancel := context.WithCancel(context.Background())
	done := watchLevelFile(ctx, path)
	cancel()
	<-done
	writeLevelFile(t, path, "error")
	time.Sleep(5 * levelPollInterval)
	if CurrentLevel() != LevelDebug {
		t.Fatalf("expected no change after the context ended, got %q", CurrentLevel())
	}
}

func TestWatchLevelFileFromEnv(t *testing.T) {
	isolateLevel(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Setenv(LevelFileEnv, "")
	if WatchLevelFileFromEnv(ctx) {
		t.Fatalf("expected false with %s unset", LevelFileEnv)
	}
	path := filepath.Join(t.TempDir(), "level")
	writeLevelFile(t, path, "warn")
	t.Setenv(LevelFileEnv, path)
	if !WatchLevelFileFromEnv(ctx) {
		t.Fatalf("expected true with %s set", LevelFileEnv)
	}
	if CurrentLevel() != LevelWarn {
		t.Fatalf("expected the file level, got %q", CurrentLevel())
	}
}
