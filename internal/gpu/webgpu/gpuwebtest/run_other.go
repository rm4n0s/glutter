//go:build !(js && wasm)

package main

import (
	"log/slog"
	"os"

	"github.com/rm4n0s/errors"
)

// main fails off the web: this program is for the browser tests.
func main() {
	err := errors.New(tagGPUWebTestNeedsTheWeb, "gpuwebtest runs in a browser: build it with GOOS=js GOARCH=wasm")
	slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("gpuwebtest did not start", "err", err)
	os.Exit(1)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGPUWebTestNeedsTheWeb = "GPUWebTestNeedsTheWeb"
)
