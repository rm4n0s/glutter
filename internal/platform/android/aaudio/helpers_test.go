//go:build linux

package aaudio

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// unsafeFloats views n float32 of C memory.
func unsafeFloats(addr uintptr, n int) []float32 {
	return unsafe.Slice((*float32)(dl.Pointer(addr)), n)
}
