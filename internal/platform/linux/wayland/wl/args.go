//go:build linux

package wl

import (
	"math"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

const argSize = unsafe.Sizeof(cArgument{})

// args is the union wl_argument array of one request, built in C memory. free
// releases it and the strings and arrays it points to.
type args struct {
	ptr    uintptr
	n      int
	allocs []uintptr
}

// newArgs allocates room for n arguments, zeroed. A request without arguments
// gets a nil array.
func newArgs(n int) *args {
	a := &args{n: n}
	if n > 0 {
		a.ptr = dl.Calloc(uintptr(n), argSize)
		if a.ptr == 0 {
			panic("wl: out of memory allocating request arguments")
		}
	}
	return a
}

func (a *args) slot(i int) unsafe.Pointer {
	return dl.Pointer(a.ptr + uintptr(i)*argSize)
}

func (a *args) setInt(i int, v int32)   { *(*int32)(a.slot(i)) = v }
func (a *args) setUint(i int, v uint32) { *(*uint32)(a.slot(i)) = v }

// setFixed stores v as a wl_fixed_t: a signed 24.8 fixed-point number.
func (a *args) setFixed(i int, v float64) { *(*int32)(a.slot(i)) = int32(math.Round(v * 256)) }

// setObject stores the wl_proxy pointer of an object, or 0 for NULL.
func (a *args) setObject(i int, proxy uintptr) { *(*uintptr)(a.slot(i)) = proxy }

// setFD stores a file descriptor. libwayland duplicates it while sending, so
// the caller keeps ownership.
func (a *args) setFD(i int, fd int32) { *(*int32)(a.slot(i)) = fd }

// setString stores a copy of s in C memory.
func (a *args) setString(i int, s string) {
	p := dl.CString(s)
	a.allocs = append(a.allocs, p)
	*(*uintptr)(a.slot(i)) = p
}

// setNullableString is setString, except that the empty string is NULL.
func (a *args) setNullableString(i int, s string) {
	if s == "" {
		return
	}
	a.setString(i, s)
}

// setArray stores a copy of b as a struct wl_array in C memory.
func (a *args) setArray(i int, b []byte) {
	arr := dl.Calloc(1, unsafe.Sizeof(cArray{}))
	if arr == 0 {
		panic("wl: out of memory allocating a wl_array")
	}
	a.allocs = append(a.allocs, arr)
	if len(b) > 0 {
		data := dl.Malloc(uintptr(len(b)))
		if data == 0 {
			panic("wl: out of memory allocating wl_array data")
		}
		a.allocs = append(a.allocs, data)
		copy(dl.Bytes(data, len(b)), b)
		c := (*cArray)(dl.Pointer(arr))
		c.Size, c.Alloc, c.Data = uintptr(len(b)), uintptr(len(b)), data
	}
	*(*uintptr)(a.slot(i)) = arr
}

func (a *args) free() {
	for _, p := range a.allocs {
		dl.Free(p)
	}
	a.allocs = nil
	dl.Free(a.ptr)
	a.ptr = 0
}

// eventArgs is the union wl_argument array that libwayland passes to the
// dispatcher. It is valid only during the call.
type eventArgs uintptr

func (a eventArgs) slot(i int) unsafe.Pointer {
	return dl.Pointer(uintptr(a) + uintptr(i)*argSize)
}

func (a eventArgs) int(i int) int32   { return *(*int32)(a.slot(i)) }
func (a eventArgs) uint(i int) uint32 { return *(*uint32)(a.slot(i)) }
func (a eventArgs) fd(i int) int32    { return *(*int32)(a.slot(i)) }

// fixed reads a wl_fixed_t.
func (a eventArgs) fixed(i int) float64 { return float64(*(*int32)(a.slot(i))) / 256 }

// object reads the wl_proxy pointer of an object argument; 0 means NULL.
func (a eventArgs) object(i int) uintptr { return *(*uintptr)(a.slot(i)) }

// str copies a string argument. NULL reads as "".
func (a eventArgs) str(i int) string { return dl.GoString(*(*uintptr)(a.slot(i))) }

// array copies a wl_array argument.
func (a eventArgs) array(i int) []byte {
	arr := *(*uintptr)(a.slot(i))
	if arr == 0 {
		return nil
	}
	c := (*cArray)(dl.Pointer(arr))
	if c.Size == 0 || c.Data == 0 {
		return nil
	}
	return append([]byte(nil), dl.Bytes(c.Data, int(c.Size))...)
}

// uint32s copies a wl_array argument that holds 32-bit unsigned integers.
func (a eventArgs) uint32s(i int) []uint32 {
	raw := a.array(i)
	if len(raw) < 4 {
		return nil
	}
	out := make([]uint32, len(raw)/4)
	for k := range out {
		out[k] = *(*uint32)(unsafe.Pointer(&raw[k*4]))
	}
	return out
}

// uint32sToBytes lays out 32-bit unsigned integers in memory order, as a
// wl_array of uint32 holds them.
func uint32sToBytes(v []uint32) []byte {
	out := make([]byte, len(v)*4)
	for i, x := range v {
		*(*uint32)(unsafe.Pointer(&out[i*4])) = x
	}
	return out
}
