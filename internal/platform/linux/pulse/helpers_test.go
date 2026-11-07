//go:build linux && !android

package pulse

import "unsafe"

// unsafeFloats views n float32 of C memory.
func unsafeFloats(addr uintptr, n int) []float32 {
	return unsafe.Slice((*float32)(*(*unsafe.Pointer)(unsafe.Pointer(&addr))), n)
}
