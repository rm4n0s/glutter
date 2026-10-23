//go:build linux && !android

package linux

// Key id planes (Flutter's key ID scheme, dev/tools/gen_keycodes/README.md).
// A logical key id is the plane in the bits above the lowest 32, and a value in
// the plane below.
const (
	// UnicodePlane holds the keys that are a character: the id is the Unicode
	// code point of the lower-case character.
	//
	// Flutter: dev/tools/gen_keycodes kUnicodePlane
	UnicodePlane uint64 = 0
	// GTKPlane holds the keys that Flutter has no id for: the id is the keysym
	// plus the plane.
	//
	// Flutter: dev/tools/gen_keycodes kGtkPlane
	GTKPlane uint64 = 0x15_0000_0000
)

// layoutGoal is a key whose logical key should not depend on the layout, so that
// shortcuts such as Control+C work on every layout (Flutter's layout_goals.json).
type layoutGoal struct {
	xkbKeycode uint32
	// logical is the id of the logical key on the US layout.
	logical uint64
	// mandatory goals fall back to logical on a layout that has no Latin letter
	// on the key; the others keep the layout's own character.
	mandatory bool
}

// LayoutGoal describes one key whose logical key is chosen by looking across the
// layouts of the keymap (Flutter's layout goals).
//
// Flutter: dev/tools/gen_keycodes/data/layout_goals.json
type LayoutGoal struct {
	// Keycode is the XKB keycode of the key.
	Keycode uint32
	// Default is the logical key id on the US layout.
	Default uint64
	// Mandatory is whether Default applies when no layout has a Latin character on
	// the key.
	Mandatory bool
}

// PhysicalKey returns the id of Flutter's PhysicalKeyboardKey for an XKB keycode
// (the evdev scancode plus 8), and false for a keycode Flutter does not know.
//
// Flutter: dev/tools/gen_keycodes xkb_to_physical_key_map
func PhysicalKey(xkbKeycode uint32) (uint64, bool) {
	id, ok := xkbToPhysicalKey[xkbKeycode]
	return id, ok
}

// KeysymLogicalKey returns the id of Flutter's LogicalKeyboardKey for a keysym
// that is not a printable character (Return, Shift_L, F1, ...), and false for any
// other keysym.
//
// Flutter: dev/tools/gen_keycodes gtk_keyval_to_logical_key_map
func KeysymLogicalKey(keysym uint32) (uint64, bool) {
	id, ok := keysymToLogicalKey[keysym]
	return id, ok
}

// LayoutGoals returns the layout goals, ordered by keycode.
//
// Flutter: dev/tools/gen_keycodes/data/layout_goals.json
func LayoutGoals() []LayoutGoal {
	goals := make([]LayoutGoal, len(layoutGoals))
	for i, g := range layoutGoals {
		goals[i] = LayoutGoal{Keycode: g.xkbKeycode, Default: g.logical, Mandatory: g.mandatory}
	}
	return goals
}
