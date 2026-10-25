//go:build windows

package windows

import (
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the glue of the keyboard of a window (SPEC §25.6): it reads the key messages and
// gives them to the plain-Go manager (keymanager.go, keyboard.go), and answers Windows what the
// manager says.

// winKeyHost is the keyHost of the platform: the calls of user32 for the keyboard layout of the
// UI thread.
type winKeyHost struct{}

var _ keyHost = winKeyHost{}

func (winKeyHost) KeyState(vk uint32) int16 { return win32.GetKeyState(vk) }

func (winKeyHost) ScanCode(vk uint32, extended bool) uint32 {
	if extended {
		return win32.MapVirtualKey(vk, win32.MapVKVKToVSCEx)
	}
	return win32.MapVirtualKey(vk, win32.MapVKVKToVSC)
}

func (winKeyHost) VirtualKeyOfScan(scan uint32) uint32 {
	return win32.MapVirtualKey(scan, win32.MapVKVSCToVKEx)
}

func (winKeyHost) Layout() uintptr { return win32.GetKeyboardLayout(0) }

func (winKeyHost) CharOfKey(scan uint32) uint32 {
	hkl := win32.GetKeyboardLayout(0)
	vk := win32.MapVirtualKeyEx(scan, win32.MapVKVSCToVKEx, hkl)
	if vk == 0 {
		return 0
	}
	return win32.MapVirtualKeyEx(vk, win32.MapVKVKToChar, hkl)
}

// winKeyQueue reads the key messages of one window.
type winKeyQueue struct{ hwnd uintptr }

var _ keyQueue = winKeyQueue{}

func (q winKeyQueue) Peek() (keyMsg, bool) {
	var m win32.Msg
	if !win32.PeekMessage(&m, q.hwnd, win32.WMKeyFirst, win32.WMKeyLast, win32.PMNoRemove) {
		return keyMsg{}, false
	}
	return keyMsg{Message: m.Message, WParam: m.WParam, LParam: m.LParam, Time: m.Time}, true
}

func (q winKeyQueue) Take() {
	var m win32.Msg
	win32.PeekMessage(&m, q.hwnd, win32.WMKeyFirst, win32.WMKeyLast, win32.PMRemove)
}

// sendKey gives key data to the framework at once and reports whether it handled it. A handler
// that cannot answer at once gets the data as it comes, and the key counts as not handled.
func (p *Platform) sendKey(d ui.KeyData) bool {
	if p.handler == nil {
		return false
	}
	if s, ok := p.handler.(platform.SyncInputHandler); ok {
		return s.OnKeySync(d)
	}
	p.handler.OnKey(d)
	return false
}

// onKey handles the key messages of the window. It returns the answer of the window procedure.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_KEYDOWN, WM_SYSKEYDOWN, WM_KEYUP, WM_SYSKEYUP)
func (w *window) onKey(msg uint32, wparam, lparam uintptr) (uintptr, bool) {
	if w.p.handler == nil {
		return 0, false
	}
	switch w.p.keys.handleKey(winKeyQueue{w.hwnd}, msg, wparam, lparam, win32.GetMessageTime()) {
	case keyConsumed:
		return 0, true
	}
	return 0, false
}

// onChar handles the character messages that reach the window procedure.
//
// Flutter: shell/platform/windows/flutter_window.cc FlutterWindow::HandleMessage (WM_CHAR, WM_SYSCHAR, WM_DEADCHAR)
func (w *window) onChar(msg uint32) (uintptr, bool) {
	if w.p.handler != nil && w.p.keys.handleChar(msg) == keyConsumed {
		return 0, true
	}
	return 0, false
}

// runDevKey does one scripted key step with SendInput, the way a keyboard does: the keys arrive as
// WM_KEYDOWN and WM_KEYUP at the window that has the focus, the first window here, which is put
// in front, and TranslateMessage makes their characters. The key state that Windows keeps (Shift
// held) is the real one, so the characters and GetKeyState are what the user's keys would give.
//
// Flutter: no counterpart
func (p *Platform) runDevKey(ev DevEvent) {
	w := p.devWindow()
	if w == nil {
		p.log.Warn("a scripted event found no window", "step", ev.Kind)
		return
	}
	win32.SetForegroundWindow(w.hwnd)
	win32.SetFocus(w.hwnd)
	var inputs []win32.KeyInput
	add := func(vk uint32, up bool) {
		in := win32.KeyInput{Type: win32.InputKeyboard, Key: win32.KeybdInput{
			VK: uint16(vk), Scan: uint16(win32.MapVirtualKey(vk, win32.MapVKVKToVSC)),
		}}
		if isExtendedKey(vk) {
			in.Key.Flags |= win32.KeyEventFExtendedKey
		}
		if up {
			in.Key.Flags |= win32.KeyEventFKeyUp
		}
		inputs = append(inputs, in)
	}
	switch ev.Kind {
	case "key":
		for _, vk := range ev.Keys {
			add(vk, false)
		}
		for i := len(ev.Keys) - 1; i >= 0; i-- {
			add(ev.Keys[i], true)
		}
	case "keydown":
		add(ev.Keys[0], false)
	case "keyup":
		add(ev.Keys[0], true)
	}
	if n := win32.SendKeyInput(inputs); int(n) != len(inputs) {
		p.log.Warn("a scripted key was not sent", "step", ev.Kind, "sent", n, "of", len(inputs))
	}
}
