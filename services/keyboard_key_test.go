package services

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// These tests are ported from packages/flutter/test/services/keyboard_key_test.dart.

func skipUnlessDebug(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("needs the debug checks")
	}
}

func TestPhysicalKeysCanBeLookedUpByCode(t *testing.T) {
	tests := []struct {
		name string
		code uint64
		want PhysicalKeyboardKey
	}{
		{"a letter key", 0x00070004, PhysicalKeyboardKeyKeyA},
		{"a control key", 0x00070029, PhysicalKeyboardKeyEscape},
		{"a modifier key", 0x000700e1, PhysicalKeyboardKeyShiftLeft},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := FindPhysicalKeyByCode(tc.code); !ok || got != tc.want {
				t.Errorf("FindPhysicalKeyByCode(%#x) = %v, %v; want %v", tc.code, got, ok, tc.want)
			}
		})
	}
	if _, ok := FindPhysicalKeyByCode(0x12345); ok {
		t.Error("an unknown code was found")
	}
}

func TestPhysicalKeyValuesAreEqual(t *testing.T) {
	if PhysicalKeyboardKeyKeyA != NewPhysicalKeyboardKey(PhysicalKeyboardKeyKeyA.USBHidUsage()) {
		t.Error("keys with the same usage differ")
	}
	if NewPhysicalKeyboardKey(0x12345) != PhysicalKeyboardKey(0x12345) {
		t.Error("keys with the same usage differ")
	}
}

func TestPhysicalKeyDebugNames(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		key  PhysicalKeyboardKey
		want string
	}{
		{PhysicalKeyboardKeyKeyA, "Key A"},
		{PhysicalKeyboardKeyBackslash, "Backslash"},
		{PhysicalKeyboardKey(0x12345), "Key with ID 0x00012345"},
	}
	for _, tc := range tests {
		if got := tc.key.DebugName(); got != tc.want {
			t.Errorf("DebugName of %#x = %q, want %q", uint64(tc.key), got, tc.want)
		}
	}
}

func TestLogicalKeysCanBeLookedUpByID(t *testing.T) {
	for _, want := range []LogicalKeyboardKey{LogicalKeyboardKeyKeyA, LogicalKeyboardKeyEscape, LogicalKeyboardKeyShiftLeft} {
		if got, ok := FindLogicalKeyByKeyID(want.KeyID()); !ok || got != want {
			t.Errorf("FindLogicalKeyByKeyID(%#x) = %v, %v; want %v", want.KeyID(), got, ok, want)
		}
	}
	if _, ok := FindLogicalKeyByKeyID(0x100012345); ok {
		t.Error("an unknown id was found")
	}
}

func TestControlCharactersAreRecognized(t *testing.T) {
	tests := []struct {
		label string
		want  bool
	}{
		{"\x08", true}, {"\x09", true}, {"\x0a", true}, {"\x0d", true}, {"\x1b", true}, {"\x7f", true},
		{"A", false}, {" ", false}, {"~", false}, {" ", false}, {"", false}, {"ab", false},
	}
	for _, tc := range tests {
		if got := IsControlCharacter(tc.label); got != tc.want {
			t.Errorf("IsControlCharacter(%q) = %v, want %v", tc.label, got, tc.want)
		}
	}
}

func TestControlCharacterKeysUseTheirCharCodes(t *testing.T) {
	tests := []struct {
		key  LogicalKeyboardKey
		code uint64
	}{
		{LogicalKeyboardKeyBackspace, 0x08},
		{LogicalKeyboardKeyTab, 0x09},
		{LogicalKeyboardKeyEnter, 0x0d},
		{LogicalKeyboardKeyEscape, 0x1b},
		{LogicalKeyboardKeyDelete, 0x7f},
	}
	for _, tc := range tests {
		if got, want := tc.key.KeyID(), LogicalKeyboardKeyUnprintablePlane+tc.code; got != want {
			t.Errorf("%v has id %#x, want %#x", tc.key, got, want)
		}
	}
}

func TestBasicSynonymsCanBeLookedUp(t *testing.T) {
	tests := []struct{ key, want LogicalKeyboardKey }{
		{LogicalKeyboardKeyShiftLeft, LogicalKeyboardKeyShift},
		{LogicalKeyboardKeyControlLeft, LogicalKeyboardKeyControl},
		{LogicalKeyboardKeyAltLeft, LogicalKeyboardKeyAlt},
		{LogicalKeyboardKeyMetaLeft, LogicalKeyboardKeyMeta},
		{LogicalKeyboardKeyShiftRight, LogicalKeyboardKeyShift},
		{LogicalKeyboardKeyControlRight, LogicalKeyboardKeyControl},
		{LogicalKeyboardKeyAltRight, LogicalKeyboardKeyAlt},
		{LogicalKeyboardKeyMetaRight, LogicalKeyboardKeyMeta},
	}
	for _, tc := range tests {
		if got := tc.key.Synonyms(); !got.Equal(NewKeySet(tc.want)) {
			t.Errorf("synonyms of %v = %v, want {%v}", tc.key, got, tc.want)
		}
	}
	if got := LogicalKeyboardKeyKeyA.Synonyms(); len(got) != 0 {
		t.Errorf("synonyms of A = %v, want none", got)
	}
}

func TestSynonymsGetCollapsedAndExpanded(t *testing.T) {
	left := NewKeySet(LogicalKeyboardKeyShiftLeft, LogicalKeyboardKeyControlLeft, LogicalKeyboardKeyAltLeft, LogicalKeyboardKeyMetaLeft)
	right := NewKeySet(LogicalKeyboardKeyShiftRight, LogicalKeyboardKeyControlRight, LogicalKeyboardKeyAltRight, LogicalKeyboardKeyMetaRight)
	both := left.Clone()
	for k := range right {
		both[k] = struct{}{}
	}
	pseudo := NewKeySet(LogicalKeyboardKeyShift, LogicalKeyboardKeyControl, LogicalKeyboardKeyAlt, LogicalKeyboardKeyMeta)
	for name, in := range map[string]KeySet[LogicalKeyboardKey]{"left": left, "right": right, "both": both} {
		if got := CollapseSynonyms(in); !got.Equal(pseudo) {
			t.Errorf("collapsing %s gave %v", name, got)
		}
	}
	if got := CollapseSynonyms(KeySet[LogicalKeyboardKey]{}); len(got) != 0 {
		t.Errorf("collapsing nothing gave %v", got)
	}
	// Keys without a synonym stay as they are.
	if got := CollapseSynonyms(NewKeySet(LogicalKeyboardKeyKeyA, LogicalKeyboardKeyShiftLeft)); !got.Equal(NewKeySet(LogicalKeyboardKeyKeyA, LogicalKeyboardKeyShift)) {
		t.Errorf("collapsing A and left shift gave %v", got)
	}
	if got := ExpandSynonyms(pseudo); !got.Equal(both) {
		t.Errorf("expanding the pseudo-keys gave %v", got)
	}
	if got := ExpandSynonyms(NewKeySet(LogicalKeyboardKeyKeyA, LogicalKeyboardKeyShift)); !got.Equal(NewKeySet(LogicalKeyboardKeyKeyA, LogicalKeyboardKeyShiftLeft, LogicalKeyboardKeyShiftRight)) {
		t.Errorf("expanding A and shift gave %v", got)
	}
}

func TestLogicalKeyValuesAreEqual(t *testing.T) {
	if LogicalKeyboardKeyKeyA != NewLogicalKeyboardKey(LogicalKeyboardKeyKeyA.KeyID()) {
		t.Error("keys with the same id differ")
	}
}

func TestLogicalKeyLabel(t *testing.T) {
	tests := []struct {
		key  LogicalKeyboardKey
		want string
	}{
		{LogicalKeyboardKeyKeyA, "A"},
		{LogicalKeyboardKeyBackslash, `\`},
		{LogicalKeyboardKey(0xD9), "Ù"},
		{LogicalKeyboardKey(0xF9), "Ù"},
		{LogicalKeyboardKeyShiftLeft, "Shift Left"},
		{LogicalKeyboardKeyNumpadDecimal, "Numpad Decimal"},
		{LogicalKeyboardKeyNumpad1, "Numpad 1"},
		{LogicalKeyboardKeyDelete, "Delete"},
		{LogicalKeyboardKeyF12, "F12"},
		{LogicalKeyboardKeyMediaPlay, "Media Play"},
		{LogicalKeyboardKey(0x100012345), ""},
	}
	for _, tc := range tests {
		if got := tc.key.KeyLabel(); got != tc.want {
			t.Errorf("KeyLabel of %#x = %q, want %q", uint64(tc.key), got, tc.want)
		}
	}
}

func TestLogicalKeyDebugName(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		key  LogicalKeyboardKey
		want string
	}{
		{LogicalKeyboardKeyKeyA, "Key A"},
		{LogicalKeyboardKeyBackslash, "Backslash"},
		{LogicalKeyboardKey(0xD9), "Key Ù"},
		{LogicalKeyboardKeyMediaPlay, "Media Play"},
		{LogicalKeyboardKey(0x100012345), "Key with ID 0x00100012345"},
	}
	for _, tc := range tests {
		if got := tc.key.DebugName(); got != tc.want {
			t.Errorf("DebugName of %#x = %q, want %q", uint64(tc.key), got, tc.want)
		}
	}
}

func TestKeyAutogeneratedAndKnownLists(t *testing.T) {
	if LogicalKeyboardKey(LogicalKeyboardKeyGTKPlane|0x1234).IsAutogenerated() != true {
		t.Error("a key of a platform plane should be autogenerated")
	}
	if LogicalKeyboardKeyEnter.IsAutogenerated() {
		t.Error("Enter should not be autogenerated")
	}
	logical, physical := KnownLogicalKeys(), KnownPhysicalKeys()
	if len(logical) != 444 || len(physical) != 269 {
		t.Errorf("%d logical and %d physical keys, want 444 and 269", len(logical), len(physical))
	}
	for i := 1; i < len(logical); i++ {
		if logical[i-1] >= logical[i] {
			t.Fatalf("the logical keys are not in order at %d", i)
		}
	}
	// Changing the copy does not change the list.
	logical[0] = 0
	if KnownLogicalKeys()[0] == 0 {
		t.Error("KnownLogicalKeys returned the list itself")
	}
}

func TestPhysicalKeyOf(t *testing.T) {
	tests := []struct {
		logical LogicalKeyboardKey
		want    PhysicalKeyboardKey
		ok      bool
	}{
		{LogicalKeyboardKeyKeyA, PhysicalKeyboardKeyKeyA, true},
		{LogicalKeyboardKeyEnter, PhysicalKeyboardKeyEnter, true},
		{LogicalKeyboardKeyShiftLeft, PhysicalKeyboardKeyShiftLeft, true},
		{LogicalKeyboardKeyArrowLeft, PhysicalKeyboardKeyArrowLeft, true},
		{LogicalKeyboardKeyF5, PhysicalKeyboardKeyF5, true},
		{LogicalKeyboardKeyShift, 0, false},
		{LogicalKeyboardKeyExclamation, 0, false},
	}
	for _, tc := range tests {
		if got, ok := PhysicalKeyOf(tc.logical); ok != tc.ok || got != tc.want {
			t.Errorf("PhysicalKeyOf(%v) = %v, %v; want %v, %v", tc.logical, got, ok, tc.want, tc.ok)
		}
	}
}
