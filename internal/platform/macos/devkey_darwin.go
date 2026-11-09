//go:build darwin && !ios

package macos

import (
	"strings"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// The dev events that make key input (SPEC §29.6) build real NSEvents and give them to the key handling of
// the first window, the function that keyDown:, keyUp: and flagsChanged: call: a key that nobody takes is
// not given to the next responder (it would beep), and nothing is posted to the system.

// The NSEventType values of the key events.
const (
	eventTypeKeyDown      uint64 = 10
	eventTypeKeyUp        uint64 = 11
	eventTypeFlagsChanged uint64 = 12
)

// devKeyFocusDelay is how long the script waits after the window got the focus for its keys: the framework
// moves its focus into the window's route in the frame after the focus event.
const devKeyFocusDelay = 150 * time.Millisecond

// focusForDevKeys gives the first window the keyboard focus when a script has key or text steps and the window is not
// the key window, and says whether it did: an accessory app never takes the focus from the user's apps, and
// the framework sends keys to the focused view only. The script waits a moment, as a user would.
func (p *Platform) focusForDevKeys() bool {
	w := p.devWindow()
	if !hasTextSteps(p.devEvents) || w == nil || w.closed || w.activated {
		return false
	}
	w.keyChanged(true)
	return true
}

// runDevKey runs a "key" step; it reports whether the step was one.
func (p *Platform) runDevKey(ev DevEvent) bool {
	if ev.Kind != "key" {
		return false
	}
	w := p.devWindow()
	if w == nil || w.closed {
		return true
	}
	key, ok := devKeyByName(ev.Key)
	if !ok {
		return true // ParseDevEvents refused it
	}
	p.guard(func() {
		var mods []devModifier
		for _, name := range strings.Split(ev.Mods, "+") {
			if m, ok := devModifierKeys[name]; ok {
				mods = append(mods, m)
			}
		}
		flags := devBaseFlags
		for _, m := range mods {
			flags |= m.flag | m.side
			w.devKeyEvent(eventTypeFlagsChanged, m.code, flags, "", "")
		}
		char, unmodified := key.char, key.char
		shift := strings.Contains(ev.Mods, "shift")
		if shift {
			char, unmodified = key.shift, key.shift
		}
		if strings.Contains(ev.Mods, "control") && len(char) == 1 && char[0] >= 'a' && char[0] <= 'z' {
			char = string(rune(char[0] - 0x60)) // Control+letter makes a control character
		}
		w.devKeyEvent(eventTypeKeyDown, key.code, flags, char, unmodified)
		w.devKeyEvent(eventTypeKeyUp, key.code, flags, char, unmodified)
		for i := len(mods) - 1; i >= 0; i-- {
			flags &^= mods[i].flag | mods[i].side
			w.devKeyEvent(eventTypeFlagsChanged, mods[i].code, flags, "", "")
		}
	})
	return true
}

// devKeyEvent makes the NSEvent of a key and handles it.
func (w *window) devKeyEvent(typ uint64, code uint16, flags uint64, chars, unmodified string) {
	w.keyEvent(w.makeKeyEvent(typ, code, flags, chars, unmodified), keyKindOf(typ), false)
}

// makeKeyEvent makes the NSEvent of a key, as AppKit makes it for the window.
//
// The selector has twelve arguments, so isARepeat and keyCode go on the stack, and purego puts each argument
// that objc.Send takes as a variadic one in a slot of 8 bytes, while the arm64 ABI of Apple packs them by
// their size (the BOOL at offset 0, the unsigned short at offset 2). The two arguments are given as two
// words that satisfy both: the first holds the BOOL in its low byte and the key code at bits 16 to 31 (what
// the packed callee reads on arm64), the second is the key code alone (the slot that the callee reads on
// amd64, where every argument has its own slot).
func (w *window) makeKeyEvent(typ uint64, code uint16, flags uint64, chars, unmodified string) objc.ID {
	cs, us := objcx.NewString(chars), objcx.NewString(unmodified)
	defer objcx.Release(cs)
	defer objcx.Release(us)
	return class("NSEvent").Send(sel("keyEventWithType:location:modifierFlags:timestamp:windowNumber:context:characters:charactersIgnoringModifiers:isARepeat:keyCode:"),
		typ, nsPoint{}, flags, w.p.s.mediaTime(), send[int64](w.ns, "windowNumber"), objc.ID(0), cs, us,
		uint64(code)<<16, uint64(code))
}

// keyKindOf is the kind of key event of an NSEventType.
func keyKindOf(typ uint64) int {
	switch typ {
	case eventTypeKeyUp:
		return keyKindUp
	case eventTypeFlagsChanged:
		return keyKindFlagsChanged
	}
	return keyKindDown
}
