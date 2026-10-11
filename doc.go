// Package log provides zerolog-based structured logging for Go services, with
// OpenTelemetry trace_id/span_id correlation drawn from the active span
// context.
//
// New and Ctx return a concrete zerolog.Logger for callers that already
// depend on zerolog. NewLogger and LoggerFromContext return the neutral
// Logger interface instead: no zerolog type appears in its method
// signatures, so a consumer can wrap or depend on it without importing
// zerolog directly. Both paths honor the same LOG_LEVEL/LOG_FORMAT
// environment controls and the same trace correlation; zerolog stays an
// internal implementation detail behind the neutral path.
//
// NewLoggerWithOptions builds a neutral Logger tuned by options: WithOutput
// picks the io.Writer (a CLI uses os.Stderr), and WithDefaultLevel and
// WithDefaultFormat set the fallbacks used when LOG_LEVEL or LOG_FORMAT is
// unset or invalid. It returns a TraceLogger, which adds Trace; Nop returns a
// TraceLogger that writes nothing.
//
// LOG_LEVEL sets the start level. SetLevel changes the level live for the
// whole process, reaching loggers that already exist, and WatchLevelFile (or
// WatchLevelFileFromEnv with LOG_LEVEL_FILE) follows a one-word level file
// such as a mounted Kubernetes ConfigMap key.
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
