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
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// Level names a minimum log level for WithDefaultLevel. The values match the
// LOG_LEVEL vocabulary, so a string from a flag or config file converts with
// Level(s). An empty or unrecognized Level means info.
type Level string

// The levels WithDefaultLevel accepts, from most to least verbose.
const (
	LevelTrace    Level = "trace"
	LevelDebug    Level = "debug"
	LevelInfo     Level = "info"
	LevelWarn     Level = "warn"
	LevelError    Level = "error"
	LevelFatal    Level = "fatal"
	LevelPanic    Level = "panic"
	LevelDisabled Level = "disabled"
)

// Format names an output format for WithDefaultFormat. The values match the
// LOG_FORMAT vocabulary. An empty or unrecognized Format means JSON.
type Format string

// The formats WithDefaultFormat accepts.
const (
	// FormatJSON writes structured JSON. It is the default.
	FormatJSON Format = "json"
	// FormatConsole writes human-readable, colorized text.
	FormatConsole Format = "console"
	// FormatBoth writes JSON to the output and a console rendering to stderr.
	FormatBoth Format = "both"
)

// config holds what the Options set. Start from defaultConfig, not the zero
// value.
type config struct {
	out    io.Writer
	level  zerolog.Level
	format Format
}

func defaultConfig() config {
	return config{out: os.Stdout, level: zerolog.InfoLevel, format: FormatJSON}
}

// Option configures a Logger built by NewLoggerWithOptions.
type Option func(*config)

// WithOutput sends log output to w instead of stdout, whatever the format. A
// CLI passes os.Stderr so its logs never mix with what it prints on stdout.
// With LOG_FORMAT=both the JSON half goes to w and the console half still goes
// to stderr. A nil w leaves the output on stdout.
func WithOutput(w io.Writer) Option {
	return func(c *config) {
		if w != nil {
			c.out = w
		}
	}
}

// WithDefaultLevel sets the level used when LOG_LEVEL is unset or not a level
// name, in place of info. LOG_LEVEL still wins whenever it is valid, so a
// deployment can always override the code. A CLI maps its -v flag here.
func WithDefaultLevel(l Level) Option {
	return func(c *config) {
		lvl, err := zerolog.ParseLevel(strings.ToLower(string(l)))
		// ParseLevel accepts "" as "no level", which would log everything.
		if err != nil || l == "" {
			lvl = zerolog.InfoLevel
		}
		c.level = lvl
	}
}

// WithDefaultFormat sets the format used when LOG_FORMAT is unset or
// unrecognized, in place of JSON. LOG_FORMAT still wins whenever it is valid.
// A CLI passes FormatConsole so people read plain text unless they ask for
// JSON with LOG_FORMAT=json.
func WithDefaultFormat(f Format) Option {
	return func(c *config) {
		c.format = Format(strings.ToLower(string(f)))
	}
}

// NewLoggerWithOptions returns a neutral Logger for a service, like NewLogger,
// configured by opts. With no options it behaves exactly like NewLogger. The
// environment still has the last word: LOG_LEVEL and LOG_FORMAT override the
// defaults the options set whenever they hold a valid value.
//
// The result is a TraceLogger, so Trace can be called on it directly. Like
// NewLogger, it also becomes the logger the package-level Ctx and
// LoggerFromContext derive from.
func NewLoggerWithOptions(service string, opts ...Option) TraceLogger {
	c := defaultConfig()
	for _, o := range opts {
		o(&c)
	}
	return neutralLogger{l: build(service, c)}
}
