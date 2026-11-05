//go:build !android

// Command android-scenes is the program of the Android device test of the explicit device (step 30.1);
// see main.go. It needs the Android platform, so on other systems it only says so.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "android-scenes runs on Android: go run ./cmd/glutter-android run ./examples/dev/android-scenes")
	os.Exit(2)
}
