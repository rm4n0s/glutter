// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/widgets"
)

// shortcutGraphicEquivalents are the keys that are shown as a symbol.
//
// Flutter: material/menu_anchor.dart _LocalizedShortcutLabeler._shortcutGraphicEquivalents
var shortcutGraphicEquivalents = map[services.LogicalKeyboardKey]string{
	services.LogicalKeyboardKeyArrowLeft:  "←",
	services.LogicalKeyboardKeyArrowRight: "→",
	services.LogicalKeyboardKeyArrowUp:    "↑",
	services.LogicalKeyboardKeyArrowDown:  "↓",
	services.LogicalKeyboardKeyEnter:      "↵",
}

// shortcutKeyNames is the table of the localized names of the keys that have
// one.
func shortcutKeyNames(l MaterialLocalizations) map[services.LogicalKeyboardKey]string {
	return map[services.LogicalKeyboardKey]string{
		services.LogicalKeyboardKeyAltGraph:         l.KeyboardKeyAltGraph(),
		services.LogicalKeyboardKeyBackspace:        l.KeyboardKeyBackspace(),
		services.LogicalKeyboardKeyCapsLock:         l.KeyboardKeyCapsLock(),
		services.LogicalKeyboardKeyChannelDown:      l.KeyboardKeyChannelDown(),
		services.LogicalKeyboardKeyChannelUp:        l.KeyboardKeyChannelUp(),
		services.LogicalKeyboardKeyDelete:           l.KeyboardKeyDelete(),
		services.LogicalKeyboardKeyEject:            l.KeyboardKeyEject(),
		services.LogicalKeyboardKeyEnd:              l.KeyboardKeyEnd(),
		services.LogicalKeyboardKeyEscape:           l.KeyboardKeyEscape(),
		services.LogicalKeyboardKeyFn:               l.KeyboardKeyFn(),
		services.LogicalKeyboardKeyHome:             l.KeyboardKeyHome(),
		services.LogicalKeyboardKeyInsert:           l.KeyboardKeyInsert(),
		services.LogicalKeyboardKeyNumLock:          l.KeyboardKeyNumLock(),
		services.LogicalKeyboardKeyNumpad1:          l.KeyboardKeyNumpad1(),
		services.LogicalKeyboardKeyNumpad2:          l.KeyboardKeyNumpad2(),
		services.LogicalKeyboardKeyNumpad3:          l.KeyboardKeyNumpad3(),
		services.LogicalKeyboardKeyNumpad4:          l.KeyboardKeyNumpad4(),
		services.LogicalKeyboardKeyNumpad5:          l.KeyboardKeyNumpad5(),
		services.LogicalKeyboardKeyNumpad6:          l.KeyboardKeyNumpad6(),
		services.LogicalKeyboardKeyNumpad7:          l.KeyboardKeyNumpad7(),
		services.LogicalKeyboardKeyNumpad8:          l.KeyboardKeyNumpad8(),
		services.LogicalKeyboardKeyNumpad9:          l.KeyboardKeyNumpad9(),
		services.LogicalKeyboardKeyNumpad0:          l.KeyboardKeyNumpad0(),
		services.LogicalKeyboardKeyNumpadAdd:        l.KeyboardKeyNumpadAdd(),
		services.LogicalKeyboardKeyNumpadComma:      l.KeyboardKeyNumpadComma(),
		services.LogicalKeyboardKeyNumpadDecimal:    l.KeyboardKeyNumpadDecimal(),
		services.LogicalKeyboardKeyNumpadDivide:     l.KeyboardKeyNumpadDivide(),
		services.LogicalKeyboardKeyNumpadEnter:      l.KeyboardKeyNumpadEnter(),
		services.LogicalKeyboardKeyNumpadEqual:      l.KeyboardKeyNumpadEqual(),
		services.LogicalKeyboardKeyNumpadMultiply:   l.KeyboardKeyNumpadMultiply(),
		services.LogicalKeyboardKeyNumpadParenLeft:  l.KeyboardKeyNumpadParenLeft(),
		services.LogicalKeyboardKeyNumpadParenRight: l.KeyboardKeyNumpadParenRight(),
		services.LogicalKeyboardKeyNumpadSubtract:   l.KeyboardKeyNumpadSubtract(),
		services.LogicalKeyboardKeyPageDown:         l.KeyboardKeyPageDown(),
		services.LogicalKeyboardKeyPageUp:           l.KeyboardKeyPageUp(),
		services.LogicalKeyboardKeyPower:            l.KeyboardKeyPower(),
		services.LogicalKeyboardKeyPowerOff:         l.KeyboardKeyPowerOff(),
		services.LogicalKeyboardKeyPrintScreen:      l.KeyboardKeyPrintScreen(),
		services.LogicalKeyboardKeyScrollLock:       l.KeyboardKeyScrollLock(),
		services.LogicalKeyboardKeySelect:           l.KeyboardKeySelect(),
		services.LogicalKeyboardKeySpace:            l.KeyboardKeySpace(),
	}
}

// menuModifier is one of the four modifier keys.
type menuModifier int

const (
	menuModifierMeta menuModifier = iota
	menuModifierAlt
	menuModifierControl
	menuModifierShift
)

// MenuShortcutLabel is the text that a menu item shows for a shortcut, in the
// style of the platform: "Ctrl+Shift+A" or, on macOS and iOS, "⌃ ⇧ A". The
// names of the keys are localized.
//
// Flutter: material/menu_anchor.dart _LocalizedShortcutLabeler.getShortcutLabel
func MenuShortcutLabel(shortcut widgets.MenuSerializableShortcut, localizations MaterialLocalizations) string {
	serialized := shortcut.SerializeForMenu()
	keySeparator := "+" // Use "Ctrl+Shift+A" style.
	if usesSymbolicModifiers() {
		keySeparator = " " // Use "⌃ ⇧ A" style on macOS and iOS.
	}
	if trigger, ok := serialized.Trigger().Get(); ok {
		var modifiers []string
		add := func(on bool, m menuModifier) {
			if on {
				modifiers = append(modifiers, shortcutModifierLabel(m, localizations))
			}
		}
		if usesSymbolicModifiers() {
			// macOS/iOS platform convention uses this ordering, with ⌘ always last.
			add(serialized.Control(), menuModifierControl)
			add(serialized.Alt(), menuModifierAlt)
			add(serialized.Shift(), menuModifierShift)
			add(serialized.Meta(), menuModifierMeta)
		} else {
			// This order matches the LogicalKeySet version.
			add(serialized.Alt(), menuModifierAlt)
			add(serialized.Control(), menuModifierControl)
			add(serialized.Meta(), menuModifierMeta)
			add(serialized.Shift(), menuModifierShift)
		}
		var shortcutTrigger string
		if graphic, ok := shortcutGraphicEquivalents[trigger]; ok {
			shortcutTrigger = graphic
		} else {
			// Otherwise, look it up, and if there is no translation for it, then
			// fall back to the key label.
			name, found := shortcutKeyNames(localizations)[trigger]
			shortcutTrigger = name
			if !found && trigger.KeyID()&services.LogicalKeyboardKeyPlaneMask == 0 {
				// If the trigger is a Unicode-character-producing key, then use the
				// character.
				shortcutTrigger = strings.ToUpper(string(rune(trigger.KeyID() & services.LogicalKeyboardKeyValueMask)))
				found = true
			}
			// Fall back to the key label if all else fails.
			if !found {
				shortcutTrigger = trigger.KeyLabel()
			}
		}
		if shortcutTrigger != "" {
			modifiers = append(modifiers, shortcutTrigger)
		}
		return strings.Join(modifiers, keySeparator)
	}
	if character, ok := serialized.Character().Get(); ok {
		var modifiers []string
		add := func(on bool, m menuModifier) {
			if on {
				modifiers = append(modifiers, shortcutModifierLabel(m, localizations))
			}
		}
		// Character based shortcuts cannot check shifted keys.
		if usesSymbolicModifiers() {
			// macOS/iOS platform convention uses this ordering, with ⌘ always last.
			add(serialized.Control(), menuModifierControl)
			add(serialized.Alt(), menuModifierAlt)
			add(serialized.Meta(), menuModifierMeta)
		} else {
			// This order matches the LogicalKeySet version.
			add(serialized.Alt(), menuModifierAlt)
			add(serialized.Control(), menuModifierControl)
			add(serialized.Meta(), menuModifierMeta)
		}
		return strings.Join(append(modifiers, character), keySeparator)
	}
	panic(errors.New(tagMenuShortcutNotSerializable, "the shortcut has neither a trigger nor a character to show in a menu"))
}

// shortcutModifierLabel is the label of a modifier key on the platform.
//
// Flutter: material/menu_anchor.dart _LocalizedShortcutLabeler._getModifierLabel
func shortcutModifierLabel(modifier menuModifier, l MaterialLocalizations) string {
	platform := foundation.DefaultTargetPlatform()
	apple := platform == foundation.TargetPlatformIOS || platform == foundation.TargetPlatformMacOS
	switch modifier {
	case menuModifierMeta:
		switch platform {
		case foundation.TargetPlatformWindows:
			return l.KeyboardKeyMetaWindows()
		case foundation.TargetPlatformIOS, foundation.TargetPlatformMacOS:
			return "⌘"
		}
		return l.KeyboardKeyMeta()
	case menuModifierAlt:
		if apple {
			return "⌥"
		}
		return l.KeyboardKeyAlt()
	case menuModifierControl:
		// '⎈' (a boat helm wheel, not an asterisk) is apparently the standard
		// icon for "control", but only seems to appear on the French Canadian
		// keyboard. A '✲' (an open center asterisk) appears on some Microsoft
		// keyboards. For all but macOS (which has standardized on "⌃", it seems),
		// the local translation of "Ctrl" is used.
		if apple {
			return "⌃"
		}
		return l.KeyboardKeyControl()
	default:
		if apple {
			return "⇧"
		}
		return l.KeyboardKeyShift()
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagMenuShortcutNotSerializable = "MenuShortcutNotSerializable"
)
