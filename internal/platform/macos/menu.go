package macos

import (
	"strings"
	"unicode/utf16"

	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the menu bar (SPEC §29.8): the key equivalent of a shortcut, the
// modifier mask, the selector of each item that the platform provides and the app's name in a title. The
// calls into AppKit are menu_darwin.go. It is a port of FlutterMenuPlugin.mm.

// appNameMarker is what a title has in the place of the app's name; Flutter's template menus and the apps'
// own titles use it.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm kAppName
const appNameMarker = "APP_NAME"

// replaceAppName puts the app's name in the place of APP_NAME in a title.
func replaceAppName(title, name string) string { return strings.ReplaceAll(title, appNameMarker, name) }

// The bits of NSEventModifierFlags that a key equivalent uses.
const (
	menuMaskShift   uint64 = 1 << 17
	menuMaskControl uint64 = 1 << 18
	menuMaskOption  uint64 = 1 << 19
	menuMaskCommand uint64 = 1 << 20
)

// keyEquivalentMask is the modifier mask of NSMenuItem for the modifier bits that the framework sends.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm KeyEquivalentModifierMaskForModifiers
func keyEquivalentMask(modifiers int) uint64 {
	var mask uint64
	if modifiers&ui.PlatformMenuShortcutModifierMeta != 0 {
		mask |= menuMaskCommand
	}
	if modifiers&ui.PlatformMenuShortcutModifierShift != 0 {
		mask |= menuMaskShift
	}
	if modifiers&ui.PlatformMenuShortcutModifierAlt != 0 {
		mask |= menuMaskOption
	}
	if modifiers&ui.PlatformMenuShortcutModifierControl != 0 {
		mask |= menuMaskControl
	}
	return mask
}

// specialKeys maps the logical key ids of the keys that have no character to the character that NSMenuItem
// uses for them: the control characters and the private-use characters of the function keys.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm GetMacOsSpecialKeys
var specialKeys = func() map[uint64]uint16 {
	m := map[uint64]uint16{
		0x00100000008: 0x08,   // Backspace
		0x00100000009: 0x09,   // Tab
		0x0010000000a: 0x0a,   // Newline
		0x0010000000c: 0x0c,   // FormFeed
		0x0010000000d: 0x0d,   // Enter
		0x0010000007f: 0x7f,   // Delete
		0x00100000302: 0xf702, // ArrowLeft
		0x00100000303: 0xf703, // ArrowRight
		0x00100000304: 0xf700, // ArrowUp
		0x00100000301: 0xf701, // ArrowDown
		0x00100000306: 0xf729, // Home
		0x00100000305: 0xf72b, // End
		0x00100000308: 0xf72c, // PageUp
		0x00100000307: 0xf72d, // PageDown
		0x0010000001b: 0x001b, // Escape
	}
	for i := uint64(0); i < 20; i++ {
		m[0x00100000801+i] = uint16(0xf704 + i) // F1 to F20
	}
	return m
}()

// keyEquivalent is the key equivalent string of the NSMenuItem of an item, and the modifier mask; the key is
// empty when the item has no shortcut. A character is the key as it is; a trigger key is a special key, or
// the lowercase character of a key of the Unicode plane; a trigger of another plane that is not special has
// none.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm menuItemFromFlutterRepresentation
func keyEquivalent(d ui.PlatformMenuItemData) (key string, mask uint64) {
	switch {
	case d.HasShortcutCharacter:
		key = d.ShortcutCharacter
	case d.HasShortcutTrigger:
		if c, ok := specialKeys[d.ShortcutTrigger]; ok {
			key = string(utf16.Decode([]uint16{c}))
		} else if d.ShortcutTrigger&0xff00000000 == 0 { // the Unicode plane of the key ids
			key = strings.ToLower(string(rune(uint16(d.ShortcutTrigger & 0x00ffffffff))))
		}
	}
	if key != "" {
		mask = keyEquivalentMask(d.ShortcutModifiers)
	}
	return key, mask
}

// providedSelector is the action of each item that the platform provides, in the order of
// widgets.PlatformProvidedMenuItemType. The Services submenu has no action of its own: it is the first item
// with a submenu action in the first menu of the default menu bar.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm GetMacOSProvidedMenus
var providedSelectors = [...]string{
	"orderFrontStandardAboutPanel:", // about
	"terminate:",                    // quit
	"submenuAction:",                // servicesSubmenu
	"hide:",                         // hide
	"hideOtherApplications:",        // hideOtherApplications
	"unhideAllApplications:",        // showAllApplications
	"startSpeaking:",                // startSpeaking
	"stopSpeaking:",                 // stopSpeaking
	"toggleFullScreen:",             // toggleFullScreen
	"performMiniaturize:",           // minimizeWindow
	"performZoom:",                  // zoomWindow
	"arrangeInFront:",               // arrangeWindowsInFront
}

// providedServicesIndex is the index of the Services submenu in providedSelectors.
const providedServicesIndex = 2

// providedSelector is the selector of the provided item with the index, false for an index that no item has.
func providedSelector(index int) (string, bool) {
	if index < 0 || index >= len(providedSelectors) {
		return "", false
	}
	return providedSelectors[index], true
}
