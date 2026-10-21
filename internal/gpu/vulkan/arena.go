//go:build linux

package vulkan

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// arenaChunk is the size of the blocks of C memory that an arena asks libc for.
const arenaChunk = 64 << 10

// arena hands out zeroed C memory for the create infos and the arrays they point to
// (SPEC §8.1, §9.4): Vulkan structs that point to other data cannot be built in Go
// memory, which the garbage collector may move. Nothing is freed one by one; reset
// makes the memory available again, and free gives it back to libc. It is used on
// the UI thread only.
type arena struct {
	chunks []arenaBlock
	cur    int // the chunk that alloc takes from
}

type arenaBlock struct {
	base uintptr
	size uintptr
	used uintptr
}

// arenaMark is a position of the arena, for rewind.
type arenaMark struct {
	cur  int
	used uintptr
}

// alloc returns size bytes of zeroed C memory, aligned to align (a power of two,
// at most 16). A request larger than a chunk gets a chunk of its own.
func (a *arena) alloc(size, align uintptr) uintptr {
	if size == 0 {
		size = 1
	}
	for ; a.cur < len(a.chunks); a.cur++ {
		c := &a.chunks[a.cur]
		start := (c.used + align - 1) &^ (align - 1)
		if start+size <= c.size {
			c.used = start + size
			p := c.base + start
			clear(dl.Bytes(p, int(size)))
			return p
		}
	}
	n := max(uintptr(arenaChunk), size+align)
	base := dl.Calloc(1, n)
	start := (base+align-1)&^(align-1) - base
	a.chunks = append(a.chunks, arenaBlock{base: base, size: n, used: start + size})
	a.cur = len(a.chunks) - 1
	return base + start
}

// mark returns the position of the arena, so that rewind can free what was
// allocated after it.
func (a *arena) mark() arenaMark {
	if a.cur >= len(a.chunks) {
		return arenaMark{cur: a.cur}
	}
	return arenaMark{cur: a.cur, used: a.chunks[a.cur].used}
}

// rewind makes the memory allocated after the mark available again.
func (a *arena) rewind(m arenaMark) {
	for i := m.cur + 1; i < len(a.chunks); i++ {
		a.chunks[i].used = 0
	}
	if m.cur < len(a.chunks) {
		a.chunks[m.cur].used = m.used
	}
	a.cur = m.cur
}

// reset makes all the memory available again.
func (a *arena) reset() {
	for i := range a.chunks {
		a.chunks[i].used = 0
	}
	a.cur = 0
}

// free gives the memory back to libc. The arena can be used again.
func (a *arena) free() {
	for _, c := range a.chunks {
		dl.Free(c.base)
	}
	a.chunks, a.cur = nil, 0
}

// put copies v into C memory and returns its address. T must not hold Go pointers:
// the mirrors of Vulkan structs have uintptr and integers only.
func put[T any](a *arena, v T) uintptr {
	var zero T
	p := a.alloc(unsafe.Sizeof(zero), max(unsafe.Alignof(zero), 8))
	*(*T)(dl.Pointer(p)) = v
	return p
}

// putSlice copies the elements into C memory and returns the address of the first;
// 0 for an empty slice.
func putSlice[T any](a *arena, vs []T) uintptr {
	if len(vs) == 0 {
		return 0
	}
	var zero T
	p := a.alloc(unsafe.Sizeof(zero)*uintptr(len(vs)), max(unsafe.Alignof(zero), 8))
	copy(unsafe.Slice((*T)(dl.Pointer(p)), len(vs)), vs)
	return p
}

// cstring copies s into C memory with a terminating zero.
func (a *arena) cstring(s string) uintptr {
	p := a.alloc(uintptr(len(s))+1, 1)
	copy(dl.Bytes(p, len(s)), s)
	return p
}

// cstrings returns an array of char* for the strings, as ppEnabledExtensionNames is;
// 0 for no strings.
func (a *arena) cstrings(ss []string) uintptr {
	if len(ss) == 0 {
		return 0
	}
	ptrs := make([]uintptr, len(ss))
	for i, s := range ss {
		ptrs[i] = a.cstring(s)
	}
	return putSlice(a, ptrs)
}

// out returns the address of 8 bytes of zeroed C memory, for a result that a call
// writes: a handle, a count, a pointer.
func (a *arena) out() uintptr { return a.alloc(8, 8) }

// u32 reads a uint32 at the address, which holds a result that a call wrote.
func u32(p uintptr) uint32 { return *(*uint32)(dl.Pointer(p)) }

// u64 reads a uint64 at the address.
func u64(p uintptr) uint64 { return *(*uint64)(dl.Pointer(p)) }

// ptr reads a pointer at the address.
func ptr(p uintptr) uintptr { return *(*uintptr)(dl.Pointer(p)) }

// readSlice copies n elements of C memory at p into a Go slice.
func readSlice[T any](p uintptr, n int) []T {
	if p == 0 || n <= 0 {
		return nil
	}
	out := make([]T, n)
	copy(out, unsafe.Slice((*T)(dl.Pointer(p)), n))
	return out
}

// The sizes of the arrays that Vulkan fills in, for the allocations of the counts a
// call returned.
const (
	sizeofExtensionProperties = unsafe.Sizeof(extensionProperties{})
	sizeofLayerProperties     = unsafe.Sizeof(layerProperties{})
)

// unsafeSizeof returns the size of T in bytes.
func unsafeSizeof[T any]() uintptr {
	var zero T
	return unsafe.Sizeof(zero)
}

const sizeofMemoryProperties = unsafe.Sizeof(physicalDeviceMemoryProperties{})

// readBytes copies n bytes of C memory at p into a Go slice.
func readBytes(p uintptr, n int) []byte {
	out := make([]byte, n)
	copy(out, dl.Bytes(p, n))
	return out
}
