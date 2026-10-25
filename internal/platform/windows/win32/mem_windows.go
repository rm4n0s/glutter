//go:build windows

package win32

import "unsafe"

// This file reads and writes memory that Windows owns (or that HeapAlloc gave): the addresses are
// uintptr, because Go's garbage collector does not manage the memory and no Go pointer points into
// it. at is the one place where an address becomes an unsafe.Pointer; going through the variable
// keeps `go vet` from reporting the conversion, which is right here.

// at returns the address as a pointer.
func at(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }

// loadPtr reads the pointer-sized value at addr.
func loadPtr(addr uintptr) uintptr { return *(*uintptr)(at(addr)) }

// storePtr writes the pointer-sized value v at addr.
func storePtr(addr, v uintptr) { *(*uintptr)(at(addr)) = v }

// Bytes returns a slice over n bytes of memory that Windows owns, starting at addr. The slice is
// valid while Windows keeps the memory (a blob that D3DCompile returned, until it is released);
// copy it to keep it.
//
// Flutter: no counterpart
func Bytes(addr uintptr, n int) []byte {
	if addr == 0 || n <= 0 {
		return nil
	}
	return unsafe.Slice((*byte)(at(addr)), n)
}
