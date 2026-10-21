//go:build linux

package xkb

import (
	"syscall"
	"unsafe"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// Keycode is an XKB keycode: the evdev scancode plus 8.
//
// Flutter: no counterpart
type Keycode = uint32

// Keysym is an XKB keysym, such as 0x61 for "a".
//
// Flutter: no counterpart
type Keysym = uint32

// scratchSize is the size of the C buffer that text comes back in. No key has
// text that is longer.
const scratchSize = 64

// ruleNames is struct xkb_rule_names: strings that name a keymap in the xkb
// data files. A zero pointer means the default from the environment.
type ruleNames struct {
	rules, model, layout, variant, options uintptr
}

// Names selects a keymap from the xkb data files, like setxkbmap does. An empty
// field uses the default from the environment (XKB_DEFAULT_*) or the library.
//
// Flutter: no counterpart
type Names struct {
	// Rules is the rules file, such as "evdev".
	Rules string
	// Model is the keyboard model, such as "pc105".
	Model string
	// Layout is the layout, or a comma-separated list of layouts, such as "de" or
	// "us,ru".
	Layout string
	// Variant is the variant of each layout, such as "nodeadkeys".
	Variant string
	// Options is a comma-separated list of options, such as "grp:alt_shift_toggle".
	Options string
}

// Context is an xkb_context. Close releases it. It must outlive the keymaps made
// with it only until they exist: keymaps keep their own reference.
//
// Flutter: no counterpart
type Context struct {
	lib *library
	ptr uintptr
}

// NewContext opens libxkbcommon and creates a context with the default
// include path. It fails with XKBLoadFailed (or SymbolMissing) when the library
// is unusable and with XKBContextFailed when the context cannot be created.
//
// Flutter: no counterpart
func NewContext() (*Context, error) {
	lib, err := load()
	if err != nil {
		return nil, err
	}
	ptr := lib.contextNew(0)
	if ptr == 0 {
		return nil, errors.New("XKBContextFailed", "xkb_context_new failed")
	}
	return &Context{lib: lib, ptr: ptr}, nil
}

// Close releases the context. It is safe to call twice.
//
// Flutter: no counterpart
func (c *Context) Close() {
	if c.ptr != 0 {
		c.lib.contextUnref(c.ptr)
		c.ptr = 0
	}
}

// Keymap is an xkb_keymap: the layouts of one keyboard.
//
// Flutter: no counterpart
type Keymap struct {
	lib     *library
	ptr     uintptr
	scratch uintptr // C memory for one pointer, the out parameter of key_get_syms_by_level
}

func (c *Context) newKeymap(ptr uintptr, source string) (*Keymap, error) {
	if ptr == 0 {
		return nil, errors.New("XKBKeymapFailed", "libxkbcommon could not compile the keymap", "source", source)
	}
	return &Keymap{lib: c.lib, ptr: ptr, scratch: dl.Calloc(1, unsafe.Sizeof(uintptr(0)))}, nil
}

// KeymapFromNames compiles the keymap that the rule names select. It fails with
// XKBKeymapFailed when the xkb data files lack that layout.
//
// Flutter: no counterpart
func (c *Context) KeymapFromNames(n Names) (*Keymap, error) {
	strs := []*string{&n.Rules, &n.Model, &n.Layout, &n.Variant, &n.Options}
	var cstrs [5]uintptr
	defer func() {
		for _, p := range cstrs {
			dl.Free(p) // Free(0) is a no-op
		}
	}()
	for i, s := range strs {
		if *s != "" {
			cstrs[i] = dl.CString(*s)
		}
	}
	// The struct is in C memory so that no Go pointer is passed to C.
	mem := dl.Calloc(1, unsafe.Sizeof(ruleNames{}))
	defer dl.Free(mem)
	*(*ruleNames)(dl.Pointer(mem)) = ruleNames{cstrs[0], cstrs[1], cstrs[2], cstrs[3], cstrs[4]}
	return c.newKeymap(c.lib.keymapNewFromNames(c.ptr, mem, 0), "names "+n.Layout)
}

// KeymapFromString compiles a keymap in the text format (what
// xkb_keymap_get_as_string and a compositor's keymap file contain).
//
// Flutter: no counterpart
func (c *Context) KeymapFromString(text string) (*Keymap, error) {
	cs := dl.CString(text)
	defer dl.Free(cs)
	return c.newKeymap(c.lib.keymapNewFromString(c.ptr, cs, keymapFormatTextV1, 0), "string")
}

// KeymapFromFD compiles the keymap that a compositor sends: size bytes of text
// in the file fd, which are read through a read-only mapping. The caller keeps
// ownership of fd. A trailing NUL byte is not part of the text.
//
// Flutter: no counterpart
func (c *Context) KeymapFromFD(fd int, size uint32) (*Keymap, error) {
	if size == 0 {
		return nil, errors.New("XKBKeymapFailed", "the keymap is empty", "source", "fd")
	}
	data, err := syscall.Mmap(fd, 0, int(size), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, errors.NewErr("XKBKeymapFailed", err).SetMetadata("source", "fd")
	}
	n := len(data)
	if data[n-1] == 0 {
		n--
	}
	km, kerr := c.newKeymap(c.lib.keymapNewFromBuffer(c.ptr, uintptr(unsafe.Pointer(&data[0])), uintptr(n), keymapFormatTextV1, 0), "fd")
	if err := syscall.Munmap(data); err != nil && kerr == nil {
		km.Close()
		return nil, errors.NewErr("XKBKeymapFailed", err).SetMetadata("source", "fd")
	}
	return km, kerr
}

// Close releases the keymap. It is safe to call twice.
//
// Flutter: no counterpart
func (k *Keymap) Close() {
	if k.ptr != 0 {
		k.lib.keymapUnref(k.ptr)
		k.ptr = 0
		dl.Free(k.scratch)
		k.scratch = 0
	}
}

// String returns the keymap in the text format, which KeymapFromString reads
// back.
//
// Flutter: no counterpart
func (k *Keymap) String() string {
	p := k.lib.keymapGetAsString(k.ptr, keymapFormatTextV1)
	defer dl.Free(p)
	return dl.GoString(p)
}

// KeyRepeats reports whether holding the key produces repeats (modifiers do
// not).
//
// Flutter: no counterpart
func (k *Keymap) KeyRepeats(key Keycode) bool { return k.lib.keymapKeyRepeats(k.ptr, key) != 0 }

// NumLayouts returns the number of layouts (groups) that the key has.
//
// Flutter: no counterpart
func (k *Keymap) NumLayouts(key Keycode) uint32 { return k.lib.keymapNumLayoutsFor(k.ptr, key) }

// SymAt returns the keysym that the key produces in a layout at a shift level,
// and false when the key has none there or several (a key with several keysyms
// at one level is not a plain letter).
//
// Flutter: no counterpart
func (k *Keymap) SymAt(key Keycode, layout, level uint32) (Keysym, bool) {
	n := k.lib.keymapGetSymsLevel(k.ptr, key, layout, level, k.scratch)
	if n != 1 {
		return 0, false
	}
	p := *(*uintptr)(dl.Pointer(k.scratch))
	return *(*uint32)(dl.Pointer(p)), true
}

// State is an xkb_state: the keymap plus which modifiers and layout are active.
//
// Flutter: no counterpart
type State struct {
	lib     *library
	ptr     uintptr
	scratch uintptr
}

// NewState creates the state of a keyboard with no key pressed. It fails with
// XKBStateFailed.
//
// Flutter: no counterpart
func (k *Keymap) NewState() (*State, error) {
	ptr := k.lib.stateNew(k.ptr)
	if ptr == 0 {
		return nil, errors.New("XKBStateFailed", "xkb_state_new failed")
	}
	return &State{lib: k.lib, ptr: ptr, scratch: dl.Calloc(1, scratchSize)}, nil
}

// Close releases the state. It is safe to call twice.
//
// Flutter: no counterpart
func (s *State) Close() {
	if s.ptr != 0 {
		s.lib.stateUnref(s.ptr)
		s.ptr = 0
		dl.Free(s.scratch)
		s.scratch = 0
	}
}

// UpdateMask sets the modifiers and layout from what the compositor sent in
// wl_keyboard.modifiers (depressed, latched, locked and the group). The
// compositor, not the client, tracks which keys are down.
//
// Flutter: no counterpart
func (s *State) UpdateMask(depressed, latched, locked, group uint32) {
	// The compositor sends one group; it is the locked layout.
	s.lib.stateUpdateMask(s.ptr, depressed, latched, locked, 0, 0, group)
}

// UpdateMaskGroups is UpdateMask for a source that reports the layout group in
// three parts, as X11 does: the base (depressed), latched and locked group.
//
// Flutter: no counterpart
func (s *State) UpdateMaskGroups(depressed, latched, locked, baseGroup, latchedGroup, lockedGroup uint32) {
	s.lib.stateUpdateMask(s.ptr, depressed, latched, locked, baseGroup, latchedGroup, lockedGroup)
}

// KeyOneSym returns the keysym that the key produces now, with the active
// modifiers and layout, or 0 when there is none or there are several.
//
// Flutter: no counterpart
func (s *State) KeyOneSym(key Keycode) Keysym { return s.lib.stateKeyGetOneSym(s.ptr, key) }

// KeyLayout returns the layout (group) that applies to the key now.
//
// Flutter: no counterpart
func (s *State) KeyLayout(key Keycode) uint32 { return s.lib.stateKeyGetLayout(s.ptr, key) }

// KeyUTF8 returns the text that the key produces now, with the active modifiers
// (so Control+a gives "\x01"), or "".
//
// Flutter: no counterpart
func (s *State) KeyUTF8(key Keycode) string {
	n := s.lib.stateKeyGetUTF8(s.ptr, key, s.scratch, scratchSize)
	return utf8At(s.scratch, n)
}

// ModActive reports whether the modifier, named as in xkb ("Shift", "Control",
// "Mod1" for Alt, "Lock" for Caps Lock, "Mod2" for Num Lock), is effective now.
//
// Flutter: no counterpart
func (s *State) ModActive(name string) bool {
	cs := dl.CString(name)
	defer dl.Free(cs)
	return s.lib.stateModNameIs(s.ptr, cs, stateModsEffective) > 0
}

// utf8At copies n bytes of C text at addr (as reported by xkbcommon, which may
// say more than fit) into a string.
func utf8At(addr uintptr, n int32) string {
	if n <= 0 {
		return ""
	}
	return string(dl.Bytes(addr, min(int(n), scratchSize-1)))
}

// KeysymToUTF32 returns the Unicode character of a keysym, or 0 when it has none
// (dead keys, function keys). Keysyms for control keys give the control
// characters: Return gives 13, BackSpace 8.
//
// Flutter: no counterpart
func KeysymToUTF32(sym Keysym) (rune, error) {
	lib, err := load()
	if err != nil {
		return 0, err
	}
	return rune(lib.keysymToUTF32(sym)), nil
}

// KeysymToLower returns the lower-case keysym of a keysym, or the keysym itself
// when it has no case.
//
// Flutter: no counterpart
func KeysymToLower(sym Keysym) (Keysym, error) {
	lib, err := load()
	if err != nil {
		return 0, err
	}
	return lib.keysymToLower(sym), nil
}

// Compose is the compose state for dead keys and the Compose key. A sequence
// such as dead_circumflex then "e" turns into "ê".
//
// Flutter: no counterpart
type Compose struct {
	lib     *library
	table   uintptr
	state   uintptr
	scratch uintptr
}

// LocaleFromEnv picks the locale for the compose table the way libc does:
// LC_ALL, then LC_CTYPE, then LANG, then "C".
//
// Flutter: no counterpart
func LocaleFromEnv(getenv func(string) string) string {
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := getenv(name); v != "" {
			return v
		}
	}
	return "C"
}

// NewCompose loads the compose table of the locale (such as "en_US.UTF-8") and
// starts a compose state. It fails with XKBComposeFailed when the locale has no
// table.
//
// Flutter: no counterpart
func (c *Context) NewCompose(locale string) (*Compose, error) {
	cl := dl.CString(locale)
	defer dl.Free(cl)
	table := c.lib.composeTableNewFromLocale(c.ptr, cl, 0)
	if table == 0 {
		return nil, errors.New("XKBComposeFailed", "libxkbcommon has no compose table for the locale", "locale", locale)
	}
	state := c.lib.composeStateNew(table, 0)
	if state == 0 {
		c.lib.composeTableUnref(table)
		return nil, errors.New("XKBComposeFailed", "xkb_compose_state_new failed", "locale", locale)
	}
	return &Compose{lib: c.lib, table: table, state: state, scratch: dl.Calloc(1, scratchSize)}, nil
}

// Close releases the compose state and table. It is safe to call twice.
//
// Flutter: no counterpart
func (c *Compose) Close() {
	if c.state != 0 {
		c.lib.composeStateUnref(c.state)
		c.lib.composeTableUnref(c.table)
		c.state, c.table = 0, 0
		dl.Free(c.scratch)
		c.scratch = 0
	}
}

// Feed gives a pressed key's keysym to the sequence and returns the status after
// it. Modifier keysyms leave the sequence alone.
//
// Flutter: no counterpart
func (c *Compose) Feed(sym Keysym) ComposeStatus {
	c.lib.composeStateFeed(c.state, sym)
	return ComposeStatus(c.lib.composeStateGetStatus(c.state))
}

// UTF8 returns the text of a completed sequence, or "".
//
// Flutter: no counterpart
func (c *Compose) UTF8() string {
	n := c.lib.composeStateGetUTF8(c.state, c.scratch, scratchSize)
	return utf8At(c.scratch, n)
}

// Sym returns the keysym of a completed sequence, or 0.
//
// Flutter: no counterpart
func (c *Compose) Sym() Keysym { return c.lib.composeStateGetOneSym(c.state) }

// Reset drops the sequence in progress.
//
// Flutter: no counterpart
func (c *Compose) Reset() { c.lib.composeStateReset(c.state) }
