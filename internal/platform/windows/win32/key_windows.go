//go:build windows

package win32

import (
	"unsafe"
)

// The bindings of user32.dll for the keyboard of step 28.4 (SPEC §25.6): the key state, the
// translation of scan codes and virtual keys, the keyboard layouts, the key messages in the queue
// and the keyboard input that the tests send. A function whose failure the caller can only skip
// answers 0 or false.

// GetKeyState is the state of a virtual key as the messages that the thread has handled say: the
// high bit is set while the key is down and the low bit while a lock key is toggled on (the value is
// the SHORT of the C function).
//
// Flutter: no counterpart (shell/platform/windows/flutter_windows_engine.cc, which passes ::GetKeyState)
func GetKeyState(vk uint32) int16 {
	r, _, err := user32.call("GetKeyState", uintptr(vk))
	if err != nil {
		return 0
	}
	return int16(r)
}

// MapVirtualKey translates between virtual keys, scan codes and characters (the MAPVK_ constants
// give the direction) for the keyboard layout of the thread; it returns 0 when there is no
// translation. For MapVKVKToChar the high bit of the result is set when the key is a dead key.
//
// Flutter: no counterpart (shell/platform/windows/flutter_window.cc FlutterWindow::Win32MapVkToChar)
func MapVirtualKey(code, mapType uint32) uint32 {
	r, _, err := user32.call("MapVirtualKeyW", uintptr(code), uintptr(mapType))
	if err != nil {
		return 0
	}
	return uint32(r)
}

// MapVirtualKeyEx is MapVirtualKey for the keyboard layout hkl.
//
// Flutter: no counterpart
func MapVirtualKeyEx(code, mapType uint32, hkl uintptr) uint32 {
	r, _, err := user32.call("MapVirtualKeyExW", uintptr(code), uintptr(mapType), hkl)
	if err != nil {
		return 0
	}
	return uint32(r)
}

// GetKeyboardLayout is the handle of the keyboard layout of a thread (0 for the calling thread).
//
// Flutter: no counterpart
func GetKeyboardLayout(thread uint32) uintptr {
	r, _, err := user32.call("GetKeyboardLayout", uintptr(thread))
	if err != nil {
		return 0
	}
	return r
}

// LoadKeyboardLayout loads a keyboard layout by its name, the eight hexadecimal digits of its
// identifier such as "00000407" (German), and returns its handle, 0 when it cannot be loaded. With
// KLFActivate it becomes the layout of the thread.
//
// Flutter: no counterpart
func LoadKeyboardLayout(name string, flags uint32) uintptr {
	p, err := UTF16FromString(name)
	if err != nil {
		return 0
	}
	r, _, err := user32.call("LoadKeyboardLayoutW", uintptr(unsafe.Pointer(&p[0])), uintptr(flags))
	if err != nil {
		return 0
	}
	return r
}

// ActivateKeyboardLayout makes a layout the one of the thread and returns the layout it had.
//
// Flutter: no counterpart
func ActivateKeyboardLayout(hkl uintptr, flags uint32) uintptr {
	r, _, err := user32.call("ActivateKeyboardLayout", hkl, uintptr(flags))
	if err != nil {
		return 0
	}
	return r
}

// GetMessageTime is the time of the last message the thread took from its queue, in milliseconds.
//
// Flutter: no counterpart
func GetMessageTime() uint32 {
	r, _, err := user32.call("GetMessageTime")
	if err != nil {
		return 0
	}
	return uint32(r)
}

// SetFocus gives the keyboard focus to a window of the calling thread and returns the window that
// had it.
//
// Flutter: no counterpart
func SetFocus(hwnd uintptr) uintptr {
	r, _, err := user32.call("SetFocus", hwnd)
	if err != nil {
		return 0
	}
	return r
}

// GetFocus is the window of the calling thread that has the keyboard focus, or 0.
//
// Flutter: no counterpart
func GetFocus() uintptr {
	r, _, err := user32.call("GetFocus")
	if err != nil {
		return 0
	}
	return r
}

// SendKeyInput puts keyboard input into the system's input queue as a keyboard does, and returns
// how many inputs it inserted. The scripted events and the tests use it; the keys arrive as
// WM_KEYDOWN and WM_KEYUP at the window that has the focus, with the character messages that
// TranslateMessage makes of them.
//
// Flutter: no counterpart
func SendKeyInput(inputs []KeyInput) uint32 {
	if len(inputs) == 0 {
		return 0
	}
	r, _, err := user32.call("SendInput", uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(KeyInput{}))
	if err != nil {
		return 0
	}
	return uint32(r)
}

// GetKeyboardState copies the key states of the calling thread (what GetKeyState reads) into a
// table of 256 bytes, one for each virtual key: the high bit says down, the low bit toggled.
//
// Flutter: no counterpart
func GetKeyboardState() ([256]byte, bool) {
	var state [256]byte
	r, _, err := user32.call("GetKeyboardState", uintptr(unsafe.Pointer(&state[0])))
	return state, err == nil && r != 0
}

// SetKeyboardState replaces the key states of the calling thread. The tests use it to make a lock
// key change without a key message, as it does when the window has no focus.
//
// Flutter: no counterpart
func SetKeyboardState(state [256]byte) bool {
	r, _, err := user32.call("SetKeyboardState", uintptr(unsafe.Pointer(&state[0])))
	return err == nil && r != 0
}
