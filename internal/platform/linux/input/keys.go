package input

import (
	"time"

	"github.com/rm4n0s/glutter/ui"
)

// KeyTracker remembers the keys that are down, with the data of their key-down
// event, so that the key-up reuses the physical and logical key even when the
// modifiers changed in between, and so that the keys can be released when the
// window loses the keyboard focus (SPEC §8.5).
//
// A KeyTracker is used on the UI thread only.
//
// Flutter: no counterpart
type KeyTracker struct {
	pressed []trackedKey
}

type trackedKey struct {
	code uint32
	data ui.KeyData
}

// Down records the key-down data of the key with the platform's key code, and
// reports false, recording nothing, when the key is already down.
//
// Flutter: no counterpart
func (t *KeyTracker) Down(code uint32, data ui.KeyData) bool {
	if t.index(code) >= 0 {
		return false
	}
	t.pressed = append(t.pressed, trackedKey{code, data})
	return true
}

// Data returns the key-down data of a key that is down, and false for a key that
// is not.
//
// Flutter: no counterpart
func (t *KeyTracker) Data(code uint32) (ui.KeyData, bool) {
	i := t.index(code)
	if i < 0 {
		return ui.KeyData{}, false
	}
	return t.pressed[i].data, true
}

// Up forgets the key and returns its key-up event, made from its key-down data at
// time now. It reports false for a key that is not down.
//
// Flutter: no counterpart
func (t *KeyTracker) Up(code uint32, now time.Duration) (ui.KeyData, bool) {
	i := t.index(code)
	if i < 0 {
		return ui.KeyData{}, false
	}
	up := released(t.pressed[i].data, now, false)
	t.pressed = append(t.pressed[:i], t.pressed[i+1:]...)
	return up, true
}

// ReleaseAll forgets every key and returns their key-up events, synthesized, in
// the order the keys went down.
//
// Flutter: no counterpart
func (t *KeyTracker) ReleaseAll(now time.Duration) []ui.KeyData {
	out := make([]ui.KeyData, len(t.pressed))
	for i, pk := range t.pressed {
		out[i] = released(pk.data, now, true)
	}
	t.pressed = nil
	return out
}

func (t *KeyTracker) index(code uint32) int {
	for i, pk := range t.pressed {
		if pk.code == code {
			return i
		}
	}
	return -1
}

// released makes the key-up event of a key-down event.
func released(down ui.KeyData, now time.Duration, synthesized bool) ui.KeyData {
	up := down
	up.TimeStamp = now
	up.Type = ui.KeyEventTypeUp
	up.Character = ""
	up.Synthesized = synthesized
	return up
}
