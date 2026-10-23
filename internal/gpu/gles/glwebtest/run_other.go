//go:build !(js && wasm)

package main

import (
	"log/slog"
	"os"

	"github.com/rm4n0s/errors"
)

// main fails off the web: this program is for the browser tests.
func main() {
	err := errors.New(tagGLWebTestNeedsTheWeb, "glwebtest runs in a browser: build it with GOOS=js GOARCH=wasm")
	slog.Error("glwebtest", "err", err)
	os.Exit(1)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGLWebTestNeedsTheWeb = "GLWebTestNeedsTheWeb"
)
