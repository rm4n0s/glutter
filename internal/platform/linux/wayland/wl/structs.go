//go:build linux

package wl

// Go mirrors of the C structs of wayland-util.h. They are only ever laid over
// C memory (see layout_test.go for the sizes and offsets from the header).

// cMessage mirrors struct wl_message.
type cMessage struct {
	Name      uintptr // const char *name
	Signature uintptr // const char *signature
	Types     uintptr // const struct wl_interface **types
}

// cInterface mirrors struct wl_interface.
type cInterface struct {
	Name        uintptr // const char *name
	Version     int32
	MethodCount int32
	Methods     uintptr // const struct wl_message *methods
	EventCount  int32
	Events      uintptr // const struct wl_message *events
}

// cArgument mirrors union wl_argument: eight bytes that hold an int32, a
// uint32, a wl_fixed_t, a pointer or a file descriptor.
type cArgument struct {
	Bits uint64
}

// cArray mirrors struct wl_array.
type cArray struct {
	Size  uintptr
	Alloc uintptr
	Data  uintptr
}
