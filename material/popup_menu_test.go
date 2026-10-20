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
