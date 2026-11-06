//go:build !android

// Command android-viewport3d is the program of the Android device test of the 3D view (step 30.9); see main.go. It needs the
// Android platform, so on other systems it only says so.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "android-viewport3d runs on Android: go run ./cmd/glutter-android run ./examples/dev/android-viewport3d")
	os.Exit(2)
}
