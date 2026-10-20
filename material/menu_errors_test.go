// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// unserializableShortcut is a shortcut that has nothing to show in a menu.
type unserializableShortcut struct{ widgets.SingleActivator }

func (unserializableShortcut) SerializeForMenu() widgets.ShortcutSerialization {
	return widgets.ShortcutSerialization{}
}

// TestMenuErrors checks the route of the failures of the menus, the popup menus
// and the dropdown buttons.
func TestMenuErrors(t *testing.T) {
	skipUnlessDebug(t)
	pump := func(t *testing.T, w widgets.Widget) *gluttertest.Tester {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.PumpWidget(w)
		return ts
	}
	app := func(body widgets.Widget) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: body})})
	}
	text := func(s string) widgets.Widget { return widgets.NewText(s) }
	items := []*DropdownMenuItem[string]{
		NewDropdownMenuItem(text("a"), DropdownMenuItemOpts[string]{Value: foundation.Some("a")}),
		NewDropdownMenuItem(text("b"), DropdownMenuItemOpts[string]{Value: foundation.Some("b")}),
	}
	tests := []struct {
		name      string
		run       func(t *testing.T)
		wantRoute string
	}{
		{"no accelerator binding", func(t *testing.T) {
			takeAndPanic(pump(t, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				MenuAcceleratorCallbackBindingOf(bc)
				return widgets.NewSizedBox(nil)
			})))
		}, "MenuAcceleratorCallbackBindingMissing"},
		{"shortcut with nothing to show", func(t *testing.T) {
			MenuShortcutLabel(unserializableShortcut{widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyA)}, DefaultMaterialLocalizations{})
		}, "MenuShortcutNotSerializable"},
		{"serialized character length", func(*testing.T) { widgets.NewShortcutSerializationCharacter("ab") }, "ShortcutSerializationCharacterLength"},
		{"serialized modifier as trigger", func(*testing.T) {
			widgets.NewShortcutSerializationModifier(services.LogicalKeyboardKeyShiftLeft)
		}, "ShortcutSerializationModifierTrigger"},
		{"hover delay in a menu bar", func(t *testing.T) {
			takeAndPanic(pump(t, app(NewMenuBar([]widgets.Widget{NewSubmenuButton(
				[]widgets.Widget{NewMenuItemButton(MenuItemButtonOpts{Child: text("x")})}, text("Menu"),
				SubmenuButtonOpts{HoverOpenDelay: 200 * time.Millisecond})}))))
		}, "SubmenuButtonHoverOpenDelayInMenuBar"},
		{"popup menu without items", func(t *testing.T) {
			ts := pump(t, app(widgets.NewSizedBox(nil)))
			ShowMenu(ts.Element(gluttertest.ByType[*Scaffold]()), nil, ShowMenuOpts{Position: foundation.Some(rendering.RelativeRect{})})
		}, "ShowMenuNoItems"},
		{"popup menu without a position", func(t *testing.T) {
			ts := pump(t, app(widgets.NewSizedBox(nil)))
			ShowMenu(ts.Element(gluttertest.ByType[*Scaffold]()), []PopupMenuEntry{NewPopupMenuItem[string](text("x"))})
		}, "ShowMenuPosition"},
		{"popup button with a child and an icon", func(*testing.T) {
			NewPopupMenuButton(func(widgets.BuildContext) []PopupMenuEntry { return nil },
				PopupMenuButtonOpts[string]{Child: text("x"), Icon: NewIcon(Icons.Add())})
		}, "PopupMenuButtonChildAndIcon"},
		{"dropdown value not in the items", func(*testing.T) {
			NewDropdownButton(items, func(foundation.Opt[string]) {}, DropdownButtonOpts[string]{Value: foundation.Some("z")})
		}, "DropdownButtonValueNotUnique"},
		{"dropdown value twice in the items", func(*testing.T) {
			NewDropdownButton(append([]*DropdownMenuItem[string]{items[0]}, items...), func(foundation.Opt[string]) {},
				DropdownButtonOpts[string]{Value: foundation.Some("a")})
		}, "DropdownButtonValueNotUnique"},
		{"dropdown item height", func(*testing.T) {
			NewDropdownButton(items, func(foundation.Opt[string]) {}, DropdownButtonOpts[string]{ItemHeight: foundation.Some(20.0)})
		}, "DropdownButtonItemHeight"},
		{"dropdown opened while open", func(t *testing.T) {
			ts := pump(t, app(NewDropdownButton(items, func(foundation.Opt[string]) {}, DropdownButtonOpts[string]{Value: foundation.Some("a")})))
			ts.Tap(gluttertest.ByType[*DropdownButton[string]]())
			ts.PumpAndSettle()
			widgets.ActionsInvoke(ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*DropdownButton[string]](), gluttertest.ByType[*InkWell]()))), widgets.ActivateIntent{})
		}, "DropdownButtonRouteOpen"},
		{"dropdown selected item builder length", func(t *testing.T) {
			takeAndPanic(pump(t, app(NewDropdownButton(items, func(foundation.Opt[string]) {}, DropdownButtonOpts[string]{
				Value: foundation.Some("a"),
				SelectedItemBuilder: func(widgets.BuildContext) []widgets.Widget {
					return []widgets.Widget{text("only one")}
				},
			}))))
		}, "DropdownButtonSelectedItemBuilder"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(func() { tc.run(t) }), tc.wantRoute)
		})
	}
}
