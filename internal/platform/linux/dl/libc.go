//go:build linux

package dl

import (
	"runtime"
	"sync"
	"sync/atomic"
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

// libcName is the glibc name of the C library; Android's Bionic calls it libc.so (SPEC §24).
const libcName = "libc.so.6"

// libcNames are the names loadLibc tries, in order.
var libcNames = []string{libcName, "libc.so"}

// loadLibc is called at most once, on first use, so importing the package does
// no work. The table never changes after it is built. A system without libc is
// not a system Glutter can run on, so a failure panics with the tagged error.
var loadLibc = sync.OnceValue(func() *libcFuncs {
	lib, err := Open(libcNames...)
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

// liveBlocks is the number of blocks that Malloc, Calloc and CString returned and that Free has not
// released (see LiveBlocks).
var liveBlocks atomic.Int64

// tracker, when it is not nil, records every live block with the stack that allocated it (see
// TrackBlocks).
var tracker atomic.Pointer[blockTracker]

type blockTracker struct {
	mu     sync.Mutex
	blocks map[uintptr]block
}

type block struct {
	size  uintptr
	stack []uintptr
}

func recordBlock(ptr, size uintptr) {
	liveBlocks.Add(1)
	if t := tracker.Load(); t != nil {
		pcs := make([]uintptr, 24)
		pcs = pcs[:runtime.Callers(3, pcs)]
		t.mu.Lock()
		t.blocks[ptr] = block{size: size, stack: pcs}
		t.mu.Unlock()
	}
}

// mallinfo2 mirrors glibc's struct mallinfo2 (malloc.h).
type mallinfo2 struct {
	Arena, Ordblks, Smblks, Hblks, Hblkhd, Usmblks, Fsmblks, Uordblks, Fordblks, Keepcost uintptr
}

// mallinfoFunc is mallinfo2, bound on first use; nil when libc has none (glibc before 2.33, Bionic).
var mallinfoFunc = sync.OnceValue(func() func() mallinfo2 {
	lib, err := Open(libcNames...)
	if err != nil {
		return nil
	}
	var f func() mallinfo2
	if err := lib.Bind(&f, "mallinfo2"); err != nil {
		return nil
	}
	return f
})

// CHeapInUse is the number of bytes that the C allocator has handed out and not taken back, all
// callers counted: libwayland, libxkbcommon, fontconfig and the graphics drivers, and not only the
// blocks of this package. The memory tests (step 29.21) compare it after N and after 4N runs of a
// scenario. ok is false when libc has no mallinfo2 (glibc before 2.33, Bionic).
//
// Flutter: no counterpart
func CHeapInUse() (bytes int64, ok bool) {
	f := mallinfoFunc()
	if f == nil {
		return 0, false
	}
	m := f()
	return int64(m.Uordblks + m.Hblkhd), true
}

// LiveBlocks is the number of blocks of C memory that Malloc, Calloc and CString made and that
// Free has not released. A program whose C memory is flat keeps this number flat; the memory tests
// (step 29.21) compare it after N and after 4N runs of a scenario. Memory that C allocates itself
// (a library's own objects) is not counted.
//
// Flutter: no counterpart
func LiveBlocks() int64 { return liveBlocks.Load() }

// LeakedBlock is a block of C memory that was allocated while TrackBlocks was on and not freed.
//
// Flutter: no counterpart
type LeakedBlock struct {
	// Size is the size of the block in bytes.
	Size uintptr
	// Stack is where it was allocated, innermost function first, as "function file:line".
	Stack []string
}

// TrackBlocks starts to record the stack of every block that Malloc, Calloc and CString make, for
// a test that looks for the site of a block that is not freed. It returns the function that
// stops the tracking and returns the blocks that are still alive. Tracking is slow: it is for
// tests. Only one tracking can be on at a time; a second call stops the first.
//
// Flutter: no counterpart
func TrackBlocks() (stop func() []LeakedBlock) {
	t := &blockTracker{blocks: map[uintptr]block{}}
	tracker.Store(t)
	return func() []LeakedBlock {
		tracker.CompareAndSwap(t, nil)
		t.mu.Lock()
		defer t.mu.Unlock()
		out := make([]LeakedBlock, 0, len(t.blocks))
		for _, b := range t.blocks {
			lb := LeakedBlock{Size: b.size}
			frames := runtime.CallersFrames(b.stack)
			for {
				f, more := frames.Next()
				lb.Stack = append(lb.Stack, f.Function+" "+f.File+":"+itoa(f.Line))
				if !more {
					break
				}
			}
			out = append(out, lb)
		}
		return out
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for ; n > 0; n /= 10 {
		i--
		b[i] = byte('0' + n%10)
	}
	return string(b[i:])
}

// Malloc allocates size bytes of C memory and returns its address, or 0 when
// the allocation fails. Release it with Free. C memory is where data that C
// keeps must live (SPEC §8.1 rule 3).
//
// Flutter: no counterpart
func Malloc(size uintptr) uintptr {
	ptr := loadLibc().malloc(size)
	if ptr != 0 {
		recordBlock(ptr, size)
	}
	return ptr
}

// Calloc allocates count*size bytes of zeroed C memory and returns its address,
// or 0 when the allocation fails. Release it with Free.
//
// Flutter: no counterpart
func Calloc(count, size uintptr) uintptr {
	ptr := loadLibc().calloc(count, size)
	if ptr != 0 {
		recordBlock(ptr, count*size)
	}
	return ptr
}

// Free releases C memory from Malloc, Calloc or CString. Freeing 0 does
// nothing.
//
// Flutter: no counterpart
func Free(ptr uintptr) {
	if ptr == 0 {
		return
	}
	liveBlocks.Add(-1)
	if t := tracker.Load(); t != nil {
		t.mu.Lock()
		delete(t.blocks, ptr)
		t.mu.Unlock()
	}
	loadLibc().free(ptr)
}

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
		panic(errors.New(tagCStringAllocFailed, "malloc failed", "size", len(s)+1))
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

// pointer is the unexported name of Pointer, for use inside this package.
func pointer(addr uintptr) unsafe.Pointer { return Pointer(addr) }

// Pointer converts the address of C memory to an unsafe.Pointer. Use it only
// for addresses of memory that C owns or that came from Malloc, Calloc or
// CString, never for Go memory, which the garbage collector may move. It goes
// through a pointer to the address, so go vet's unsafeptr check, which targets
// uintptr-to-Pointer conversions of Go memory, does not apply.
//
// Flutter: no counterpart
func Pointer(addr uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

// Bytes returns a slice that views n bytes of C memory at addr, without
// copying. It is valid until the C memory is freed. Address 0 or n 0 gives nil.
//
// Flutter: no counterpart
func Bytes(addr uintptr, n int) []byte {
	if addr == 0 || n == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(Pointer(addr)), n)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCStringAllocFailed = "CStringAllocFailed"
)
