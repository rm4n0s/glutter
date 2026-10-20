// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// withPlatform runs the test with the target platform replaced.
func withPlatform(t *testing.T, p foundation.TargetPlatform) {
	t.Helper()
	skipUnlessDebug(t)
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(p))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func TestMenuShortcutLabel(t *testing.T) {
	ctrl := widgets.SingleActivatorOpts{Control: true}
	tests := []struct {
		name     string
		platform foundation.TargetPlatform
		shortcut widgets.MenuSerializableShortcut
		want     string
	}{
		{"control letter", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyA, ctrl), "Ctrl+A"},
		{"all modifiers", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyZ,
			widgets.SingleActivatorOpts{Control: true, Shift: true, Alt: true, Meta: true}), "Alt+Ctrl+Meta+Shift+Z"},
		{"arrow", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeyArrowLeft, ctrl), "Ctrl+←"},
		{"enter", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeyEnter), "↵"},
		{"escape", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeyEscape), "Esc"},
		{"space", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeySpace, ctrl), "Ctrl+Space"},
		{"delete", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeyDelete), "Del"},
		{"function key", foundation.TargetPlatformLinux, widgets.NewSingleActivator(services.LogicalKeyboardKeyF1), "F1"},
		{"windows meta", foundation.TargetPlatformWindows, widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyR, widgets.SingleActivatorOpts{Meta: true}), "Win+R"},
		{"character", foundation.TargetPlatformLinux, widgets.NewCharacterActivator("e", widgets.CharacterActivatorOpts{Alt: true, Control: true}), "Alt+Ctrl+e"},
		{"macOS symbols", foundation.TargetPlatformMacOS, widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyS,
			widgets.SingleActivatorOpts{Control: true, Alt: true, Shift: true, Meta: true}), "⌃ ⌥ ⇧ ⌘ S"},
		{"macOS command", foundation.TargetPlatformMacOS, widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyC, widgets.SingleActivatorOpts{Meta: true}), "⌘ C"},
		{"macOS character", foundation.TargetPlatformMacOS, widgets.NewCharacterActivator("x", widgets.CharacterActivatorOpts{Meta: true}), "⌘ x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withPlatform(t, tc.platform)
			if got := MenuShortcutLabel(tc.shortcut, DefaultMaterialLocalizations{}); got != tc.want {
				t.Errorf("label = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMenuAcceleratorLabelMarkers(t *testing.T) {
	tests := []struct {
		label     string
		want      string
		wantIndex int
		hasAccel  bool
	}{
		{"Open", "Open", -1, false},
		{"&Open", "Open", 0, true},
		{"Op&en", "Open", 2, true},
		{"Fish && Chips", "Fish & Chips", -1, false},
		{"&Fish && Chips", "Fish & Chips", 0, true},
		{"Fish && &Chips", "Fish & Chips", 7, true},
		{"Trailing &", "Trailing ", -1, false},
		{"A & B", "A  B", -1, false},
		{"&&&Open", "&Open", 1, true},
		{"Fish &&&Chips", "Fish &Chips", 6, true},
		{"First &One &Two", "First One Two", 6, true},
		{"", "", -1, false},
		{"&é accent", "é accent", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			index := 99
			got := MenuAcceleratorLabelStripAcceleratorMarkers(tc.label, func(i int) { index = i })
			if got != tc.want || index != tc.wantIndex {
				t.Errorf("strip = %q, index %d; want %q, index %d", got, index, tc.want, tc.wantIndex)
			}
			w := NewMenuAcceleratorLabel(tc.label)
			if w.DisplayLabel() != tc.want || w.HasAccelerator() != tc.hasAccel {
				t.Errorf("DisplayLabel = %q, HasAccelerator = %v; want %q, %v", w.DisplayLabel(), w.HasAccelerator(), tc.want, tc.hasAccel)
			}
		})
	}
}

// TestMenuAcceleratorLabelAlt checks that holding Alt underlines the accelerator,
// and that Alt with the letter activates the item.
func TestMenuAcceleratorLabelAlt(t *testing.T) {
	m := pumpMenuBar(t, func(m *menuTester) widgets.Widget {
		return NewMenuBar([]widgets.Widget{
			NewSubmenuButton([]widgets.Widget{
				NewMenuItemButton(MenuItemButtonOpts{OnPressed: func() { m.pressed = append(m.pressed, "Open") }, Child: NewMenuAcceleratorLabel("&Open")}),
				NewMenuItemButton(MenuItemButtonOpts{OnPressed: func() { m.pressed = append(m.pressed, "Save") }, Child: NewMenuAcceleratorLabel("&Save")}),
			}, NewMenuAcceleratorLabel("&File")),
		})
	})
	underlined := func() bool {
		found := false
		for _, w := range m.Widgets(gluttertest.ByType[*widgets.RichText]()) {
			w.(*widgets.RichText).Text().VisitChildren(func(span painting.InlineSpan) bool {
				if st := span.Style(); st != nil && st.Decoration == foundation.Some(ui.TextDecorationUnderline) {
					found = true
				}
				return true
			})
		}
		return found
	}
	if underlined() {
		t.Fatal("the accelerator is underlined without Alt")
	}
	m.SendKeyDownEvent(services.LogicalKeyboardKeyAltLeft)
	m.PumpAndSettle()
	if !underlined() {
		t.Error("the accelerator is not underlined while Alt is held")
	}
	// Alt+F opens the File menu.
	m.SendKeyEvent(services.LogicalKeyboardKeyKeyF, gluttertest.KeyOpts{Character: "f"})
	m.PumpAndSettle()
	if !m.Exists(gluttertest.ByType[*MenuItemButton]()) {
		t.Fatal("Alt+F did not open the menu")
	}
	// Alt+S activates Save.
	m.SendKeyEvent(services.LogicalKeyboardKeyKeyS, gluttertest.KeyOpts{Character: "s"})
	m.PumpAndSettle()
	m.FlushPosted()
	m.PumpAndSettle()
	m.SendKeyUpEvent(services.LogicalKeyboardKeyAltLeft)
	if len(m.pressed) != 1 || m.pressed[0] != "Save" {
		t.Errorf("pressed = %v, want [Save]", m.pressed)
	}
}
