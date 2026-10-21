//go:build linux

package wayland

import (
	"testing"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// The offsets and sizes are those of wayland-cursor.h on a 64-bit system.
func TestWlCursorLayout(t *testing.T) {
	dl.AssertLayout(t, "wl_cursor", 24, map[string]uintptr{"imageCount": 0, "images": 8, "name": 16}, wlCursor{})
	dl.AssertLayout(t, "wl_cursor_image", 20, map[string]uintptr{
		"width": 0, "height": 4, "hotspotX": 8, "hotspotY": 12, "delay": 16,
	}, wlCursorImage{})
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("the layouts above are for 64-bit systems")
	}
}
