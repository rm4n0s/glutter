package xkb

import (
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/linux/dl"
)

// LibraryEnv names the environment variable that overrides the path of
// libxkbcommon, for setups such as NixOS (SPEC §8.1).
const LibraryEnv = "GLUTTER_XKBCOMMON_LIBRARY"

// Constants from xkbcommon.h.
const (
	keymapFormatTextV1 = 1 // XKB_KEYMAP_FORMAT_TEXT_V1
	stateModsEffective = 8 // XKB_STATE_MODS_EFFECTIVE
)

// ComposeStatus is the state of a compose sequence (enum xkb_compose_status).
//
// Flutter: no counterpart
type ComposeStatus int

// The values of ComposeStatus are those of enum xkb_compose_status.
const (
	// ComposeNothing means no sequence is in progress and the key is not part of
	// one: the key's own text applies.
	//
	// Flutter: no counterpart
	ComposeNothing ComposeStatus = 0
	// Composing means a sequence is in progress; the keys produce no text yet.
	//
	// Flutter: no counterpart
	Composing ComposeStatus = 1
	// Composed means the sequence is complete; Compose.UTF8 holds the result.
	//
	// Flutter: no counterpart
	Composed ComposeStatus = 2
	// ComposeCancelled means the key did not continue the sequence, which was
	// dropped.
	//
	// Flutter: no counterpart
	ComposeCancelled ComposeStatus = 3
)

// library holds the bound functions of libxkbcommon.
type library struct {
	contextNew   func(flags int32) uintptr
	contextUnref func(ctx uintptr)

	keymapNewFromNames  func(ctx, names uintptr, flags int32) uintptr
	keymapNewFromString func(ctx, text uintptr, format, flags int32) uintptr
	keymapNewFromBuffer func(ctx, buf uintptr, length uintptr, format, flags int32) uintptr
	keymapGetAsString   func(keymap uintptr, format int32) uintptr
	keymapUnref         func(keymap uintptr)
	keymapKeyRepeats    func(keymap uintptr, key uint32) int32
	keymapNumLayoutsFor func(keymap uintptr, key uint32) uint32
	keymapGetSymsLevel  func(keymap uintptr, key, layout, level uint32, out uintptr) int32

	stateNew          func(keymap uintptr) uintptr
	stateUnref        func(state uintptr)
	stateUpdateMask   func(state uintptr, depressed, latched, locked, depressedLayout, latchedLayout, lockedLayout uint32) uint32
	stateKeyGetOneSym func(state uintptr, key uint32) uint32
	stateKeyGetLayout func(state uintptr, key uint32) uint32
	stateKeyGetUTF8   func(state uintptr, key uint32, buf uintptr, size uintptr) int32
	stateModNameIs    func(state uintptr, name uintptr, typ int32) int32

	keysymToUTF32 func(sym uint32) uint32
	keysymToLower func(sym uint32) uint32

	composeTableNewFromLocale func(ctx, locale uintptr, flags int32) uintptr
	composeTableUnref         func(table uintptr)
	composeStateNew           func(table uintptr, flags int32) uintptr
	composeStateUnref         func(state uintptr)
	composeStateFeed          func(state uintptr, sym uint32) int32
	composeStateReset         func(state uintptr)
	composeStateGetStatus     func(state uintptr) int32
	composeStateGetUTF8       func(state uintptr, buf uintptr, size uintptr) int32
	composeStateGetOneSym     func(state uintptr) uint32
}

// loadLibrary opens libxkbcommon.so.0 and binds the functions. It fails with
// XKBLoadFailed when the library cannot be loaded, and with SymbolMissing when
// a function is absent.
func loadLibrary() (*library, error) {
	lib, err := dl.OpenOverride(LibraryEnv, "libxkbcommon.so.0", "libxkbcommon.so")
	if err != nil {
		md := map[string]any{}
		if appErr, ok := errors.FromError(err); ok {
			md = appErr.Metadata
		}
		return nil, errors.New("XKBLoadFailed", "could not load libxkbcommon",
			"names", md["names"], "dlerror", md["dlerror"])
	}
	l := new(library)
	for _, b := range []struct {
		fptr   any
		symbol string
	}{
		{&l.contextNew, "xkb_context_new"},
		{&l.contextUnref, "xkb_context_unref"},
		{&l.keymapNewFromNames, "xkb_keymap_new_from_names"},
		{&l.keymapNewFromString, "xkb_keymap_new_from_string"},
		{&l.keymapNewFromBuffer, "xkb_keymap_new_from_buffer"},
		{&l.keymapGetAsString, "xkb_keymap_get_as_string"},
		{&l.keymapUnref, "xkb_keymap_unref"},
		{&l.keymapKeyRepeats, "xkb_keymap_key_repeats"},
		{&l.keymapNumLayoutsFor, "xkb_keymap_num_layouts_for_key"},
		{&l.keymapGetSymsLevel, "xkb_keymap_key_get_syms_by_level"},
		{&l.stateNew, "xkb_state_new"},
		{&l.stateUnref, "xkb_state_unref"},
		{&l.stateUpdateMask, "xkb_state_update_mask"},
		{&l.stateKeyGetOneSym, "xkb_state_key_get_one_sym"},
		{&l.stateKeyGetLayout, "xkb_state_key_get_layout"},
		{&l.stateKeyGetUTF8, "xkb_state_key_get_utf8"},
		{&l.stateModNameIs, "xkb_state_mod_name_is_active"},
		{&l.keysymToUTF32, "xkb_keysym_to_utf32"},
		{&l.keysymToLower, "xkb_keysym_to_lower"},
		{&l.composeTableNewFromLocale, "xkb_compose_table_new_from_locale"},
		{&l.composeTableUnref, "xkb_compose_table_unref"},
		{&l.composeStateNew, "xkb_compose_state_new"},
		{&l.composeStateUnref, "xkb_compose_state_unref"},
		{&l.composeStateFeed, "xkb_compose_state_feed"},
		{&l.composeStateReset, "xkb_compose_state_reset"},
		{&l.composeStateGetStatus, "xkb_compose_state_get_status"},
		{&l.composeStateGetUTF8, "xkb_compose_state_get_utf8"},
		{&l.composeStateGetOneSym, "xkb_compose_state_get_one_sym"},
	} {
		if err := lib.Bind(b.fptr, b.symbol); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// load opens libxkbcommon once; a failure is returned on every call.
var load = sync.OnceValues(loadLibrary)
