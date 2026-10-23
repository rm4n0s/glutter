//go:build linux && !android

package xkb

import (
	"path/filepath"
	"testing"
)

// withFakeX11 makes the X11 functions of the package use lib for the test.
func withFakeX11(t *testing.T, lib *x11Library) {
	t.Helper()
	old := loadX11
	loadX11 = func() (*x11Library, error) { return lib, nil }
	t.Cleanup(func() { loadX11 = old })
}

func TestKeymapFromX11Failures(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	good := func() *x11Library {
		return &x11Library{
			setupExtension: func(uintptr, uint16, uint16, int32, uintptr, uintptr, uintptr, uintptr) int32 { return 1 },
			coreKeyboardID: func(uintptr) int32 { return 3 },
			keymapNewDevice: func(ctx, conn uintptr, device, flags int32) uintptr {
				return 0 // overridden by the cases that need a keymap
			},
			stateNewDevice: func(uintptr, uintptr, int32) uintptr { return 0 },
		}
	}
	tests := []struct {
		name  string
		tweak func(*x11Library)
		route string
	}{
		{"the server has no XKB extension", func(l *x11Library) {
			l.setupExtension = func(uintptr, uint16, uint16, int32, uintptr, uintptr, uintptr, uintptr) int32 { return 0 }
		}, "Context.KeymapFromX11.XKBX11SetupFailed"},
		{"the server has no core keyboard", func(l *x11Library) { l.coreKeyboardID = func(uintptr) int32 { return -1 } },
			"Context.KeymapFromX11.XKBX11SetupFailed"},
		{"the keymap cannot be read", func(*x11Library) {}, "Context.KeymapFromX11->Context.newKeymap.XKBKeymapFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := good()
			tt.tweak(lib)
			withFakeX11(t, lib)
			_, _, err := ctx.KeymapFromX11(1)
			assertRoute(t, err, tt.route)
		})
	}

	t.Run("the keyboard state cannot be read", func(t *testing.T) {
		km, err := ctx.KeymapFromNames(Names{Layout: "us"})
		if err != nil {
			t.Fatal(err)
		}
		defer km.Close()
		withFakeX11(t, good())
		_, err = km.NewStateFromX11(1, 3)
		assertRoute(t, err, "Keymap.NewStateFromX11.XKBStateFailed")
	})
}

func TestX11LibraryLoadFailure(t *testing.T) {
	t.Setenv(X11LibraryEnv, filepath.Join(t.TempDir(), "missing", "libxkbcommon-x11.so.0"))
	_, err := loadX11Library()
	assertRoute(t, err, "loadX11Library.XKBLoadFailed")
}

func TestX11CallsReachTheLibrary(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	km, err := ctx.KeymapFromNames(Names{Layout: "us"})
	if err != nil {
		t.Fatal(err)
	}
	defer km.Close()
	var gotConn uintptr
	var gotDevice int32
	lib := &x11Library{
		setupExtension: func(conn uintptr, major, minor uint16, flags int32, a, b, c, d uintptr) int32 {
			gotConn = conn
			if major != 1 || minor != 0 || flags != 0 || a|b|c|d != 0 {
				t.Errorf("setup called with version %d.%d flags %d", major, minor, flags)
			}
			return 1
		},
		coreKeyboardID: func(uintptr) int32 { return 3 },
		keymapNewDevice: func(ctxPtr, conn uintptr, device, flags int32) uintptr {
			// Hand back a real keymap, made by the same library.
			gotDevice = device
			return ctx.lib.keymapNewFromNames(ctx.ptr, 0, 0)
		},
		stateNewDevice: func(keymap, conn uintptr, device int32) uintptr {
			gotDevice = device
			return km.lib.stateNew(keymap)
		},
	}
	withFakeX11(t, lib)
	k, device, err := ctx.KeymapFromX11(0xC011)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	if gotConn != 0xC011 || device != 3 || gotDevice != 3 {
		t.Errorf("connection 0x%x, device %d: the arguments did not reach the library", gotConn, device)
	}
	state, err := k.NewStateFromX11(0xC011, device)
	if err != nil {
		t.Fatal(err)
	}
	state.Close()
}

func TestUpdateMaskGroups(t *testing.T) {
	l := newLayout(t, Names{Layout: "us,ru"})
	key := uint32(keyA)
	// The locked group is the second layout; so is a base group.
	l.UpdateMaskGroups(0, 0, 0, 0, 0, 1)
	if k, err := l.Lookup(key); err != nil || k.Character != "ф" {
		t.Errorf("locked group 1: %+v, %v", k, err)
	}
	l.UpdateMaskGroups(0, 0, 0, 0, 0, 0)
	if k, err := l.Lookup(key); err != nil || k.Character != "a" {
		t.Errorf("group 0: %+v, %v", k, err)
	}
	l.UpdateMaskGroups(0, 0, 0, 1, 0, 0)
	if k, err := l.Lookup(key); err != nil || k.Character != "ф" {
		t.Errorf("base group 1: %+v, %v", k, err)
	}
}
