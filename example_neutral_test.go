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
	"context"
	"errors"

	log "github.com/Bugs5382/go-log"
)

// ExampleLogger shows the neutral surface: a consumer only needs the Logger
// interface, F, NewLogger, and LoggerFromContext -- zerolog is never
// imported or referenced.
func ExampleLogger() {
	logger := log.NewLogger("billing")

	logger.Info("service started", log.F("port", 8080))

	child := logger.With(log.F("request_id", "r-123"))
	child.Warn("slow downstream call", log.F("elapsed_ms", 420))

	err := errors.New("payment gateway timeout")
	child.Error(err, "request failed")

	// Inside a request/span, correlate logs with the active trace without
	// ever touching a zerolog type.
	ctx := context.Background()
	log.LoggerFromContext(ctx).Info("handling request")
}
