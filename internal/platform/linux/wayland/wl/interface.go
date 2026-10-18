package wl

import (
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Interface describes one Wayland interface. Its tables live in C memory,
// because libwayland reads them for as long as the process runs; they are never
// freed.
//
// Flutter: no counterpart
type Interface struct {
	// Name is the interface name, such as "wl_surface".
	Name string
	// Version is the highest version of the interface that this package knows.
	Version uint32

	ptr uintptr // struct wl_interface in C memory
}

// message describes one request or event for the interface tables.
type message struct {
	name      string
	signature string       // wl_message.signature, e.g. "4?oii"
	types     []*Interface // one per argument slot; nil for arguments that are not objects
}

// newInterface allocates the C memory for an interface. The messages are added
// later by setMessages, because they refer to other interfaces.
func newInterface(name string, version uint32) *Interface {
	ptr := dl.Calloc(1, unsafe.Sizeof(cInterface{}))
	if ptr == 0 {
		panic("wl: out of memory allocating an interface table")
	}
	c := (*cInterface)(dl.Pointer(ptr))
	c.Name = dl.CString(name)
	c.Version = int32(version)
	return &Interface{Name: name, Version: version, ptr: ptr}
}

// setMessages writes the request and event tables of the interface.
func (i *Interface) setMessages(requests, events []message) {
	c := (*cInterface)(dl.Pointer(i.ptr))
	c.Methods, c.MethodCount = buildMessages(requests)
	c.Events, c.EventCount = buildMessages(events)
}

// buildMessages allocates a wl_message array in C memory.
func buildMessages(msgs []message) (uintptr, int32) {
	if len(msgs) == 0 {
		return 0, 0
	}
	size := unsafe.Sizeof(cMessage{})
	array := dl.Calloc(uintptr(len(msgs)), size)
	if array == 0 {
		panic("wl: out of memory allocating a message table")
	}
	for n, m := range msgs {
		c := (*cMessage)(dl.Pointer(array + uintptr(n)*size))
		c.Name = dl.CString(m.name)
		c.Signature = dl.CString(m.signature)
		// An array of at least one slot, so Types is never NULL.
		slots := max(len(m.types), 1)
		types := dl.Calloc(uintptr(slots), unsafe.Sizeof(uintptr(0)))
		if types == 0 {
			panic("wl: out of memory allocating a type table")
		}
		for k, t := range m.types {
			if t != nil {
				*(*uintptr)(dl.Pointer(types + uintptr(k)*unsafe.Sizeof(uintptr(0)))) = t.ptr
			}
		}
		c.Types = types
	}
	return array, int32(len(msgs))
}
