//go:build !android

// Command android-parity is the parity program of the Android device tests (step 25.4); see main.go.
// It needs the Android platform, so on other systems it only says so.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "android-parity runs on Android: go run ./cmd/glutter-android run ./examples/dev/android-parity")
	os.Exit(2)
}
