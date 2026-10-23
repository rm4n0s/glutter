//go:build linux

package xkb

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform/linux/dl"

	"github.com/rm4n0s/errors"
)

// X11LibraryEnv names the environment variable that overrides the path of
// libxkbcommon-x11.
const X11LibraryEnv = "GLUTTER_XKBCOMMON_X11_LIBRARY"

// x11Library holds the bound functions of libxkbcommon-x11.
type x11Library struct {
	setupExtension  func(conn uintptr, major, minor uint16, flags int32, majorOut, minorOut, eventOut, errorOut uintptr) int32
	coreKeyboardID  func(conn uintptr) int32
	keymapNewDevice func(ctx, conn uintptr, device, flags int32) uintptr
	stateNewDevice  func(keymap, conn uintptr, device int32) uintptr
}

// loadX11Library opens libxkbcommon-x11.so.0 and binds the functions. It fails
// with XKBLoadFailed, or with SymbolMissing for a function that is absent.
func loadX11Library() (*x11Library, error) {
	lib, err := dl.OpenOverride(X11LibraryEnv, "libxkbcommon-x11.so.0", "libxkbcommon-x11.so")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New(tagXKBLoadFailed, "could not load libxkbcommon-x11",
			"names", md["names"], "dlerror", md["dlerror"])
	}
	l := new(x11Library)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.setupExtension, "xkb_x11_setup_xkb_extension"},
		{&l.coreKeyboardID, "xkb_x11_get_core_keyboard_device_id"},
		{&l.keymapNewDevice, "xkb_x11_keymap_new_from_device"},
		{&l.stateNewDevice, "xkb_x11_state_new_from_device"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// loadX11 opens libxkbcommon-x11 once; tests replace it with a fake.
var loadX11 = sync.OnceValues(loadX11Library)

// KeymapFromX11 reads the keymap of the X server's core keyboard through the XKB
// extension, on the xcb connection (Xlib's, from XGetXCBConnection). It returns
// the keymap and the device id that NewStateFromX11 needs. It fails with
// XKBLoadFailed when libxkbcommon-x11 is missing, XKBX11SetupFailed when the
// server has no XKB extension or no core keyboard, and XKBKeymapFailed when the
// keymap cannot be read.
//
// Flutter: no counterpart
func (c *Context) KeymapFromX11(conn uintptr) (*Keymap, int32, error) {
	x11, err := loadX11()
	if err != nil {
		return nil, 0, err
	}
	// Version 1.0 of the extension is what libxkbcommon-x11 needs.
	if x11.setupExtension(conn, 1, 0, 0, 0, 0, 0, 0) == 0 {
		return nil, 0, errors.New(tagXKBX11SetupFailed, "the X server has no usable XKB extension")
	}
	device := x11.coreKeyboardID(conn)
	if device == -1 {
		return nil, 0, errors.New(tagXKBX11SetupFailed, "the X server has no core keyboard")
	}
	km, err := c.newKeymap(x11.keymapNewDevice(c.ptr, conn, device, 0), "x11")
	if err != nil {
		return nil, 0, err
	}
	return km, device, nil
}

// NewStateFromX11 creates the state of the X server's keyboard, with the
// modifiers and layout that are active now. It fails with XKBStateFailed.
//
// Flutter: no counterpart
func (k *Keymap) NewStateFromX11(conn uintptr, device int32) (*State, error) {
	x11, err := loadX11()
	if err != nil {
		return nil, err
	}
	ptr := x11.stateNewDevice(k.ptr, conn, device)
	if ptr == 0 {
		return nil, errors.New(tagXKBStateFailed, "xkb_x11_state_new_from_device failed")
	}
	return &State{lib: k.lib, ptr: ptr, scratch: dl.Calloc(1, scratchSize)}, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagXKBStateFailed    = "XKBStateFailed"
	tagXKBX11SetupFailed = "XKBX11SetupFailed"
)
