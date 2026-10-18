package dl

import (
	"sync"
	"unsafe"

	"github.com/rm4n0s/errors"
)

// libcFuncs holds the libc functions this package binds.
type libcFuncs struct {
	malloc func(size uintptr) uintptr
	calloc func(count, size uintptr) uintptr
	free   func(ptr uintptr)
	strlen func(ptr uintptr) uintptr
}

// libcName is the glibc name of the C library.
const libcName = "libc.so.6"

// loadLibc is called at most once, on first use, so importing the package does
// no work. The table never changes after it is built. A system without libc is
// not a system Glutter can run on, so a failure panics with the tagged error.
var loadLibc = sync.OnceValue(func() *libcFuncs {
	lib, err := Open(libcName)
	if err != nil {
		panic(err)
	}
	f := new(libcFuncs)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&f.malloc, "malloc"},
		{&f.calloc, "calloc"},
		{&f.free, "free"},
		{&f.strlen, "strlen"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			panic(err)
		}
	}
	return f
})

// Malloc allocates size bytes of C memory and returns its address, or 0 when
// the allocation fails. Release it with Free. C memory is where data that C
// keeps must live (SPEC §8.1 rule 3).
//
// Flutter: no counterpart
func Malloc(size uintptr) uintptr { return loadLibc().malloc(size) }

// Calloc allocates count*size bytes of zeroed C memory and returns its address,
// or 0 when the allocation fails. Release it with Free.
//
// Flutter: no counterpart
func Calloc(count, size uintptr) uintptr { return loadLibc().calloc(count, size) }

// Free releases C memory from Malloc, Calloc or CString. Freeing 0 does
// nothing.
//
// Flutter: no counterpart
func Free(ptr uintptr) { loadLibc().free(ptr) }

// Strlen returns the length in bytes of the NUL-terminated C string at ptr,
// not counting the NUL.
//
// Flutter: no counterpart
func Strlen(ptr uintptr) uintptr { return loadLibc().strlen(ptr) }

// CString copies s into new C memory as a NUL-terminated string and returns its
// address. The caller releases it with Free. An s with a NUL byte inside reads
// as truncated on the C side. It panics with CStringAllocFailed when malloc
// fails.
//
// Flutter: no counterpart
func CString(s string) uintptr {
	ptr := Malloc(uintptr(len(s)) + 1)
	if ptr == 0 {
		panic(errors.New("CStringAllocFailed", "malloc failed", "size", len(s)+1))
	}
	buf := unsafe.Slice((*byte)(pointer(ptr)), len(s)+1)
	copy(buf, s)
	buf[len(s)] = 0
	return ptr
}

// GoString copies the NUL-terminated C string at ptr into a Go string. The
// address 0 gives "".
//
// Flutter: no counterpart
func GoString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	n := int(Strlen(ptr))
	return string(unsafe.Slice((*byte)(pointer(ptr)), n))
}

// pointer converts the address of C memory to an unsafe.Pointer. It goes
// through a pointer to the address, so go vet's unsafeptr check, which targets
// Go memory moved by the garbage collector, does not apply: addr is never Go
// memory.
func pointer(addr uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}
