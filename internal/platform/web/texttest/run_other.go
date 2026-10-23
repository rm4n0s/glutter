//go:build !(js && wasm)

package main

import (
	"log"

	"github.com/rm4n0s/errors"
)

// main fails off the web: this program is for the browser tests.
func main() {
	log.Fatal(errors.New(tagTextTestNeedsTheWeb, "texttest runs in a browser: build it with GOOS=js GOARCH=wasm"))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagTextTestNeedsTheWeb = "TextTestNeedsTheWeb"
)
