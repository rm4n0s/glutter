package windows

import (
	"strings"
	"unicode/utf16"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
)

// This file is the plain-Go part of the characters of the keyboard (SPEC §25.6): the character
// messages that TranslateMessage queues after a key down, and what they make of the key's text. The
// dead keys are Windows' own: the layout keeps the accent, and the character message of the next
// key is the composed character (or the accent and then the character, when they do not combine),
// so this code never calls ToUnicodeEx, which would change the layout's dead-key state.

// deadKeyCharMask is the bit that MapVirtualKey(MAPVK_VK_TO_CHAR) sets in the character of a dead
// key.
//
// Flutter: shell/platform/windows/keyboard_utils.h kDeadKeyCharMask
const deadKeyCharMask = 0x80000000

// undeadChar is the character of a dead key without the mark of a dead key.
//
// Flutter: shell/platform/windows/keyboard_utils.h UndeadChar
func undeadChar(ch uint32) uint32 { return ch &^ deadKeyCharMask }

// maxCharMessages bounds the character messages taken after one key: a dead key followed by a key
// that does not combine with it makes two, a surrogate pair two more.
const maxCharMessages = 8

// isPrintable reports whether a character is text: not a control character of the ASCII range and
// not DEL.
//
// Flutter: shell/platform/windows/keyboard_manager.cc IsPrintable
func isPrintable(c uint32) bool {
	const minPrintable = ' '
	const del = 0x7F
	return c >= minPrintable && c != del
}

// charCollector builds the text of a key from the character messages that follow it.
//
// Deviation: Flutter's KeyboardManager gives the key the first character and sends the later ones
// as text (OnText); here the framework types the character of an unhandled key itself
// (TextInput.HandleUnhandledKey), so all the characters of the key are its character (SPEC §12.10).
//
// Flutter: shell/platform/windows/keyboard_manager.cc KeyboardManager::HandleMessage (WM_CHAR, WM_DEADCHAR)
type charCollector struct {
	text strings.Builder
	// high is a high surrogate that waits for its low surrogate.
	high uint16
}

// add takes a character message (WM_CHAR or WM_DEADCHAR) and its wParam. A dead character makes no
// text; the character after it carries the accent (SPEC §25.6). A control character is no text, and
// two messages that carry a surrogate pair are one character; a surrogate on its own is dropped.
//
// Flutter: shell/platform/windows/keyboard_manager.cc KeyboardManager::HandleMessage
func (c *charCollector) add(msg uint32, wparam uintptr) {
	if msg == win32.WMDeadChar || msg == win32.WMSysDeadChar {
		c.high = 0
		return
	}
	u := uint16(wparam)
	switch {
	case utf16.IsSurrogate(rune(u)) && u < 0xDC00: // a high surrogate
		c.high = u
	case utf16.IsSurrogate(rune(u)): // a low surrogate
		if c.high != 0 {
			c.text.WriteRune(utf16.DecodeRune(rune(c.high), rune(u)))
		}
		c.high = 0
	default:
		c.high = 0
		if isPrintable(uint32(u)) {
			c.text.WriteRune(rune(u))
		}
	}
}

// String is the text so far.
func (c *charCollector) String() string { return c.text.String() }

// isCharMessage reports whether a message is one of the character messages that follow a key.
func isCharMessage(msg uint32) bool {
	switch msg {
	case win32.WMChar, win32.WMDeadChar, win32.WMSysChar, win32.WMSysDeadChar:
		return true
	}
	return false
}
