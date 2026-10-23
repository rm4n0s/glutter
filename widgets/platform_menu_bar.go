// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"unicode/utf8"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
)

// ShortcutSerialization is the parts of a shortcut that a menu shows: either a
// trigger key or a character, and the modifiers. Dart's nullable booleans of
// the other kind are false here (use Trigger and Character to tell the kinds
// apart).
//
// Deviation: Flutter's toChannelRepresentation, which sends the shortcut to the
// platform menu bar, is not ported (PlatformMenuBar is not ported).
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization
type ShortcutSerialization struct {
	trigger                   foundation.Opt[services.LogicalKeyboardKey]
	character                 foundation.Opt[string]
	alt, control, meta, shift bool
}

// ShortcutSerializationOpts are the modifiers of a ShortcutSerialization; Shift
// is for the trigger form only, as a character carries its own case.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.modifier
type ShortcutSerializationOpts struct {
	// Alt is whether the Alt key is part of the shortcut.
	Alt bool
	// Control is whether the Control key is part of the shortcut.
	Control bool
	// Meta is whether the Meta key is part of the shortcut.
	Meta bool
	// Shift is whether the Shift key is part of the shortcut.
	Shift bool
}

// NewShortcutSerializationCharacter serializes a shortcut that is a single
// character with modifiers. In debug builds it panics with the tag
// ShortcutSerializationCharacterLength when the character is not one character.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.character
func NewShortcutSerializationCharacter(character string, opts ...ShortcutSerializationOpts) ShortcutSerialization {
	o := firstOpts(opts)
	if foundation.DebugMode && utf8.RuneCountInString(character) != 1 {
		panic(errors.New(tagShortcutSerializationCharacterLength, "the character of a ShortcutSerialization must be one character", "character", character))
	}
	return ShortcutSerialization{character: foundation.Some(character), alt: o.Alt, control: o.Control, meta: o.Meta}
}

// NewShortcutSerializationModifier serializes a shortcut that is a trigger key
// with modifiers. In debug builds it panics with the tag
// ShortcutSerializationModifierTrigger when the trigger is a modifier key.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.modifier
func NewShortcutSerializationModifier(trigger services.LogicalKeyboardKey, opts ...ShortcutSerializationOpts) ShortcutSerialization {
	o := firstOpts(opts)
	if foundation.DebugMode {
		switch trigger {
		case services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight,
			services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyControlLeft, services.LogicalKeyboardKeyControlRight,
			services.LogicalKeyboardKeyMeta, services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight,
			services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight:
			panic(errors.New(tagShortcutSerializationModifierTrigger, "specifying a modifier key as a trigger is not allowed; use the modifier options instead", "trigger", trigger.String()))
		}
	}
	return ShortcutSerialization{trigger: foundation.Some(trigger), alt: o.Alt, control: o.Control, meta: o.Meta, shift: o.Shift}
}

// Trigger is the trigger key, if the shortcut is a trigger with modifiers.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.trigger
func (s ShortcutSerialization) Trigger() foundation.Opt[services.LogicalKeyboardKey] {
	return s.trigger
}

// Character is the character, if the shortcut is a character with modifiers.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.character
func (s ShortcutSerialization) Character() foundation.Opt[string] { return s.character }

// Alt is whether the Alt key is part of the shortcut.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.alt
func (s ShortcutSerialization) Alt() bool { return s.alt }

// Control is whether the Control key is part of the shortcut.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.control
func (s ShortcutSerialization) Control() bool { return s.control }

// Meta is whether the Meta key is part of the shortcut.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.meta
func (s ShortcutSerialization) Meta() bool { return s.meta }

// Shift is whether the Shift key is part of the shortcut; it is always false
// for a character.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.shift
func (s ShortcutSerialization) Shift() bool { return s.shift }

// MenuSerializableShortcut is a ShortcutActivator that a menu can show: a
// SingleActivator or a CharacterActivator, or another that can be reduced to
// one of the two.
//
// Flutter: widgets/platform_menu_bar.dart MenuSerializableShortcut
type MenuSerializableShortcut interface {
	ShortcutActivator
	// SerializeForMenu is the shortcut as a trigger or a character with its
	// modifiers.
	//
	// Flutter: widgets/platform_menu_bar.dart MenuSerializableShortcut.serializeForMenu
	SerializeForMenu() ShortcutSerialization
}

// SerializeForMenu is the trigger and the modifiers of the activator.
//
// Flutter: widgets/shortcuts.dart SingleActivator.serializeForMenu
func (a SingleActivator) SerializeForMenu() ShortcutSerialization {
	return NewShortcutSerializationModifier(a.trigger, ShortcutSerializationOpts{Shift: a.shift, Alt: a.alt, Meta: a.meta, Control: a.control})
}

// SerializeForMenu is the character and the modifiers of the activator.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.serializeForMenu
func (a CharacterActivator) SerializeForMenu() ShortcutSerialization {
	return NewShortcutSerializationCharacter(a.character, ShortcutSerializationOpts{Alt: a.alt, Control: a.control, Meta: a.meta})
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShortcutSerializationCharacterLength = "ShortcutSerializationCharacterLength"
	tagShortcutSerializationModifierTrigger = "ShortcutSerializationModifierTrigger"
)
