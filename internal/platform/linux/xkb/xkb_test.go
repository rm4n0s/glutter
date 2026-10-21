//go:build linux

package xkb

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Evdev scancodes (linux/input-event-codes.h).
const (
	keyEsc       = 1
	key1         = 2
	keyQ         = 16
	keyE         = 18
	keyY         = 21
	keyEnter     = 28
	keyA         = 30
	keyGrave     = 41
	keyLeftShift = 42
	keyZ         = 44
	keyComma     = 51
	keySpace     = 57
	keyF1        = 59
	keyKP1       = 79
	keyLeftCtrl  = 29
)

// Modifier masks of the standard xkb modifier order.
const (
	maskShift = 1 << 0
	maskLock  = 1 << 1
	maskCtrl  = 1 << 2
	maskAlt   = 1 << 3 // Mod1
	maskNum   = 1 << 4 // Mod2
	maskAltGr = 1 << 7 // Mod5 (ISO_Level3_Shift)
)

// newLayout builds a keyboard from the xkb data files, with the compose table of
// en_US.UTF-8.
func newLayout(t *testing.T, n Names) *Layout {
	t.Helper()
	ctx, err := NewContext()
	if err != nil {
		t.Fatalf("libxkbcommon is needed to run these tests (sudo dnf install libxkbcommon): %v", err)
	}
	t.Cleanup(ctx.Close)
	km, err := ctx.KeymapFromNames(n)
	if err != nil {
		t.Fatalf("keymap %+v (the xkb data files are needed: sudo dnf install xkeyboard-config): %v", n, err)
	}
	compose, err := ctx.NewCompose("en_US.UTF-8")
	if err != nil {
		t.Fatalf("compose table (sudo dnf install libX11-common): %v", err)
	}
	l, err := NewLayout(km, compose)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(l.Close)
	return l
}

func TestLayoutPress(t *testing.T) {
	us := Names{Layout: "us"}
	de := Names{Layout: "de"}
	tests := []struct {
		name     string
		names    Names
		mask     [4]uint32 // depressed, latched, locked, group
		keys     []uint32  // pressed in order; the last one is checked
		physical uint64
		logical  uint64
		char     string
	}{
		{name: "a", names: us, keys: []uint32{keyA}, physical: 0x70004, logical: 'a', char: "a"},
		{name: "Shift+a is still the logical key a", names: us, mask: [4]uint32{maskShift}, keys: []uint32{keyA}, physical: 0x70004, logical: 'a', char: "A"},
		{name: "Caps Lock", names: us, mask: [4]uint32{0, 0, maskLock, 0}, keys: []uint32{keyA}, physical: 0x70004, logical: 'a', char: "A"},
		{name: "Control+a makes the character a", names: us, mask: [4]uint32{maskCtrl}, keys: []uint32{keyA}, physical: 0x70004, logical: 'a', char: "a"},
		{name: "Shift+1 is the logical key 1", names: us, mask: [4]uint32{maskShift}, keys: []uint32{key1}, physical: 0x7001e, logical: '1', char: "!"},
		{name: "comma", names: us, keys: []uint32{keyComma}, physical: 0x70036, logical: ',', char: ","},
		{name: "space", names: us, keys: []uint32{keySpace}, physical: 0x7002c, logical: ' ', char: " "},
		{name: "Enter", names: us, keys: []uint32{keyEnter}, physical: 0x70028, logical: 0x10000000d, char: "\r"},
		{name: "Escape", names: us, keys: []uint32{keyEsc}, physical: 0x70029, logical: 0x10000001b, char: "\x1b"},
		{name: "F1", names: us, keys: []uint32{keyF1}, physical: 0x7003a, logical: 0x100000801, char: ""},
		{name: "left Shift", names: us, keys: []uint32{keyLeftShift}, physical: 0x700e1, logical: 0x200000102, char: ""},
		{name: "keypad 1 with Num Lock", names: us, mask: [4]uint32{0, 0, maskNum, 0}, keys: []uint32{keyKP1}, physical: 0x70059, logical: 0x200000231, char: "1"},
		{name: "keypad 1 without Num Lock is End", names: us, keys: []uint32{keyKP1}, physical: 0x70059, logical: 0x200000231, char: ""},
		{name: "de: the key labelled Z sits where US has y", names: de, keys: []uint32{keyY}, physical: 0x7001c, logical: 'z', char: "z"},
		{name: "de: the key labelled Y sits where US has z", names: de, keys: []uint32{keyZ}, physical: 0x7001d, logical: 'y', char: "y"},
		{name: "de: AltGr+q is @", names: de, mask: [4]uint32{maskAltGr}, keys: []uint32{keyQ}, physical: 0x70014, logical: 'q', char: "@"},
		{name: "de: dead circumflex alone makes no text (its keysym is in the GTK plane)", names: de, keys: []uint32{keyGrave}, physical: 0x70035, logical: 0x1500000000 | 0xfe52, char: ""},
		{name: "de: dead circumflex then e is ê", names: de, keys: []uint32{keyGrave, keyE}, physical: 0x70008, logical: 'e', char: "ê"},
		{name: "de: dead circumflex then space is ^", names: de, keys: []uint32{keyGrave, keySpace}, physical: 0x7002c, logical: ' ', char: "^"},
		{name: "ru: the key labelled ф is the logical key a", names: Names{Layout: "ru"}, keys: []uint32{keyA}, physical: 0x70004, logical: 'a', char: "ф"},
		{name: "ru: the comma key is not a goal, so it keeps the layout's character", names: Names{Layout: "ru"}, keys: []uint32{keyComma}, physical: 0x70036, logical: 'б', char: "б"},
		{name: "us,ru in the second group still has the Latin logical key", names: Names{Layout: "us,ru"}, mask: [4]uint32{0, 0, 0, 1}, keys: []uint32{keyA}, physical: 0x70004, logical: 'a', char: "ф"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newLayout(t, tt.names)
			l.UpdateMask(tt.mask[0], tt.mask[1], tt.mask[2], tt.mask[3])
			var got Key
			for _, k := range tt.keys {
				var err error
				if got, err = l.Press(k); err != nil {
					t.Fatal(err)
				}
			}
			if got.Physical != tt.physical {
				t.Errorf("physical = 0x%x, want 0x%x", got.Physical, tt.physical)
			}
			if got.Logical != tt.logical {
				t.Errorf("logical = 0x%x, want 0x%x", got.Logical, tt.logical)
			}
			if got.Character != tt.char {
				t.Errorf("character = %q, want %q", got.Character, tt.char)
			}
		})
	}
}

func TestLayoutLookupDoesNotAdvanceCompose(t *testing.T) {
	l := newLayout(t, Names{Layout: "de"})
	if _, err := l.Press(keyGrave); err != nil {
		t.Fatal(err)
	}
	// Releases and repeats must not feed the sequence.
	for range 3 {
		if k, err := l.Lookup(keyGrave); err != nil || k.Character != "" {
			t.Fatalf("Lookup = %+v, %v", k, err)
		}
	}
	k, err := l.Press(keyE)
	if err != nil {
		t.Fatal(err)
	}
	if k.Character != "ê" {
		t.Errorf("character = %q, want ê", k.Character)
	}
}

func TestLayoutCancelledComposeStartsOver(t *testing.T) {
	l := newLayout(t, Names{Layout: "de"})
	// dead circumflex followed by a key that makes no sequence cancels it.
	if _, err := l.Press(keyGrave); err != nil {
		t.Fatal(err)
	}
	if k, err := l.Press(keyQ); err != nil || k.Character != "" {
		t.Fatalf("the key that cancels = %+v, %v; want no text", k, err)
	}
	k, err := l.Press(keyE)
	if err != nil {
		t.Fatal(err)
	}
	if k.Character != "e" {
		t.Errorf("after the cancelled sequence, e = %q, want e", k.Character)
	}
}

func TestLayoutWithoutCompose(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	km, err := ctx.KeymapFromNames(Names{Layout: "de"})
	if err != nil {
		t.Fatal(err)
	}
	l, err := NewLayout(km, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	k, err := l.Press(keyGrave)
	if err != nil {
		t.Fatal(err)
	}
	if k.Character != "" {
		t.Errorf("a dead key without a compose table gives %q, want no text", k.Character)
	}
}

func TestKeyRepeats(t *testing.T) {
	l := newLayout(t, Names{Layout: "us"})
	tests := []struct {
		name string
		key  uint32
		want bool
	}{
		{"a", keyA, true},
		{"space", keySpace, true},
		{"left Shift", keyLeftShift, false},
		{"left Control", keyLeftCtrl, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := l.Lookup(tt.key)
			if err != nil {
				t.Fatal(err)
			}
			if k.Repeats != tt.want {
				t.Errorf("Repeats = %v, want %v", k.Repeats, tt.want)
			}
		})
	}
}

func TestUnknownKeycodeIsInTheGTKPlane(t *testing.T) {
	l := newLayout(t, Names{Layout: "us"})
	k, err := l.Lookup(300) // no such key
	if err != nil {
		t.Fatal(err)
	}
	if want := uint64(0x1500000000) | (300 + 8); k.Physical != want {
		t.Errorf("physical = 0x%x, want 0x%x", k.Physical, want)
	}
}

func TestModActive(t *testing.T) {
	l := newLayout(t, Names{Layout: "us"})
	if l.ModActive("Shift") {
		t.Fatal("Shift is active before any modifier event")
	}
	l.UpdateMask(maskShift|maskCtrl, 0, 0, 0)
	for name, want := range map[string]bool{"Shift": true, "Control": true, "Mod1": false, "Lock": false} {
		if got := l.ModActive(name); got != want {
			t.Errorf("ModActive(%s) = %v, want %v", name, got, want)
		}
	}
}

func TestKeymapRoundTrip(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	km, err := ctx.KeymapFromNames(Names{Layout: "de"})
	if err != nil {
		t.Fatal(err)
	}
	defer km.Close()
	text := km.String()
	if len(text) < 1000 {
		t.Fatalf("keymap text is only %d bytes", len(text))
	}

	fromString, err := ctx.KeymapFromString(text)
	if err != nil {
		t.Fatal(err)
	}
	defer fromString.Close()

	// A compositor sends the text, with a NUL, through a file.
	path := filepath.Join(t.TempDir(), "keymap")
	if err := os.WriteFile(path, append([]byte(text), 0), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	fromFD, err := ctx.KeymapFromFD(int(f.Fd()), uint32(len(text)+1))
	if err != nil {
		t.Fatal(err)
	}
	defer fromFD.Close()

	for name, k := range map[string]*Keymap{"string": fromString, "fd": fromFD} {
		sym, ok := k.SymAt(keyY+evdevOffset, 0, 0)
		if !ok || sym != 'z' {
			t.Errorf("%s: the key at US y is %c (ok %v), want z", name, sym, ok)
		}
	}
}

func TestSymAtSeveralLayouts(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	km, err := ctx.KeymapFromNames(Names{Layout: "us,ru"})
	if err != nil {
		t.Fatal(err)
	}
	defer km.Close()
	code := uint32(keyA + evdevOffset)
	if n := km.NumLayouts(code); n != 2 {
		t.Fatalf("NumLayouts = %d, want 2", n)
	}
	if s, ok := km.SymAt(code, 0, 0); !ok || s != 'a' {
		t.Errorf("layout 0 = %c, want a", s)
	}
	if s, ok := km.SymAt(code, 1, 0); !ok || s != 0x6c6 { // Cyrillic_ef
		t.Errorf("layout 1 = 0x%x, want 0x6c6", s)
	}
	if _, ok := km.SymAt(keyLeftShift+evdevOffset, 0, 3); ok {
		t.Error("a level that does not exist gave a keysym")
	}
}

func TestKeyUTF8FollowsModifiers(t *testing.T) {
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
	st, err := km.NewState()
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	code := uint32(keyA + evdevOffset)
	for _, tt := range []struct {
		mask uint32
		want string
	}{{0, "a"}, {maskShift, "A"}, {maskCtrl, "\x01"}} {
		st.UpdateMask(tt.mask, 0, 0, 0)
		if got := st.KeyUTF8(code); got != tt.want {
			t.Errorf("mask %d: KeyUTF8 = %q, want %q", tt.mask, got, tt.want)
		}
	}
	if st.KeyLayout(code) != 0 {
		t.Error("layout is not 0")
	}
}

func TestKeysymHelpers(t *testing.T) {
	tests := []struct {
		name  string
		sym   Keysym
		utf32 rune
		lower Keysym
	}{
		{"a", 'a', 'a', 'a'},
		{"A", 'A', 'A', 'a'},
		{"adiaeresis", 0xe4, 0xe4, 0xe4},
		{"Adiaeresis", 0xc4, 0xc4, 0xe4},
		{"Return", 0xff0d, 13, 0xff0d},
		{"BackSpace", 0xff08, 8, 0xff08},
		{"dead_circumflex", 0xfe52, 0, 0xfe52},
		{"F1", 0xffbe, 0, 0xffbe},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := KeysymToUTF32(tt.sym); err != nil || got != tt.utf32 {
				t.Errorf("KeysymToUTF32 = %d, %v; want %d", got, err, tt.utf32)
			}
			if got, err := KeysymToLower(tt.sym); err != nil || got != tt.lower {
				t.Errorf("KeysymToLower = 0x%x, %v; want 0x%x", got, err, tt.lower)
			}
		})
	}
}

func TestLocaleFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"none", nil, "C"},
		{"LANG", map[string]string{"LANG": "de_DE.UTF-8"}, "de_DE.UTF-8"},
		{"LC_CTYPE beats LANG", map[string]string{"LANG": "de_DE.UTF-8", "LC_CTYPE": "fr_FR.UTF-8"}, "fr_FR.UTF-8"},
		{"LC_ALL beats all", map[string]string{"LANG": "de_DE.UTF-8", "LC_CTYPE": "fr_FR.UTF-8", "LC_ALL": "C.UTF-8"}, "C.UTF-8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LocaleFromEnv(func(k string) string { return tt.env[k] }); got != tt.want {
				t.Errorf("LocaleFromEnv = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRuleNamesLayout(t *testing.T) {
	dl.AssertLayout(t, "xkb_rule_names", 40, map[string]uintptr{
		"rules": 0, "model": 8, "layout": 16, "variant": 24, "options": 32,
	}, ruleNames{})
}

func TestFailures(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	t.Run("a layout the xkb data does not have", func(t *testing.T) {
		_, err := ctx.KeymapFromNames(Names{Layout: "no-such-layout"})
		assertRoute(t, err, "Context.KeymapFromNames->Context.newKeymap.XKBKeymapFailed")
	})
	t.Run("keymap text that does not compile", func(t *testing.T) {
		_, err := ctx.KeymapFromString("xkb_keymap { fake };")
		assertRoute(t, err, "Context.KeymapFromString->Context.newKeymap.XKBKeymapFailed")
	})
	t.Run("an empty keymap file", func(t *testing.T) {
		_, err := ctx.KeymapFromFD(0, 0)
		assertRoute(t, err, "Context.KeymapFromFD.XKBKeymapFailed")
	})
	t.Run("a keymap file that cannot be mapped", func(t *testing.T) {
		_, err := ctx.KeymapFromFD(-1, 10)
		assertRoute(t, err, "Context.KeymapFromFD.XKBKeymapFailed")
	})
	t.Run("a keymap file with garbage", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "garbage")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				t.Error(err)
			}
		}()
		if _, err := f.WriteString("not a keymap\x00"); err != nil {
			t.Fatal(err)
		}
		_, err = ctx.KeymapFromFD(int(f.Fd()), 13)
		assertRoute(t, err, "Context.KeymapFromFD->Context.newKeymap.XKBKeymapFailed")
	})
	t.Run("a locale without a compose table", func(t *testing.T) {
		_, err := ctx.NewCompose("zz_ZZ.nonsense")
		assertRoute(t, err, "Context.NewCompose.XKBComposeFailed")
	})
}

func TestLoadFailureRoute(t *testing.T) {
	t.Setenv(LibraryEnv, filepath.Join(t.TempDir(), "missing", "libxkbcommon.so.0"))
	_, err := loadLibrary()
	assertRoute(t, err, "loadLibrary.XKBLoadFailed")
}

func TestLoadOverrideMustBeAbsolute(t *testing.T) {
	t.Setenv(LibraryEnv, "libxkbcommon.so.0")
	_, err := loadLibrary()
	assertRoute(t, err, "loadLibrary.XKBLoadFailed")
}

func assertRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

func TestCloseTwice(t *testing.T) {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	km, err := ctx.KeymapFromNames(Names{Layout: "us"})
	if err != nil {
		t.Fatal(err)
	}
	st, err := km.NewState()
	if err != nil {
		t.Fatal(err)
	}
	cp, err := ctx.NewCompose("en_US.UTF-8")
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		cp.Close()
		st.Close()
		km.Close()
		ctx.Close()
	}
}

func TestKeysymAndXState(t *testing.T) {
	l := newLayout(t, Names{Layout: "us"})
	const evdevA = 30
	if got := l.Keysym(evdevA); got != 'a' {
		t.Errorf("Keysym(A) = 0x%x, want 'a'", got)
	}
	if got := l.XState(); got != 0 {
		t.Errorf("XState() = %d with no modifier, want 0", got)
	}
	l.UpdateMask(maskShift|maskCtrl, 0, 0, 0)
	if got := l.Keysym(evdevA); got != 'A' {
		t.Errorf("Keysym(A) with Shift = 0x%x, want 'A'", got)
	}
	// Shift 1, Control 4.
	if got := l.XState(); got != 1|4 {
		t.Errorf("XState() = %d, want 5", got)
	}
	l.UpdateMask(0, 0, maskLock, 0)
	if got := l.XState(); got != 2 {
		t.Errorf("XState() = %d with Caps Lock, want 2", got)
	}
	l.UpdateMask(maskAlt|maskNum, 0, 0, 0)
	if got := l.XState(); got != 8|16 {
		t.Errorf("XState() = %d with Alt and NumLock, want 24", got)
	}
}
