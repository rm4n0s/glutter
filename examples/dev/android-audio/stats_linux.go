//go:build linux

package main

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/android/aaudio"
)

// snapshot is the timestamp, the dropouts and the buffer of an AAudio device; ok is false (and the rest 0) on a device of another
// backend.
func snapshot(dev platform.AudioDevice) (frame, nanos int64, ok bool, xruns, buffer, burst int) {
	s, has := dev.(aaudio.Stats)
	if !has {
		return 0, 0, false, 0, 0, 0
	}
	frame, nanos, ok = s.Timestamp()
	buffer, burst = s.BufferFrames()
	return frame, nanos, ok, s.XRuns(), buffer, burst
}
