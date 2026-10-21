//go:build linux

package wayland

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
	"github.com/rm4n0s/glutter/internal/platform/linux/input"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/fakecomp"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// The capability bits of wl_seat.
const (
	capPointer = 1
	capTouch   = 4
)

// startInput starts a platform on a fake compositor whose seat has the given
// capabilities and the extra globals, with a window.
func startInput(t *testing.T, caps uint32, seatVersion uint32, extra ...fakecomp.Global) *keyEnv {
	t.Helper()
	globals := fakecomp.DefaultGlobals()
	for i := range globals {
		if globals[i].Interface == "wl_seat" && seatVersion != 0 {
			globals[i].Version = seatVersion
		}
	}
	return startKeyboardNoFocus(t, fakecomp.Config{Capabilities: caps, Globals: append(globals, extra...)})
}

var (
	gesturesGlobal    = fakecomp.Global{Name: 20, Interface: "zwp_pointer_gestures_v1", Version: 3}
	cursorShapeGlobal = fakecomp.Global{Name: 21, Interface: "wp_cursor_shape_manager_v1", Version: 1}
)

// batches waits for the compositor's messages to be handled and returns the
// batches that reached the handler since the last call.
func (e *keyEnv) batches() [][]ui.PointerData {
	e.t.Helper()
	e.settle()
	all := e.h.pointerBatches()
	out := all[e.seenBatches:]
	e.seenBatches = len(all)
	return out
}

func changes(batch []ui.PointerData) []ui.PointerChange {
	out := make([]ui.PointerChange, len(batch))
	for i, d := range batch {
		out[i] = d.Change
	}
	return out
}

func wantChanges(t *testing.T, batch []ui.PointerData, want ...ui.PointerChange) {
	t.Helper()
	got := changes(batch)
	if len(got) != len(want) {
		t.Fatalf("changes = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("changes = %v, want %v", got, want)
		}
	}
}

func TestPointerBatchesAreDeliveredAtFrame(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.srv.PointerEnter(e.surface, 10, 20)
	e.srv.PointerMotion(12, 22)
	e.srv.PointerButton(0x110, true)
	if got := e.batches(); len(got) != 0 {
		t.Fatalf("a batch arrived before the frame: %v", got)
	}
	e.srv.PointerFrame()
	got := e.batches()
	if len(got) != 1 {
		t.Fatalf("%d batches, want 1", len(got))
	}
	wantChanges(t, got[0], ui.PointerChangeAdd, ui.PointerChangeHover, ui.PointerChangeDown)
	if got[0][1].PhysicalX != 12 || got[0][1].PhysicalDeltaY != 2 || got[0][2].Buttons != 1 || got[0][0].ViewID != 0 {
		t.Errorf("batch = %+v", got[0])
	}

	e.srv.PointerButton(0x110, false)
	e.srv.PointerLeave(e.surface)
	e.srv.PointerFrame()
	got = e.batches()
	if len(got) != 1 {
		t.Fatalf("%d batches, want 1", len(got))
	}
	wantChanges(t, got[0], ui.PointerChangeUp, ui.PointerChangeRemove)
}

func TestPointerWithoutFramesDeliversEachEvent(t *testing.T) {
	e := startInput(t, capPointer, 4) // wl_pointer.frame is since version 5
	e.srv.PointerEnter(e.surface, 1, 2)
	e.srv.PointerMotion(3, 4)
	got := e.batches()
	if len(got) != 2 {
		t.Fatalf("%d batches, want 2 (one per event)", len(got))
	}
	wantChanges(t, got[0], ui.PointerChangeAdd)
	wantChanges(t, got[1], ui.PointerChangeHover)
}

func TestPointerWheelAndTouchpadThroughTheCompositor(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.srv.PointerEnter(e.surface, 4, 6)
	e.srv.PointerFrame()
	e.batches()

	// A high-resolution wheel notch.
	e.srv.PointerAxisSource(0)
	e.srv.PointerAxis(0, 15)
	e.srv.PointerAxisValue120(0, 120)
	e.srv.PointerAxisDirection(0, 0)
	e.srv.PointerFrame()
	got := e.batches()
	if len(got) != 1 || len(got[0]) != 1 || got[0][0].SignalKind != ui.PointerSignalKindScroll || got[0][0].ScrollDeltaY != 53 {
		t.Fatalf("wheel: %+v", got)
	}

	// A kinetic touchpad scroll: movement, then the fingers lift.
	e.srv.PointerAxisSource(1)
	e.srv.PointerAxis(0, 4)
	e.srv.PointerFrame()
	got = e.batches()
	if len(got) != 1 {
		t.Fatalf("touchpad: %d batches", len(got))
	}
	wantChanges(t, got[0], ui.PointerChangePanZoomStart, ui.PointerChangePanZoomUpdate)
	e.srv.PointerAxisSource(1)
	e.srv.PointerAxisStop(0)
	e.srv.PointerFrame()
	got = e.batches()
	if len(got) != 1 {
		t.Fatalf("touchpad stop: %d batches", len(got))
	}
	wantChanges(t, got[0], ui.PointerChangePanZoomEnd)
}

func TestPointerPinchAndSwipeThroughTheCompositor(t *testing.T) {
	e := startInput(t, capPointer, 0, gesturesGlobal)
	for _, want := range []string{"zwp_pointer_gestures_v1.get_pinch_gesture", "zwp_pointer_gestures_v1.get_swipe_gesture"} {
		if len(e.srv.RequestsWith(want)) != 1 {
			t.Errorf("%s was not requested once: %v", want, e.srv.Requests())
		}
	}
	e.srv.PointerEnter(e.surface, 4, 6)
	e.srv.PointerFrame()
	e.batches()

	e.srv.PinchBegin(e.surface, 2)
	e.srv.PinchUpdate(3, 1, 1.25, 45)
	e.srv.PinchEnd(false)
	e.srv.SwipeBegin(e.surface, 3)
	e.srv.SwipeUpdate(-2, 0)
	e.srv.SwipeEnd(true)
	var all []ui.PointerData
	for _, batch := range e.batches() {
		all = append(all, batch...)
	}
	wantChanges(t, all,
		ui.PointerChangePanZoomStart, ui.PointerChangePanZoomUpdate, ui.PointerChangePanZoomEnd,
		ui.PointerChangePanZoomStart, ui.PointerChangePanZoomUpdate, ui.PointerChangePanZoomEnd)
	if u := all[1]; u.Scale != 1.25 || u.PanDeltaX != 3 || u.Rotation < 0.78 || u.Rotation > 0.79 {
		t.Errorf("pinch update: %+v", u)
	}
	if u := all[4]; u.PanDeltaX != -2 || u.Scale != 1 {
		t.Errorf("swipe update: %+v", u)
	}
}

func TestTouchThroughTheCompositor(t *testing.T) {
	e := startInput(t, capTouch, 0)
	e.srv.TouchDown(e.surface, 0, 10, 20)
	e.srv.TouchShape(0, 10, 6)
	e.srv.TouchFrame()
	got := e.batches()
	if len(got) != 1 {
		t.Fatalf("%d batches, want 1", len(got))
	}
	wantChanges(t, got[0], ui.PointerChangeAdd, ui.PointerChangeDown)

	e.srv.TouchDown(e.surface, 1, 50, 60)
	e.srv.TouchMotion(0, 11, 21)
	e.srv.TouchFrame()
	got = e.batches()
	wantChanges(t, got[0], ui.PointerChangeAdd, ui.PointerChangeDown, ui.PointerChangeMove)
	if m := got[0][2]; m.PhysicalDeltaX != 1 || m.RadiusMajor != 5 {
		t.Errorf("move: %+v", m)
	}

	e.srv.TouchUp(0)
	e.srv.TouchFrame()
	wantChanges(t, e.batches()[0], ui.PointerChangeUp, ui.PointerChangeRemove)

	// The compositor takes the touch for a gesture of its own.
	e.srv.TouchCancel()
	got = e.batches()
	if len(got) != 1 {
		t.Fatalf("%d batches after the cancel, want 1", len(got))
	}
	wantChanges(t, got[0], ui.PointerChangeCancel, ui.PointerChangeRemove)
	if got[0][0].Device != input.TouchDeviceBase+1 {
		t.Errorf("cancelled device %d, want the second finger", got[0][0].Device)
	}
}

// otherSurface makes a wl_surface that is not a window, and returns its ID.
func (e *keyEnv) otherSurface() uint32 {
	e.t.Helper()
	e.onLoop(func() bool { e.p.sess.compositor.CreateSurface(); return true })
	e.settle()
	id := e.srv.ID("wl_surface")
	if id == e.surface {
		e.t.Fatal("the new surface is the window's")
	}
	return id
}

func TestTouchOnASurfaceThatIsNotOursIsIgnored(t *testing.T) {
	e := startInput(t, capTouch, 0)
	other := e.otherSurface()
	e.srv.TouchDown(other, 0, 1, 1)
	e.srv.TouchFrame()
	if got := e.batches(); len(got) != 0 {
		t.Errorf("batches for a touch outside our windows: %v", got)
	}
}

func TestClosingTheWindowLetsPointerAndTouchGo(t *testing.T) {
	e := startInput(t, capPointer|capTouch, 0)
	e.srv.PointerEnter(e.surface, 4, 6)
	e.srv.PointerButton(0x110, true)
	e.srv.PointerFrame()
	e.srv.TouchDown(e.surface, 3, 8, 8)
	e.srv.TouchFrame()
	e.batches()

	var batches [][]ui.PointerData
	e.onLoop(func() bool {
		before := len(e.h.pointerBatches())
		for _, w := range e.p.windows {
			w.Close()
		}
		batches = e.h.pointerBatches()[before:]
		return true
	})
	var all []ui.PointerData
	for _, b := range batches {
		all = append(all, b...)
	}
	// The pointer (with its button) goes first, then the finger.
	wantChanges(t, all, ui.PointerChangeUp, ui.PointerChangeRemove, ui.PointerChangeCancel, ui.PointerChangeRemove)
	if !all[0].Synthesized {
		t.Error("the release of the button is not marked as synthesized")
	}
}

func TestSeatPointerAndTouchComeAndGo(t *testing.T) {
	e := startInput(t, 8, 0) // a seat with a capability the backend does not know
	if len(e.srv.RequestsWith("wl_seat.get_pointer"))+len(e.srv.RequestsWith("wl_seat.get_touch"))+len(e.srv.RequestsWith("wl_seat.get_keyboard")) != 0 {
		t.Fatalf("devices were requested from an empty seat: %v", e.srv.Requests())
	}
	e.srv.SeatCapabilities(capPointer | capTouch)
	waitUntil(t, func() bool {
		return len(e.srv.RequestsWith("wl_seat.get_pointer")) == 1 && len(e.srv.RequestsWith("wl_seat.get_touch")) == 1
	}, "pointer and touch were not requested")

	e.srv.PointerEnter(e.surface, 1, 1)
	e.srv.PointerFrame()
	e.srv.TouchDown(e.surface, 0, 2, 2)
	e.srv.TouchFrame()
	e.batches()

	// Unplugged: the pointer leaves, the finger is cancelled, and the objects go
	// back to the compositor.
	e.srv.SeatCapabilities(8)
	waitUntil(t, func() bool {
		return len(e.srv.RequestsWith("wl_pointer.release")) == 1 && len(e.srv.RequestsWith("wl_touch.release")) == 1
	}, "the devices were not released")
	var all []ui.PointerData
	for _, b := range e.batches() {
		all = append(all, b...)
	}
	wantChanges(t, all, ui.PointerChangeRemove, ui.PointerChangeCancel, ui.PointerChangeRemove)
}

func TestPointerOnASurfaceThatIsNotOursIsIgnored(t *testing.T) {
	e := startInput(t, capPointer, 0)
	e.srv.PointerEnter(e.otherSurface(), 1, 1)
	e.srv.PointerMotion(2, 2)
	e.srv.PointerFrame()
	if got := e.batches(); len(got) != 0 {
		t.Errorf("batches for a pointer outside our windows: %v", got)
	}
}

// ---- cursors ----

func TestCursorShapeIsSetOnEnterAndOnChange(t *testing.T) {
	e := startInput(t, capPointer, 0, cursorShapeGlobal)
	if len(e.srv.RequestsWith("wp_cursor_shape_manager_v1.get_pointer")) != 1 {
		t.Fatalf("no cursor shape device: %v", e.srv.Requests())
	}
	e.srv.PointerEnter(e.surface, 1, 1)
	e.srv.PointerFrame()
	e.settle()
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("set_shape")) == 1 }, "the shape was not set on enter")
	if got := e.srv.RequestsWith("set_shape")[0]; !strings.HasSuffix(got, "shape=1") { // default
		t.Errorf("set_shape on enter: %s", got)
	}

	tests := []struct {
		cursor platform.MouseCursor
		shape  int
	}{
		{platform.MouseCursorText, 9},
		{platform.MouseCursorClick, 4},
		{platform.MouseCursorGrab, 16},
		{platform.MouseCursorGrabbing, 17},
		{platform.MouseCursorResizeLeftRight, 26},
		{platform.MouseCursorResizeUpLeft, 21},
		{platform.MouseCursorWait, 6},
		{platform.MouseCursorForbidden, 15},
		{platform.MouseCursorBasic, 1},
		{platform.MouseCursorUnspecified, 1},
	}
	for i, tt := range tests {
		e.onLoop(func() bool {
			for _, w := range e.p.windows {
				w.SetCursor(tt.cursor)
			}
			return true
		})
		want := i + 2
		waitUntil(t, func() bool { return len(e.srv.RequestsWith("set_shape")) == want }, "set_shape was not sent for "+tt.cursor.String())
		got := e.srv.RequestsWith("set_shape")[want-1]
		if !strings.HasSuffix(got, "shape="+itoaInt(tt.shape)) {
			t.Errorf("%s: %s, want shape %d", tt.cursor, got, tt.shape)
		}
	}
	// The serial is the one of the enter event.
	if got := e.srv.RequestsWith("set_shape")[0]; !strings.Contains(got, "serial=") {
		t.Errorf("no serial in %s", got)
	}
}

func itoaInt(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoaInt(n/10) + string(rune('0'+n%10))
}

func TestCursorNoneHidesTheCursor(t *testing.T) {
	for _, withShapes := range []bool{true, false} {
		name := "without cursor shapes"
		var extra []fakecomp.Global
		if withShapes {
			name = "with cursor shapes"
			extra = append(extra, cursorShapeGlobal)
		}
		t.Run(name, func(t *testing.T) {
			e := startInput(t, capPointer, 0, extra...)
			e.onLoop(func() bool {
				for _, w := range e.p.windows {
					w.SetCursor(platform.MouseCursorNone)
				}
				return true
			})
			e.srv.PointerEnter(e.surface, 1, 1)
			e.srv.PointerFrame()
			waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 1 }, "set_cursor was not sent")
			if got := e.srv.RequestsWith("wl_pointer.set_cursor")[0]; !strings.Contains(got, "surface=false") {
				t.Errorf("none: %s, want no surface", got)
			}
			if len(e.srv.RequestsWith("set_shape")) != 0 {
				t.Error("a shape was set for none")
			}
		})
	}
}

// writeXCursor writes a cursor file of the XCursor format with one image of w by
// h pixels, all of one color, and the hotspot.
func writeXCursor(t *testing.T, path string, w, h, hotX, hotY uint32) {
	t.Helper()
	le := binary.LittleEndian
	var b []byte
	u32 := func(v uint32) { b = le.AppendUint32(b, v) }
	b = append(b, 'X', 'c', 'u', 'r')
	u32(16)      // header size
	u32(0x10000) // version
	u32(1)       // one table of contents entry
	u32(0xfffd0002)
	u32(24) // nominal size
	u32(28) // position of the chunk
	u32(36) // chunk header size
	u32(0xfffd0002)
	u32(24)
	u32(1) // chunk version
	u32(w)
	u32(h)
	u32(hotX)
	u32(hotY)
	u32(0) // delay
	for range w * h {
		u32(0xff6750a4)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// useTheme makes libwayland-cursor load the theme at dir.
func useTheme(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("XCURSOR_PATH", dir)
	t.Setenv("XCURSOR_THEME", "glutter-test")
	t.Setenv("XCURSOR_SIZE", "24")
}

func TestCursorFromTheThemeWhenThereAreNoShapes(t *testing.T) {
	if _, err := loadWaylandCursor(); err != nil {
		t.Skipf("libwayland-cursor is not available: %v", err)
	}
	dir := t.TempDir()
	// The theme has the old X names only.
	writeXCursor(t, filepath.Join(dir, "glutter-test", "cursors", "left_ptr"), 8, 8, 1, 2)
	writeXCursor(t, filepath.Join(dir, "glutter-test", "cursors", "xterm"), 6, 12, 3, 6)
	useTheme(t, dir)

	e := startInput(t, capPointer, 0)
	e.srv.PointerEnter(e.surface, 1, 1)
	e.srv.PointerFrame()
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 1 }, "the cursor was not set on enter")
	if got := e.srv.RequestsWith("wl_pointer.set_cursor")[0]; !strings.Contains(got, "surface=true") || !strings.HasSuffix(got, "hotspot=1,2") {
		t.Errorf("default cursor: %s", got)
	}
	// The cursor surface got the image of the theme.
	if got := e.srv.RequestsWith("wl_surface.commit buffer="); len(got) == 0 || !strings.HasSuffix(got[len(got)-1], "8x8") {
		t.Errorf("commits: %v", got)
	}

	e.onLoop(func() bool {
		for _, w := range e.p.windows {
			w.SetCursor(platform.MouseCursorText)
		}
		return true
	})
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 2 }, "the text cursor was not set")
	if got := e.srv.RequestsWith("wl_pointer.set_cursor")[1]; !strings.HasSuffix(got, "hotspot=3,6") {
		t.Errorf("text cursor: %s", got)
	}
	commits := e.srv.RequestsWith("wl_surface.commit buffer=")
	if !strings.HasSuffix(commits[len(commits)-1], "6x12") {
		t.Errorf("last commit: %s", commits[len(commits)-1])
	}

	// A cursor the theme does not have is reported and leaves the last one in
	// place.
	e.onLoop(func() bool {
		for _, w := range e.p.windows {
			w.SetCursor(platform.MouseCursorZoomIn)
		}
		return true
	})
	e.settle()
	if n := len(e.srv.RequestsWith("wl_pointer.set_cursor")); n != 2 {
		t.Errorf("%d set_cursor requests, want 2: a missing cursor must not be set", n)
	}
}

func TestCursorFailures(t *testing.T) {
	if _, err := loadWaylandCursor(); err != nil {
		t.Skipf("libwayland-cursor is not available: %v", err)
	}
	dir := t.TempDir()
	writeXCursor(t, filepath.Join(dir, "glutter-test", "cursors", "left_ptr"), 4, 4, 0, 0)
	useTheme(t, dir)
	e := startInput(t, capPointer, 0)

	// The theme has no such cursor.
	err := e.onLoopErr(func() error {
		_, _, err := e.p.cursors.find(platform.MouseCursorZoomOut, 1)
		return err
	})
	assertRoute(t, err, "cursors.find.CursorNotFound")
	if appErr, _ := errors.FromError(err); appErr.Metadata["cursor"] != "MouseCursorZoomOut" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}

	// The library cannot be loaded, or the theme cannot be: reported once.
	err = e.onLoopErr(func() error {
		c := newCursors(e.p)
		c.api = nil
		c.load = func() (*waylandCursor, error) {
			return nil, errors.New("LibraryLoadFailed", "x", "names", []string{"libwayland-cursor.so.0"}, "dlerror", "missing")
		}
		_, _, err := c.find(platform.MouseCursorBasic, 1)
		if !c.failed {
			t.Error("a failed load is retried")
		}
		return err
	})
	assertRoute(t, err, "cursors.find->cursors.loadTheme.CursorLibraryFailed")

	err = e.onLoopErr(func() error {
		c := newCursors(e.p)
		c.api = &waylandCursor{themeLoad: func(uintptr, int32, uintptr) uintptr { return 0 }}
		_, _, err := c.find(platform.MouseCursorBasic, 1)
		return err
	})
	assertRoute(t, err, "cursors.find->cursors.loadTheme.CursorThemeFailed")
}

func TestCursorThemeSizeFromTheEnvironment(t *testing.T) {
	if _, err := loadWaylandCursor(); err != nil {
		t.Skipf("libwayland-cursor is not available: %v", err)
	}
	e := startInput(t, capPointer, 0)
	tests := []struct {
		env  map[string]string
		want int32
		name string
	}{
		{map[string]string{}, 24, "default"},
		{map[string]string{"XCURSOR_SIZE": "48"}, 48, "XCURSOR_SIZE"},
		{map[string]string{"XCURSOR_SIZE": "big"}, 24, "not a number"},
		{map[string]string{"XCURSOR_SIZE": "-3"}, 24, "negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSize int32
			var gotName string
			e.onLoop(func() bool {
				c := newCursors(e.p)
				c.getenv = func(k string) string { return tt.env[k] }
				c.api = &waylandCursor{themeLoad: func(name uintptr, size int32, shm uintptr) uintptr {
					gotSize, gotName = size, dl.GoString(name)
					return 0
				}}
				_, _ = c.loadTheme(1) // fails: the fake returns no theme
				return true
			})
			if gotSize != tt.want || gotName != "" {
				t.Errorf("size %d, theme %q; want %d and the default theme", gotSize, gotName, tt.want)
			}
		})
	}
	var gotName string
	e.onLoop(func() bool {
		c := newCursors(e.p)
		c.getenv = func(k string) string {
			if k == "XCURSOR_THEME" {
				return "Adwaita"
			}
			return ""
		}
		c.api = &waylandCursor{themeLoad: func(name uintptr, size int32, shm uintptr) uintptr {
			gotName = dl.GoString(name)
			return 0
		}}
		_, _ = c.loadTheme(1)
		return true
	})
	if gotName != "Adwaita" {
		t.Errorf("theme = %q, want Adwaita", gotName)
	}
}

func TestCursorShapesCoverEveryCursor(t *testing.T) {
	for c := platform.MouseCursorUnspecified; c.Valid(); c++ {
		if c == platform.MouseCursorNone {
			continue // hidden: there is no shape
		}
		if _, ok := cursorShapes[c]; !ok {
			t.Errorf("%s has no wp_cursor_shape_v1 shape", c)
		}
	}
	if _, ok := cursorShapes[platform.MouseCursorNone]; ok {
		t.Error("none has a shape")
	}
}

// assertRoute checks that err is an *errors.Error whose route contains route.
func assertRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

// recordThemeLoads makes the cursors load the theme through a wrapper that records
// the name and the size of every wl_cursor_theme_load. Call it before the pointer
// enters.
func recordThemeLoads(t *testing.T, e *keyEnv) func() []string {
	t.Helper()
	var mu sync.Mutex
	var loads []string
	e.onLoop(func() bool {
		e.p.cursors.load = func() (*waylandCursor, error) {
			real, err := loadWaylandCursor()
			if err != nil {
				return nil, err
			}
			wrapped := *real
			wrapped.themeLoad = func(name uintptr, size int32, shm uintptr) uintptr {
				mu.Lock()
				loads = append(loads, fmt.Sprintf("%s@%d", dl.GoString(name), size))
				mu.Unlock()
				return real.themeLoad(name, size, shm)
			}
			return &wrapped, nil
		}
		return true
	})
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), loads...)
	}
}

func TestCursorFromTheThemeIsLoadedAtTheScaleOfTheWindow(t *testing.T) {
	if _, err := loadWaylandCursor(); err != nil {
		t.Skipf("libwayland-cursor is not available: %v", err)
	}
	dir := t.TempDir()
	writeXCursor(t, filepath.Join(dir, "glutter-test", "cursors", "left_ptr"), 48, 48, 8, 12)
	useTheme(t, dir)

	e := startKeyboardNoFocus(t, fakecomp.Config{Capabilities: capPointer, PreferredBufferScale: 2})
	loads := recordThemeLoads(t, e)
	e.srv.PointerEnter(e.surface, 1, 1)
	e.srv.PointerFrame()
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 1 }, "the cursor was not set on enter")
	// At scale 2 the theme is loaded at twice the size, the cursor surface has buffer
	// scale 2 and the hotspot is in surface coordinates.
	if got := loads(); len(got) != 1 || got[0] != "glutter-test@48" {
		t.Errorf("theme loads = %v, want glutter-test@48", got)
	}
	if got := e.srv.RequestsWith("wl_pointer.set_cursor")[0]; !strings.HasSuffix(got, "hotspot=4,6") {
		t.Errorf("set_cursor: %s, want the hotspot halved", got)
	}
	if got := e.srv.RequestsWith("wl_surface.set_buffer_scale 2"); len(got) < 2 { // the window's and the cursor's
		t.Errorf("buffer scale requests = %v, want one for the cursor surface", got)
	}

	// The window gets the scale 1: the cursor is set again from a theme of size 24.
	e.srv.SendPreferredBufferScale(1)
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 2 }, "the cursor was not set again for the new scale")
	if got := loads(); len(got) != 2 || got[1] != "glutter-test@24" {
		t.Errorf("theme loads = %v, want a second one at 24", got)
	}
	if got := e.srv.RequestsWith("wl_pointer.set_cursor")[1]; !strings.HasSuffix(got, "hotspot=8,12") {
		t.Errorf("set_cursor: %s, want the whole hotspot", got)
	}
}

func TestCursorThemeAndSizeFromTheSettings(t *testing.T) {
	if _, err := loadWaylandCursor(); err != nil {
		t.Skipf("libwayland-cursor is not available: %v", err)
	}
	dir := t.TempDir()
	writeXCursor(t, filepath.Join(dir, "glutter-test", "cursors", "left_ptr"), 8, 8, 1, 2)
	writeXCursor(t, filepath.Join(dir, "other", "cursors", "left_ptr"), 16, 16, 3, 4)
	useTheme(t, dir)

	e := startInput(t, capPointer, 0)
	loads := recordThemeLoads(t, e)
	e.srv.PointerEnter(e.surface, 1, 1)
	e.srv.PointerFrame()
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 1 }, "the cursor was not set on enter")

	// The settings win over the environment, and the cursor over the pointer changes.
	e.onLoop(func() bool { e.p.cursors.configure("other", 36); return true })
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 2 }, "the cursor did not change with the settings")
	if got := loads(); len(got) != 2 || got[0] != "glutter-test@24" || got[1] != "other@36" {
		t.Errorf("theme loads = %v", got)
	}
	if got := e.srv.RequestsWith("wl_pointer.set_cursor")[1]; !strings.HasSuffix(got, "hotspot=3,4") {
		t.Errorf("set_cursor: %s, want the cursor of the other theme", got)
	}

	// The same settings change nothing; empty settings go back to the environment.
	e.onLoop(func() bool { e.p.cursors.configure("other", 36); return true })
	e.settle()
	if n := len(e.srv.RequestsWith("wl_pointer.set_cursor")); n != 2 {
		t.Errorf("%d set_cursor requests after the same settings, want 2", n)
	}
	e.onLoop(func() bool { e.p.cursors.configure("", 0); return true })
	waitUntil(t, func() bool { return len(e.srv.RequestsWith("wl_pointer.set_cursor")) == 3 }, "the cursor did not go back")
	if got := loads(); len(got) != 3 || got[2] != "glutter-test@24" {
		t.Errorf("theme loads = %v", got)
	}
}

func TestEveryCursorHasItsShape(t *testing.T) {
	// The values of wp_cursor_shape_device_v1.shape.
	tests := []struct {
		cursor platform.MouseCursor
		shape  wl.WpCursorShapeDeviceV1Shape
	}{
		{platform.MouseCursorUnspecified, 1}, {platform.MouseCursorBasic, 1}, {platform.MouseCursorContextMenu, 2},
		{platform.MouseCursorHelp, 3}, {platform.MouseCursorClick, 4}, {platform.MouseCursorProgress, 5},
		{platform.MouseCursorWait, 6}, {platform.MouseCursorCell, 7}, {platform.MouseCursorPrecise, 8},
		{platform.MouseCursorText, 9}, {platform.MouseCursorVerticalText, 10}, {platform.MouseCursorAlias, 11},
		{platform.MouseCursorCopy, 12}, {platform.MouseCursorMove, 13}, {platform.MouseCursorNoDrop, 14},
		{platform.MouseCursorForbidden, 15}, {platform.MouseCursorGrab, 16}, {platform.MouseCursorGrabbing, 17},
		{platform.MouseCursorResizeRight, 18}, {platform.MouseCursorResizeUp, 19}, {platform.MouseCursorResizeUpRight, 20},
		{platform.MouseCursorResizeUpLeft, 21}, {platform.MouseCursorResizeDown, 22}, {platform.MouseCursorResizeDownRight, 23},
		{platform.MouseCursorResizeDownLeft, 24}, {platform.MouseCursorResizeLeft, 25}, {platform.MouseCursorResizeLeftRight, 26},
		{platform.MouseCursorResizeUpDown, 27}, {platform.MouseCursorResizeUpRightDownLeft, 28},
		{platform.MouseCursorResizeUpLeftDownRight, 29}, {platform.MouseCursorResizeColumn, 30}, {platform.MouseCursorResizeRow, 31},
		{platform.MouseCursorAllScroll, 32}, {platform.MouseCursorZoomIn, 33}, {platform.MouseCursorZoomOut, 34},
		{platform.MouseCursorDisappearing, 1},
	}
	covered := map[platform.MouseCursor]bool{platform.MouseCursorNone: true}
	for _, tt := range tests {
		if got := cursorShapes[tt.cursor]; got != tt.shape {
			t.Errorf("%s has shape %d, want %d", tt.cursor, got, tt.shape)
		}
		covered[tt.cursor] = true
	}
	for c := platform.MouseCursorUnspecified; c.Valid(); c++ {
		if !covered[c] {
			t.Errorf("%s is not in the table", c)
		}
	}
}
