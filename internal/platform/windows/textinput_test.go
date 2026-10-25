package windows

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/ime"
	"github.com/rm4n0s/glutter/internal/platform/windows/ime/imetest"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests join the keyboard (keymanager.go) and the text input (package ime) the way the window
// procedure does (SPEC §25.6, §25.7): the characters of a key are the key's, and a character that no
// key made is typed by the text input. The two never type the same character.

// textTester runs the messages of a window over a key tester and a text input.
type textTester struct {
	*keyTester
	in     *ime.Input
	events []ui.TextInputEvent
}

func newTextTester(t *testing.T) *textTester {
	t.Helper()
	tt := &textTester{keyTester: newKeyTester(t, nil)}
	tt.in = ime.New(imetest.New(1), 1, func(e ui.TextInputEvent) { tt.events = append(tt.events, e) })
	tt.in.SetState(ui.TextInputState{Enabled: true}, 1)
	return tt
}

// char is a WM_CHAR that reaches the window procedure: the key manager has the system characters, the
// text input the others (onChar).
func (tt *textTester) char(c rune) {
	if tt.m.handleChar(win32.WMChar) == keyConsumed {
		return
	}
	tt.in.HandleChar(uint16(c))
}

// TestTelexTypingAFinishesWithAPackedCharacter is VietnameseTelexAddDiacriticWithFastResponse of
// keyboard_unittests.cc: pressing F after A makes the Telex input method send a Backspace and then the
// character à as VK_PACKET. The key A types "a" as a key; the Backspace is a key with no text; the
// packed character is typed by the text input, once; the keys VK_PACKET make no key event.
func TestTelexTypingAFinishesWithAPackedCharacter(t *testing.T) {
	tt := newTextTester(t)
	tt.handled = false
	if got := tt.deliver(kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, chr(win32.WMChar, 'a')); got != keyToSystem {
		t.Errorf("A down: answer %v, want the system", got)
	}
	tt.expect(we{down, physKeyA, logKeyA, "a", false})
	if len(tt.q.msgs) != 0 {
		t.Errorf("the character of the key is still in the queue: %v", tt.q.msgs)
	}
	tt.deliver(kmsg{msg: win32.WMKeyUp, vk: vkA, scan: scanKeyA})
	tt.expect(we{up, physKeyA, logKeyA, "", false})

	tt.deliver(kmsg{msg: win32.WMKeyDown, vk: win32.VKBack, scan: 0x0e}, chr(win32.WMChar, 8))
	tt.deliver(kmsg{msg: win32.WMKeyUp, vk: win32.VKBack, scan: 0x0e})
	if len(tt.got) != 2 || tt.got[0].Character != "" || tt.got[0].Type != down || tt.got[1].Type != up {
		t.Errorf("the Backspace: key data %+v", tt.got)
	}
	tt.got = nil

	if got := tt.deliver(kmsg{msg: win32.WMKeyDown, vk: win32.VKPacket}); got != keyToSystem {
		t.Errorf("VK_PACKET down: answer %v, want the system (the text input takes its character)", got)
	}
	tt.char('à')
	if got := tt.deliver(kmsg{msg: win32.WMKeyUp, vk: win32.VKPacket}); got != keyToSystem {
		t.Errorf("VK_PACKET up: answer %v, want the system", got)
	}
	if len(tt.got) != 0 {
		t.Errorf("VK_PACKET made key data: %+v", tt.got)
	}
	want := []ui.TextInputEvent{{Commit: "à", PreeditCursorBegin: -1, PreeditCursorEnd: -1}}
	if !reflect.DeepEqual(tt.events, want) {
		t.Errorf("text events = %+v, want %+v", tt.events, want)
	}
}

// TestKeysTheInputMethodTookTypeNothing: a key that an input method used (VK_PROCESSKEY) is the
// logical key Process, makes no key data, is handled and types nothing; the composition comes by the
// input method's own messages.
func TestKeysTheInputMethodTookTypeNothing(t *testing.T) {
	tests := []struct {
		name  string
		msg   uint32
		after []kmsg
	}{
		{"a letter", win32.WMKeyDown, nil},
		{"a letter with a character message that should not exist", win32.WMKeyDown, []kmsg{chr(win32.WMChar, 'a')}},
		{"the release of it", win32.WMKeyUp, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := newTextTester(t)
			tt.handled = false
			if got := tt.deliver(kmsg{msg: tc.msg, vk: win32.VKProcessKey, scan: scanKeyA}, tc.after...); got != keyConsumed {
				t.Errorf("answer %v, want the message handled", got)
			}
			tt.expect(we{down, 0, 0, "", false}) // the empty event: nothing for the framework
			if len(tt.events) != 0 {
				t.Errorf("text events %+v, want none", tt.events)
			}
		})
	}
	tt := newTextTester(t)
	if got := tt.m.emb.logicalKey(win32.VKProcessKey, false, scanKeyA); got != vkProcessKey {
		t.Errorf("the logical key of VK_PROCESSKEY = %#x, want %#x (Process)", got, vkProcessKey)
	}
}

// TestTextInputIsNotObstructedAfterEnter is the end of TextInputSubmit of keyboard_unittests.cc (the
// regression test for flutter/flutter#97706): Enter makes a key and a control character, which is
// no text, and the letter after it is a key with its character as before. The action of Enter is the
// framework's (TextInput.HandleUnhandledKey), so no text event is made.
func TestTextInputIsNotObstructedAfterEnter(t *testing.T) {
	tt := newTextTester(t)
	tt.handled = false
	tt.deliver(kmsg{msg: win32.WMKeyDown, vk: win32.VKReturn, scan: 0x1c}, chr(win32.WMChar, '\n'))
	if len(tt.got) != 1 || tt.got[0].Type != down || tt.got[0].Character != "" {
		t.Fatalf("Enter: key data %+v, want one key down with no character", tt.got)
	}
	tt.got = nil
	tt.deliver(kmsg{msg: win32.WMKeyUp, vk: win32.VKReturn, scan: 0x1c})
	tt.got = nil
	tt.deliver(kmsg{msg: win32.WMKeyDown, vk: vkA, scan: scanKeyA}, chr(win32.WMChar, 'a'))
	tt.expect(we{down, physKeyA, logKeyA, "a", false})
	if len(tt.events) != 0 {
		t.Errorf("text events %+v, want none", tt.events)
	}
}
