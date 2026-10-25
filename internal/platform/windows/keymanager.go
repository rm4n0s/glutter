package windows

import (
	"time"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the key messages of a window (SPEC §25.6): a port of Flutter's
// KeyboardManager. A WM_KEYDOWN or WM_SYSKEYDOWN takes the character messages that
// TranslateMessage queued right after it, the converter (keyEmbedder) makes the key data, and the
// framework answers at once, which tells the window procedure what to answer Windows.

// keyMsg is a message of the key range in the queue of a window.
type keyMsg struct {
	Message uint32
	WParam  uintptr
	LParam  uintptr
	// Time is the time of the message in milliseconds, as GetMessageTime reports it.
	Time uint32
}

// keyQueue is the queue of key messages of a window, to read the character messages that follow a
// key.
//
// Flutter: shell/platform/windows/keyboard_manager.h KeyboardManager::WindowDelegate::Win32PeekMessage
type keyQueue interface {
	// Peek returns the next message of the key range (WM_KEYFIRST to WM_KEYLAST) of the window
	// without taking it.
	Peek() (keyMsg, bool)
	// Take removes that message from the queue.
	Take()
}

// keyAnswer is what the window procedure answers to Windows for a key message.
type keyAnswer int

const (
	// keyToSystem passes the message to DefWindowProc.
	keyToSystem keyAnswer = iota
	// keyConsumed returns 0: the message is handled.
	keyConsumed
)

// keyboardManager handles the key messages of the windows of the platform, which share one
// keyboard.
//
// Flutter: shell/platform/windows/keyboard_manager.cc KeyboardManager
type keyboardManager struct {
	host keyHost
	emb  *keyEmbedder
	// send gives key data to the framework and reports whether it handled it.
	send func(ui.KeyData) bool

	// fakeCtrl says that the left Control press that comes with AltGr was dropped, and its key up
	// is dropped as well.
	fakeCtrl bool
	// sysDownHandled says the framework handled the last system key down, so the WM_SYSCHAR
	// that follows it is consumed.
	sysDownHandled bool
}

// newKeyboardManager makes the manager of a platform. send gives the key data to the framework.
//
// Flutter: shell/platform/windows/keyboard_manager.cc KeyboardManager::KeyboardManager
func newKeyboardManager(host keyHost, now func() time.Duration, send func(ui.KeyData) bool) *keyboardManager {
	m := &keyboardManager{host: host, send: send}
	m.emb = newKeyEmbedder(maskedKeyHost{m}, now)
	return m
}

// maskedKeyHost is the host the converter sees: the same as the manager's, except that the state of
// the left Control key is "up" while the Control press of AltGr is dropped, or the synchronization
// of the pressed keys would press the key again.
type maskedKeyHost struct{ m *keyboardManager }

func (h maskedKeyHost) KeyState(vk uint32) int16 {
	s := h.m.host.KeyState(vk)
	if vk == win32.VKLControl && h.m.fakeCtrl {
		if s&stateMaskPressed == 0 {
			h.m.fakeCtrl = false // Windows has released the key: nothing to hide any more
		}
		s &^= stateMaskPressed
	}
	return s
}
func (h maskedKeyHost) ScanCode(vk uint32, extended bool) uint32 {
	return h.m.host.ScanCode(vk, extended)
}
func (h maskedKeyHost) VirtualKeyOfScan(scan uint32) uint32 { return h.m.host.VirtualKeyOfScan(scan) }
func (h maskedKeyHost) Layout() uintptr                     { return h.m.host.Layout() }
func (h maskedKeyHost) CharOfKey(scan uint32) uint32        { return h.m.host.CharOfKey(scan) }

// resolveKeyCode gives the key code of a modifier its side. VK_SHIFT and VK_LSHIFT are told by the
// scan code, VK_MENU and VK_CONTROL by the extended flag.
//
// Flutter: shell/platform/windows/keyboard_manager.cc ResolveKeyCode
func (m *keyboardManager) resolveKeyCode(original uint32, extended bool, scan uint32) uint32 {
	switch original {
	case win32.VKShift, win32.VKLShift:
		if vk := m.host.VirtualKeyOfScan(scan); vk != 0 {
			return vk
		}
		return original
	case win32.VKMenu, win32.VKLMenu:
		if extended {
			return win32.VKRMenu
		}
		return win32.VKLMenu
	case win32.VKControl, win32.VKLControl:
		if extended {
			return win32.VKRControl
		}
		return win32.VKLControl
	}
	return original
}

// isSysAction reports whether a message is one of the system messages (Alt held, or F10).
//
// Flutter: shell/platform/windows/keyboard_manager.cc IsSysAction
func isSysAction(msg uint32) bool {
	switch msg {
	case win32.WMSysKeyDown, win32.WMSysKeyUp, win32.WMSysChar, win32.WMSysDeadChar:
		return true
	}
	return false
}

// bypassesFramework reports whether a system key goes to DefWindowProc even when the framework
// handled it: Alt+F4 closes the window and Alt+Space opens its system menu (SPEC §25.6).
func bypassesFramework(key uint32) bool {
	return key == win32.VKF4 || key == win32.VKSpace
}

// handleKey handles WM_KEYDOWN, WM_KEYUP, WM_SYSKEYDOWN and WM_SYSKEYUP: it reads the characters
// that follow, converts the message to key data, gives them to the framework and says what the
// window procedure answers. msgTime is the time of the message (GetMessageTime).
//
// Deviation: Flutter's embedder passes every WM_SYSKEYDOWN to DefWindowProc and sends an unhandled
// key again; here the framework answers at once, a system key that it handled is consumed (but
// not Alt+F4 and Alt+Space), and WM_SYSKEYUP is always consumed (SPEC §12.10).
//
// Flutter: shell/platform/windows/keyboard_manager.cc KeyboardManager::HandleMessage
func (m *keyboardManager) handleKey(q keyQueue, msg uint32, wparam, lparam uintptr, msgTime uint32) keyAnswer {
	if wparam == win32.VKPacket {
		return keyToSystem // a character that SendInput sent: DefWindowProc ignores the key, and the WM_CHAR that follows is typed by the text input (SPEC §25.7)
	}
	scan := uint32(lparam>>16) & 0xff
	extended := (lparam>>24)&1 == 1
	key := m.resolveKeyCode(uint32(wparam), extended, scan)
	wasDown := lparam&0x40000000 != 0
	down := msg == win32.WMKeyDown || msg == win32.WMSysKeyDown
	sys := isSysAction(msg)

	if m.dropFakeControl(q, down, key, extended, msgTime) {
		return keyConsumed
	}

	var text string
	if down {
		text = m.takeText(q, sys)
		if !sys {
			m.sysDownHandled = false
		}
	}

	data, main := m.emb.hook(keyMessage{key: key, scan: scan, extended: extended, down: down, wasDown: wasDown, text: text})
	handled := main < 0 // nothing was made for the framework to answer
	for i, d := range data {
		answered := m.send != nil && m.send(d)
		if i == main {
			handled = answered
		}
	}

	switch {
	case !sys:
		if handled {
			return keyConsumed
		}
		return keyToSystem
	case msg == win32.WMSysKeyUp:
		// DefWindowProc takes the release of Alt alone for a call of the system menu, which would also
		// take the keyboard focus from the window: the release is never passed on.
		return keyConsumed
	default: // a system key down
		m.sysDownHandled = handled && !bypassesFramework(key)
		if m.sysDownHandled {
			return keyConsumed
		}
		return keyToSystem
	}
}

// handleChar handles the character messages that reach the window procedure. The characters of a
// key were taken when the key down was handled, so what comes here is a WM_SYSCHAR or
// WM_SYSDEADCHAR of a system key (consumed when the framework handled the key, so that Windows does
// not beep or open the system menu for it), or text that no key made (an input method's, or
// SendInput's), which the text input types (SPEC §25.7).
//
// Flutter: shell/platform/windows/keyboard_manager.cc KeyboardManager::HandleMessage (WM_CHAR, WM_SYSCHAR)
func (m *keyboardManager) handleChar(msg uint32) keyAnswer {
	if msg == win32.WMSysChar || msg == win32.WMSysDeadChar {
		if m.sysDownHandled {
			return keyConsumed
		}
	}
	return keyToSystem
}

// takeText takes the character messages that follow a key down.
//
// Deviation: Flutter takes them only when MapVirtualKey(MAPVK_VK_TO_CHAR) gives the key a character
// (SPEC §12.10). For a key without Alt they are taken off the queue, so they never reach the window
// procedure (the framework types the character of the key); for a system key they stay, because
// DefWindowProc needs the WM_SYSCHAR of an unhandled key (Alt+Space), and only the first is read.
//
// Flutter: shell/platform/windows/keyboard_manager.cc KeyboardManager::HandleMessage (the peek of the next key message)
func (m *keyboardManager) takeText(q keyQueue, sys bool) string {
	var c charCollector
	for i := 0; i < maxCharMessages; i++ {
		next, ok := q.Peek()
		if !ok || !isCharMessage(next.Message) {
			break
		}
		if sys {
			if next.Message == win32.WMSysChar {
				c.add(next.Message, next.WParam)
			}
			break
		}
		if next.Message != win32.WMChar && next.Message != win32.WMDeadChar {
			break
		}
		q.Take()
		c.add(next.Message, next.WParam)
	}
	return c.String()
}

// dropFakeControl finds the left Control press that Windows sends before the right Alt key of AltGr
// (same time, the Alt key next in the queue) and drops it, and the release that matches it, as
// Flutter's embedder drops the press (SPEC §25.6). It reports whether the message was dropped.
//
// Deviation: Flutter lets the framework see the fake Control and sends a key up for it after the
// release of the Alt key; the framework here types the character of an unhandled key only when
// no Control is held, so AltGr characters would be lost (SPEC §12.10).
//
// Flutter: shell/platform/windows/keyboard_manager.cc IsKeyDownAltRight, IsKeyUpAltRight, IsKeyDownCtrlLeft
func (m *keyboardManager) dropFakeControl(q keyQueue, down bool, key uint32, extended bool, msgTime uint32) bool {
	switch {
	case down && key == win32.VKLControl && !extended:
		if next, ok := q.Peek(); ok && (next.Message == win32.WMKeyDown || next.Message == win32.WMSysKeyDown) &&
			next.WParam == win32.VKMenu && (next.LParam>>24)&1 == 1 && next.Time == msgTime {
			m.fakeCtrl = true
			return true
		}
		m.fakeCtrl = false // a real press of the key
	case !down && key == win32.VKLControl && m.fakeCtrl:
		m.fakeCtrl = false
		return true
	}
	return false
}

// releaseAll releases the keys that are down when the window loses the keyboard focus.
//
// Flutter: no counterpart
func (m *keyboardManager) releaseAll() {
	m.fakeCtrl = false
	m.sendAll(m.emb.releaseAll())
}

// focusGained reads the lock states and the modifiers again when the window gets the focus.
//
// Flutter: no counterpart
func (m *keyboardManager) focusGained() { m.sendAll(m.emb.syncLocks()) }

// syncModifiers makes Shift and Control pressed or released in the framework as a mouse message
// says.
//
// Flutter: shell/platform/windows/keyboard_key_handler.cc KeyboardKeyHandler::SyncModifiersIfNeeded
func (m *keyboardManager) syncModifiers(shift, control bool) {
	m.sendAll(m.emb.syncModifiers(shift, control))
}

// layoutChanged is told that the user switched the keyboard layout (WM_INPUTLANGCHANGE).
func (m *keyboardManager) layoutChanged() { m.emb.layoutChanged() }

func (m *keyboardManager) sendAll(data []ui.KeyData) {
	for _, d := range data {
		if m.send != nil {
			m.send(d)
		}
	}
}
