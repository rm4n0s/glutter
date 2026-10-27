//go:build linux && !android

package wl

import (
	"testing"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl/layouttest"
)

// The sizes and offsets below come from wayland-util.h, compiled with gcc on
// 64-bit Linux (offsetof and sizeof of the C structs).
func TestStructLayouts(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the recorded layouts are for 64-bit pointers")
	}
	tests := []struct {
		name    string
		size    uintptr
		offsets map[string]uintptr
		got     any
	}{
		{"struct wl_message", 24, map[string]uintptr{"Name": 0, "Signature": 8, "Types": 16}, cMessage{}},
		{"struct wl_interface", 40, map[string]uintptr{
			"Name": 0, "Version": 8, "MethodCount": 12, "Methods": 16, "EventCount": 24, "Events": 32,
		}, cInterface{}},
		{"union wl_argument", 8, map[string]uintptr{"Bits": 0}, cArgument{}},
		{"struct wl_array", 24, map[string]uintptr{"Size": 0, "Alloc": 8, "Data": 16}, cArray{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layouttest.AssertLayout(t, tt.name, tt.size, tt.offsets, tt.got)
		})
	}
}
