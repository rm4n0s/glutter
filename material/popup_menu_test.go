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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestPopupMenuButtonSelects(t *testing.T) {
	var selected []string
	canceled := 0
	entries := func(bc widgets.BuildContext) []PopupMenuEntry {
		return []PopupMenuEntry{
			NewPopupMenuItem(widgets.NewText("One"), PopupMenuItemOpts[string]{Value: foundation.Some("one")}),
			NewPopupMenuItem(widgets.NewText("Two"), PopupMenuItemOpts[string]{Value: foundation.Some("two")}),
			NewPopupMenuDivider(),
			NewPopupMenuItem(widgets.NewText("Off"), PopupMenuItemOpts[string]{Value: foundation.Some("off"), Disabled: true}),
		}
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
		NewPopupMenuButton(entries, PopupMenuButtonOpts[string]{
			OnSelected: func(v string) { selected = append(selected, v) },
			OnCanceled: func() { canceled++ },
		}))})}))
	button := gluttertest.ByType[*IconButton]()
	ts.Tap(button)
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Two")) {
		t.Fatal("the menu is not shown")
	}
	// A disabled item does nothing.
	ts.Tap(gluttertest.ByText("Off"))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("Two")) {
		t.Fatal("a disabled item closed the menu")
	}
	ts.Tap(gluttertest.ByText("Two"))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("Two")) {
		t.Error("the menu is still open")
	}
	if len(selected) != 1 || selected[0] != "two" {
		t.Errorf("selected = %v, want [two]", selected)
	}
	// A tap outside cancels.
	ts.Tap(button)
	ts.PumpAndSettle()
	ts.TapAt(ui.Offset{Dx: 5, Dy: 400})
	ts.PumpAndSettle()
	if canceled != 1 || ts.Exists(gluttertest.ByText("One")) {
		t.Errorf("canceled = %d, open = %v", canceled, ts.Exists(gluttertest.ByText("One")))
	}
}

// OnSelected and OnCanceled run after the navigator has finished popping the menu, as the callback of
// Flutter's showMenu future does: they may push a route (a dialog that asks about the chosen action).
// Before, they ran while the history was changing and a push reported NavigatorLocked.
func TestPopupMenuButtonCallbacksMayChangeTheNavigator(t *testing.T) {
	dialog := func(bc widgets.BuildContext) {
		ShowDialog[any](bc, func(widgets.BuildContext) widgets.Widget {
			return NewAlertDialog(AlertDialogOpts{Title: widgets.NewText("Sure?")})
		})
	}
	page := func(bc widgets.BuildContext) {
		NavigatorPush(bc, NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget {
			return NewScaffold(ScaffoldOpts{Body: widgets.NewText("Sure?")})
		}))
	}
	tests := []struct {
		name string
		// selected and canceled are what the callbacks do with the context of the button.
		selected, canceled func(widgets.BuildContext)
		// choose taps an item of the open menu, or outside it.
		choose func(ts *gluttertest.Tester)
	}{
		{"OnSelected shows a dialog", dialog, nil, func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("Delete")) }},
		{"OnSelected pushes a page", page, nil, func(ts *gluttertest.Tester) { ts.Tap(gluttertest.ByText("Delete")) }},
		{"OnCanceled shows a dialog", nil, dialog, func(ts *gluttertest.Tester) { ts.TapAt(ui.Offset{Dx: 5, Dy: 400}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := func(widgets.BuildContext) []PopupMenuEntry {
				return []PopupMenuEntry{NewPopupMenuItem(widgets.NewText("Delete"), PopupMenuItemOpts[string]{Value: foundation.Some("delete")})}
			}
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
				widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
					o := PopupMenuButtonOpts[string]{}
					if tt.selected != nil {
						o.OnSelected = func(string) { tt.selected(bc) }
					}
					if tt.canceled != nil {
						o.OnCanceled = func() { tt.canceled(bc) }
					}
					return NewPopupMenuButton(entries, o)
				}))})}))
			ts.Tap(gluttertest.ByType[*IconButton]())
			ts.PumpAndSettle()
			if !ts.Exists(gluttertest.ByText("Delete")) {
				t.Fatal("the menu is not shown")
			}
			tt.choose(ts)
			ts.PumpAndSettle()
			for _, err := range ts.TakeErrors() {
				t.Errorf("reported: %v", err)
			}
			if ts.Exists(gluttertest.ByText("Delete")) {
				t.Error("the menu is still open")
			}
			if !ts.Exists(gluttertest.ByText("Sure?")) {
				t.Error("the route that the callback pushed is not shown")
			}
		})
	}
}
