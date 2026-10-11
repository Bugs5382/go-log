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
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LevelFileEnv names the environment variable WatchLevelFileFromEnv reads: the
// path of a file holding one level word.
const LevelFileEnv = "LOG_LEVEL_FILE"

// The live level is process-wide. Every logger this package builds is created
// at trace and filtered by zerolog's global level, which zerolog reads
// atomically on every event, so changing it reaches loggers that already exist,
// their children, and package-level loggers built at init.
//
// startLevel is the level the most recent New or NewLoggerWithOptions resolved
// from LOG_LEVEL and the default. Until SetLevel (or a level file) takes over,
// each build applies its start level; once overridden, later builds leave the
// live level alone so a new logger never undoes an operator's change.
var (
	levelMu    sync.Mutex
	startLevel = zerolog.InfoLevel
	overridden bool
)

// levelPollInterval is how often WatchLevelFile re-reads its file.
var levelPollInterval = 2 * time.Second

// applyStartLevel records l as the start level and applies it unless a live
// level is in force.
func applyStartLevel(l zerolog.Level) {
	levelMu.Lock()
	defer levelMu.Unlock()
	startLevel = l
	if !overridden {
		zerolog.SetGlobalLevel(l)
	}
}

// parseLevel accepts the LOG_LEVEL vocabulary only, case-insensitively.
func parseLevel(s string) (zerolog.Level, error) {
	switch Level(strings.ToLower(strings.TrimSpace(s))) {
	case LevelTrace, LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal, LevelPanic, LevelDisabled:
		return zerolog.ParseLevel(strings.ToLower(strings.TrimSpace(s)))
	}
	return zerolog.NoLevel, fmt.Errorf("log: unknown level %q", s)
}

// SetLevel changes the minimum level of every logger in the process, live: no
// logger needs rebuilding. It accepts the LOG_LEVEL names (trace, debug, info,
// warn, error, fatal, panic, disabled), case-insensitively. An unknown name
// returns an error and leaves the level as it was.
//
// LOG_LEVEL stays the start value; SetLevel replaces it until the process ends,
// and loggers built afterwards keep the level SetLevel chose.
func SetLevel(l Level) error {
	lvl, err := parseLevel(string(l))
	if err != nil {
		return err
	}
	levelMu.Lock()
	defer levelMu.Unlock()
	overridden = true
	zerolog.SetGlobalLevel(lvl)
	return nil
}

// CurrentLevel returns the live minimum level.
func CurrentLevel() Level {
	return Level(zerolog.GlobalLevel().String())
}

// resetToStartLevel drops a live level and goes back to the start level.
func resetToStartLevel() {
	levelMu.Lock()
	defer levelMu.Unlock()
	overridden = false
	zerolog.SetGlobalLevel(startLevel)
}

// WatchLevelFile applies the level named in the file at path, then keeps
// applying it whenever the file changes, until ctx ends. The file holds one
// level word (trace, debug, info, warn or error; surrounding whitespace is
// ignored). It is read once before WatchLevelFile returns and then every two
// seconds in the background, following symlinks on each read, so a Kubernetes
// ConfigMap volume (which swaps a symlink when the ConfigMap changes) works as
// the source.
//
// An unknown word keeps the current level and logs a warning. A missing,
// unreadable or empty file means the start level from LOG_LEVEL. The watcher's
// own lines (a level change, a bad word, a missing file) are written whatever
// the level is, since they describe the level itself; only disabled silences
// them. They go to the logger built most recently by this package.
func WatchLevelFile(ctx context.Context, path string) {
	watchLevelFile(ctx, path)
}

// WatchLevelFileFromEnv calls WatchLevelFile with the path in LOG_LEVEL_FILE and
// reports whether the variable was set. With it unset, nothing is watched and
// LOG_LEVEL alone sets the level.
func WatchLevelFileFromEnv(ctx context.Context) bool {
	path := os.Getenv(LevelFileEnv)
	if path == "" {
		return false
	}
	WatchLevelFile(ctx, path)
	return true
}

// watchLevelFile does the work of WatchLevelFile and returns a channel closed
// once the background loop has stopped.
func watchLevelFile(ctx context.Context, path string) <-chan struct{} {
	w := &levelWatcher{path: path}
	w.check()
	done := make(chan struct{})
	t := time.NewTicker(levelPollInterval)
	go func() {
		defer close(done)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				w.check()
			}
		}
	}()
	return done
}

type levelWatcher struct {
	path string
	seen bool
	last string
}

// check reads the file and acts only when its content (or its absence) has
// changed since the last read, so a steady file costs one read per interval
// and logs nothing.
func (w *levelWatcher) check() {
	b, err := os.ReadFile(w.path)
	state := strings.TrimSpace(string(b))
	if err != nil {
		state = "\x00" + err.Error()
	}
	if w.seen && state == w.last {
		return
	}
	w.seen = true
	w.last = state

	from := CurrentLevel()
	switch {
	case err != nil:
		resetToStartLevel()
		watcherLine("warn").Err(err).Str("path", w.path).Str("level_to", string(CurrentLevel())).
			Msg("log level file unreadable; using the start level")
	case state == "":
		resetToStartLevel()
		watcherLine("info").Str("path", w.path).Str("level_to", string(CurrentLevel())).
			Msg("log level file empty; using the start level")
	default:
		if err := SetLevel(Level(state)); err != nil {
			word := state
			if len(word) > 32 {
				word = word[:32]
			}
			watcherLine("warn").Str("path", w.path).Str("word", word).Str("level_current", string(from)).
				Msg("log level file holds an unknown level; keeping the current level")
			return
		}
		if CurrentLevel() != from {
			watcherLine("info").Str("path", w.path).Str("level_from", string(from)).Str("level_to", string(CurrentLevel())).
				Msg("log level changed")
		}
	}
}

// watcherLine starts a line that bypasses the live level (only disabled stops
// it) and still carries its level name.
func watcherLine(level string) *zerolog.Event {
	base := zerolog.New(os.Stderr).With().Timestamp().Logger()
	if p := baseLogger.Load(); p != nil {
		base = *p
	}
	return base.WithLevel(zerolog.NoLevel).Str(zerolog.LevelFieldName, level)
}
