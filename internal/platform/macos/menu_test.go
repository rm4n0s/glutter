package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestKeyEquivalent(t *testing.T) {
	const meta, shift, alt, ctrl = ui.PlatformMenuShortcutModifierMeta, ui.PlatformMenuShortcutModifierShift, ui.PlatformMenuShortcutModifierAlt, ui.PlatformMenuShortcutModifierControl
	trigger := func(id uint64, mods int) ui.PlatformMenuItemData {
		return ui.PlatformMenuItemData{HasShortcutTrigger: true, ShortcutTrigger: id, ShortcutModifiers: mods}
	}
	tests := []struct {
		name string
		in   ui.PlatformMenuItemData
		key  string
		mask uint64
	}{
		{"no shortcut", ui.PlatformMenuItemData{Label: "x"}, "", 0},
		{"a character with Command", ui.PlatformMenuItemData{HasShortcutCharacter: true, ShortcutCharacter: "n", ShortcutModifiers: meta}, "n", menuMaskCommand},
		{"an upper case character keeps its case", ui.PlatformMenuItemData{HasShortcutCharacter: true, ShortcutCharacter: "N", ShortcutModifiers: meta}, "N", menuMaskCommand},
		{"a letter key is lower case", trigger(0x73, meta|shift), "s", menuMaskCommand | menuMaskShift},
		{"a letter key of upper case id", trigger(0x53, meta), "s", menuMaskCommand},
		{"a digit", trigger(0x31, meta|alt), "1", menuMaskCommand | menuMaskOption},
		{"all four modifiers", trigger(0x61, meta|shift|alt|ctrl), "a", menuMaskCommand | menuMaskShift | menuMaskOption | menuMaskControl},
		{"no modifiers", trigger(0x2c, 0), ",", 0},
		{"Escape", trigger(0x0010000001b, 0), "\x1b", 0},
		{"Backspace", trigger(0x00100000008, meta), "\x08", menuMaskCommand},
		{"Tab", trigger(0x00100000009, ctrl), "\t", menuMaskControl},
		{"Enter", trigger(0x0010000000d, meta), "\r", menuMaskCommand},
		{"Delete", trigger(0x0010000007f, meta), "\x7f", menuMaskCommand},
		{"F1", trigger(0x00100000801, 0), "", 0},
		{"F12", trigger(0x0010000080c, shift), "", menuMaskShift},
		{"F20", trigger(0x00100000814, 0), "", 0},
		{"the arrows", trigger(0x00100000302, meta), "", menuMaskCommand},
		{"ArrowRight", trigger(0x00100000303, 0), "", 0},
		{"ArrowUp", trigger(0x00100000304, 0), "", 0},
		{"ArrowDown", trigger(0x00100000301, 0), "", 0},
		{"Home and End", trigger(0x00100000306, 0), "", 0},
		{"End", trigger(0x00100000305, 0), "", 0},
		{"PageUp", trigger(0x00100000308, 0), "", 0},
		{"PageDown", trigger(0x00100000307, 0), "", 0},
		{"a key of another plane with no special character", trigger(0x00100000999, meta), "", 0},
		{"the modifier bits alone make no mask", ui.PlatformMenuItemData{ShortcutModifiers: meta}, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, mask := keyEquivalent(tt.in)
			if key != tt.key || mask != tt.mask {
				t.Errorf("keyEquivalent = %q, %#x; want %q, %#x", key, mask, tt.key, tt.mask)
			}
		})
	}
}

func TestProvidedSelectors(t *testing.T) {
	for i, want := range []string{"orderFrontStandardAboutPanel:", "terminate:", "submenuAction:", "hide:", "hideOtherApplications:",
		"unhideAllApplications:", "startSpeaking:", "stopSpeaking:", "toggleFullScreen:", "performMiniaturize:", "performZoom:", "arrangeInFront:"} {
		if got, ok := providedSelector(i); !ok || got != want {
			t.Errorf("providedSelector(%d) = %q, %v; want %q", i, got, ok, want)
		}
	}
	for _, i := range []int{-1, 12, 100} {
		if _, ok := providedSelector(i); ok {
			t.Errorf("providedSelector(%d) has a selector", i)
		}
	}
	if s, _ := providedSelector(providedServicesIndex); s != "submenuAction:" {
		t.Errorf("the Services item has the selector %q", s)
	}
}

func TestReplaceAppName(t *testing.T) {
	for in, want := range map[string]string{"About APP_NAME": "About Notes", "Hide APP_NAME": "Hide Notes", "Quit": "Quit", "APP_NAME APP_NAME": "Notes Notes", "": ""} {
		if got := replaceAppName(in, "Notes"); got != want {
			t.Errorf("replaceAppName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKeyEquivalentMask(t *testing.T) {
	if got := keyEquivalentMask(0); got != 0 {
		t.Errorf("no modifiers = %#x", got)
	}
	if got := keyEquivalentMask(1 << 10); got != 0 {
		t.Errorf("an unknown bit = %#x", got)
	}
}
