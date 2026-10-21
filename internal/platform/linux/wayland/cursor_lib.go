//go:build linux

package wayland

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// waylandCursorLibrary is the versioned name of libwayland-cursor (SPEC §8.1).
const waylandCursorLibrary = "libwayland-cursor.so.0"

// wlCursor is struct wl_cursor.
type wlCursor struct {
	imageCount uint32
	_          uint32
	images     uintptr // struct wl_cursor_image **
	name       uintptr // char *
}

// wlCursorImage is struct wl_cursor_image.
type wlCursorImage struct {
	width, height      uint32
	hotspotX, hotspotY uint32
	delay              uint32 // milliseconds, for animated cursors
}

// waylandCursor is the part of libwayland-cursor that loads an XCursor theme into
// wl_shm buffers.
type waylandCursor struct {
	themeLoad      func(name uintptr, size int32, shm uintptr) uintptr
	themeDestroy   func(theme uintptr)
	themeGetCursor func(theme, name uintptr) uintptr
	imageGetBuffer func(image uintptr) uintptr
}

// loadWaylandCursor opens libwayland-cursor once. A failure (LibraryLoadFailed,
// or SymbolMissing) is returned on every call.
var loadWaylandCursor = sync.OnceValues(func() (*waylandCursor, error) {
	lib, err := dl.Open(waylandCursorLibrary)
	if err != nil {
		return nil, err
	}
	api := new(waylandCursor)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&api.themeLoad, "wl_cursor_theme_load"},
		{&api.themeDestroy, "wl_cursor_theme_destroy"},
		{&api.themeGetCursor, "wl_cursor_theme_get_cursor"},
		{&api.imageGetBuffer, "wl_cursor_image_get_buffer"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return api, nil
})

// cursorImage returns the first image of the named cursor in the theme: the
// image, its buffer and its hotspot. It reports false when the theme has no such
// cursor. Animated cursors use their first image.
//
// TODO(step 14.x): animate the cursors that have more than one image.
func (a *waylandCursor) cursorImage(theme uintptr, name string) (image wlCursorImage, buffer uintptr, ok bool) {
	cname := dl.CString(name)
	defer dl.Free(cname)
	c := a.themeGetCursor(theme, cname)
	if c == 0 {
		return wlCursorImage{}, 0, false
	}
	cur := (*wlCursor)(dl.Pointer(c))
	if cur.imageCount == 0 || cur.images == 0 {
		return wlCursorImage{}, 0, false
	}
	first := *(*uintptr)(dl.Pointer(cur.images)) // images[0]
	return *(*wlCursorImage)(dl.Pointer(first)), a.imageGetBuffer(first), true
}
