//go:build windows

package main

import (
	"runtime"

	"github.com/rm4n0s/glutter/internal/wintool"
)

// hostArch is the architecture that run and shot build for: this machine's.
var hostArch = runtime.GOARCH

// newLauncher makes the launcher that runs the .exe on this machine: the program itself. headless
// has no meaning on Windows.
func newLauncher(bool) (wintool.Launcher, error) { return wintool.LocalLauncher{}, nil }
