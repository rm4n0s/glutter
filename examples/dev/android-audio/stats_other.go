//go:build !linux

package main

import "github.com/rm4n0s/glutter/internal/platform"

// snapshot: only the AAudio backend has these numbers, and it is built for Linux and Android only.
func snapshot(dev platform.AudioDevice) (frame, nanos int64, ok bool, xruns, buffer, burst int) {
	return 0, 0, false, 0, 0, 0
}
