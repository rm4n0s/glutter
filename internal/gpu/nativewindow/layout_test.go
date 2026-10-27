//go:build linux

package nativewindow

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl/layouttest"
)

// TestWindowBufferLayout checks the Go mirror of ANativeWindow_Buffer against the C layout of
// <android/native_window.h> on a 64-bit ABI: four int32, a pointer at offset 16, six reserved
// uint32, 48 bytes in all (SPEC §8.1).
func TestWindowBufferLayout(t *testing.T) {
	var b windowBuffer
	layouttest.AssertLayout(t, "ANativeWindow_Buffer", bufferSize,
		map[string]uintptr{"Width": 0, "Height": 4, "Stride": 8, "Format": 12, "Bits": 16}, b)
}
