//go:build linux

package xkb

import (
	"unicode"

	"github.com/rm4n0s/glutter/internal/platform/linux"
)

// evdevOffset turns an evdev scancode into an XKB keycode.
const evdevOffset = 8

// latinLimit is where the Latin characters end: a layout key counts as Latin when
// its character is ASCII.
const latinLimit = 0x80

// Key is a key press translated to Flutter's terms.
//
// Flutter: no counterpart
type Key struct {
	// Physical is the id of the PhysicalKeyboardKey: the place of the key on the
	// keyboard, the same on every layout.
	Physical uint64
	// Logical is the id of the LogicalKeyboardKey: what the key means in the
	// current layout.
	Logical uint64
	// Character is the text the key press produces, or "" (for a key that makes
	// no text, and while a dead key or compose sequence is in progress).
	Character string
	// Repeats is whether holding the key produces repeats.
	Repeats bool
}

// Layout translates evdev key presses with a keymap, its modifier state and a
// compose table. It is the keyboard of one seat. A Layout is used on the UI
// thread only.
//
// Flutter: no counterpart (the GTK embedder's FlKeyboardManager and
// FlKeyEmbedderResponder do this on Linux)
type Layout struct {
	keymap  *Keymap
	state   *State
	compose *Compose // nil when the locale has no compose table
	goals   map[Keycode]linux.LayoutGoal
}

// NewLayout makes the keyboard from a keymap and, optionally, a compose
// table. The Layout takes ownership of both and releases them in Close; on an
// error they are released at once. It fails with XKBStateFailed.
//
// Flutter: no counterpart
func NewLayout(keymap *Keymap, compose *Compose) (*Layout, error) {
	state, err := keymap.NewState()
	if err != nil {
		keymap.Close()
		if compose != nil {
			compose.Close()
		}
		return nil, err
	}
	return NewLayoutWithState(keymap, state, compose), nil
}

// NewLayoutWithState is NewLayout with a state that already exists, such as one
// from NewStateFromX11. The Layout takes ownership of the state as well.
//
// Flutter: no counterpart
func NewLayoutWithState(keymap *Keymap, state *State, compose *Compose) *Layout {
	goals := map[Keycode]linux.LayoutGoal{}
	for _, g := range linux.LayoutGoals() {
		goals[g.Keycode] = g
	}
	return &Layout{keymap: keymap, state: state, compose: compose, goals: goals}
}

// Close releases the keymap, the state and the compose table.
//
// Flutter: no counterpart
func (l *Layout) Close() {
	l.state.Close()
	l.keymap.Close()
	if l.compose != nil {
		l.compose.Close()
	}
}

// Keymap returns the keymap of the layout.
//
// Flutter: no counterpart
func (l *Layout) Keymap() *Keymap { return l.keymap }

// UpdateMaskGroups sets the modifiers and the layout group the way X11 reports
// them: the group in its base, latched and locked parts.
//
// Flutter: no counterpart
func (l *Layout) UpdateMaskGroups(depressed, latched, locked, baseGroup, latchedGroup, lockedGroup uint32) {
	l.state.UpdateMaskGroups(depressed, latched, locked, baseGroup, latchedGroup, lockedGroup)
}

// UpdateMask sets the modifiers and the layout group, as wl_keyboard.modifiers
// reports them. A pending compose sequence is not touched.
//
// Flutter: no counterpart
func (l *Layout) UpdateMask(depressed, latched, locked, group uint32) {
	l.state.UpdateMask(depressed, latched, locked, group)
}

// ModActive reports whether an xkb modifier ("Shift", "Control", "Mod1", ...) is
// effective.
//
// Flutter: no counterpart
func (l *Layout) ModActive(name string) bool { return l.state.ModActive(name) }

// Press translates a key press. evdev is the scancode that the compositor sends.
// The character goes through the compose state, so a dead key gives no text and
// the key after it gives the composed character.
//
// Flutter: no counterpart
func (l *Layout) Press(evdev uint32) (Key, error) { return l.translate(evdev, true) }

// Lookup translates a key without touching the compose state: for key releases
// and repeats.
//
// Flutter: no counterpart
func (l *Layout) Lookup(evdev uint32) (Key, error) { return l.translate(evdev, false) }

func (l *Layout) translate(evdev uint32, press bool) (Key, error) {
	code := evdev + evdevOffset
	k := Key{Repeats: l.keymap.KeyRepeats(code)}
	var ok bool
	if k.Physical, ok = linux.PhysicalKey(code); !ok {
		k.Physical = linux.GTKPlane | uint64(code)
	}
	sym := l.state.KeyOneSym(code)
	var err error
	if k.Logical, err = l.logical(code, sym); err != nil {
		return Key{}, err
	}
	if k.Character, err = l.character(sym, press); err != nil {
		return Key{}, err
	}
	return k, nil
}

// logical follows the GTK embedder: a key on the layout goals has the logical
// key of the first Latin character it has in any layout, so that shortcuts work
// on every layout; any other key has the logical key of its keysym.
func (l *Layout) logical(code Keycode, sym Keysym) (uint64, error) {
	if g, ok := l.goals[code]; ok {
		return l.goalLogical(code, g, sym)
	}
	return symLogical(sym, code)
}

func (l *Layout) goalLogical(code Keycode, g linux.LayoutGoal, sym Keysym) (uint64, error) {
	n := l.keymap.NumLayouts(code)
	current := l.state.KeyLayout(code)
	// The current layout is tried first, then the others in order.
	order := make([]uint32, 0, n)
	order = append(order, current)
	for layout := range n {
		if layout != current {
			order = append(order, layout)
		}
	}
	for _, layout := range order {
		s, ok := l.keymap.SymAt(code, layout, 0)
		if !ok {
			continue
		}
		lower, err := KeysymToLower(s)
		if err != nil {
			return 0, err
		}
		r, err := KeysymToUTF32(lower)
		if err != nil {
			return 0, err
		}
		if r > 0 && r < latinLimit {
			return uint64(r), nil
		}
	}
	if g.Mandatory {
		return g.Default, nil
	}
	if s, ok := l.keymap.SymAt(code, current, 0); ok {
		return symLogical(s, code)
	}
	return symLogical(sym, code)
}

// symLogical is the logical key of a keysym: the table of keys that are not
// characters, then the lower-case character, then the GTK plane.
func symLogical(sym Keysym, code Keycode) (uint64, error) {
	if id, ok := linux.KeysymLogicalKey(sym); ok {
		return id, nil
	}
	r, err := KeysymToUTF32(sym)
	if err != nil {
		return 0, err
	}
	if r != 0 {
		return linux.UnicodePlane | uint64(unicode.ToLower(r)), nil
	}
	if sym == 0 {
		return linux.GTKPlane | uint64(code), nil
	}
	return linux.GTKPlane | uint64(sym), nil
}

// character is the text of a key press. Like the GTK embedder it comes from the
// keysym, not from the modifiers, so Control+a is "a"; the compose state can
// replace it.
func (l *Layout) character(sym Keysym, press bool) (string, error) {
	if press && l.compose != nil && sym != 0 {
		switch l.compose.Feed(sym) {
		case Composing:
			return "", nil
		case ComposeCancelled:
			l.compose.Reset()
			return "", nil
		case Composed:
			text := l.compose.UTF8()
			l.compose.Reset()
			return text, nil
		}
	}
	r, err := KeysymToUTF32(sym)
	if err != nil {
		return "", err
	}
	if r == 0 {
		return "", nil
	}
	return string(r), nil
}

// Keysym returns the keysym that the key has with the modifiers and the group
// that apply now, without touching the compose state. An input method gets
// this together with XState.
//
// Flutter: no counterpart
func (l *Layout) Keysym(evdev uint32) Keysym { return l.state.KeyOneSym(evdev + evdevOffset) }

// X11 core modifier bits, the "state" that an input method expects.
const (
	xShiftMask   = 1 << 0
	xLockMask    = 1 << 1
	xControlMask = 1 << 2
	xMod1Mask    = 1 << 3
	xMod2Mask    = 1 << 4
	xMod3Mask    = 1 << 5
	xMod4Mask    = 1 << 6
	xMod5Mask    = 1 << 7
)

// XState returns the effective modifiers as the X11 core modifier mask (Shift 1,
// Lock 2, Control 4, Mod1 8, Mod2 16, Mod3 32, Mod4 64, Mod5 128), which IBus
// and Fcitx5 take as the state of a key event.
//
// Flutter: no counterpart
func (l *Layout) XState() uint32 {
	var mask uint32
	for _, m := range [...]struct {
		name string
		bit  uint32
	}{
		{"Shift", xShiftMask}, {"Lock", xLockMask}, {"Control", xControlMask},
		{"Mod1", xMod1Mask}, {"Mod2", xMod2Mask}, {"Mod3", xMod3Mask},
		{"Mod4", xMod4Mask}, {"Mod5", xMod5Mask},
	} {
		if l.state.ModActive(m.name) {
			mask |= m.bit
		}
	}
	return mask
}
