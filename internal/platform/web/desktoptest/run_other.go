//go:build !(js && wasm)

package main

import "github.com/rm4n0s/errors"

// run fails off the web: this program is for the browser tests.
func run() error {
	return errors.New("DesktopTestNeedsTheWeb", "desktoptest runs in a browser: build it with GOOS=js GOARCH=wasm")
}
