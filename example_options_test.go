package log_test

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
	"os"

	log "github.com/Bugs5382/go-log"
)

// ExampleNewLoggerWithOptions shows a CLI setup: logs on stderr so they never
// mix with command output, console text unless LOG_FORMAT asks for JSON, and
// a -v flag that selects trace unless LOG_LEVEL says otherwise.
func ExampleNewLoggerWithOptions() {
	verbose := true

	level := log.LevelInfo
	if verbose {
		level = log.LevelTrace
	}

	logger := log.NewLoggerWithOptions("my-cli",
		log.WithOutput(os.Stderr),
		log.WithDefaultFormat(log.FormatConsole),
		log.WithDefaultLevel(level),
	)

	logger.Trace("resolved config", log.F("path", "config.yaml"))
	logger.Info("done", log.F("files", 12))
}

// ExampleNop shows the no-op Logger for tests and quiet modes.
func ExampleNop() {
	var logger log.Logger = log.Nop()
	logger.Info("never written")
	log.Trace(logger, "never written either")
}
