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

func colorEntries() []DropdownMenuEntry[string] {
	return []DropdownMenuEntry[string]{
		NewDropdownMenuEntry("red", "Red"),
		NewDropdownMenuEntry("green", "Green"),
		NewDropdownMenuEntry("blue", "Blue"),
		NewDropdownMenuEntry("black", "Black", DropdownMenuEntryOpts{Disabled: true}),
	}
}

func pumpDropdownMenu(t *testing.T, menu widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(menu, widgets.AlignOpts{Alignment: painting.AlignmentTopCenter})})}))
	return ts
}

func fieldText(ts *gluttertest.Tester) string {
	return ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).Controller().Text()
}

// menuItem finds the button of the entry in the open menu: the last of the
// widgets with the label, because the field also lays out an invisible copy of
// the entries to measure them.
func menuItem(label string) gluttertest.Finder {
	return gluttertest.Last(gluttertest.Descendant(gluttertest.ByType[*MenuItemButton](), gluttertest.ByText(label)))
}

func openMenu(ts *gluttertest.Tester) {
	ts.Tap(gluttertest.ByType[*TextField]())
	ts.PumpAndSettle()
}

func TestDropdownMenuSelectsAnEntry(t *testing.T) {
	var selected []foundation.Opt[string]
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{
		Label:      widgets.NewText("Color"),
		OnSelected: func(v foundation.Opt[string]) { selected = append(selected, v) },
	}))
	if fieldText(ts) != "" {
		t.Fatalf("initial text = %q", fieldText(ts))
	}
	openMenu(ts)
	ts.Tap(menuItem("Green"))
	ts.PumpAndSettle()
	if len(selected) != 1 || selected[0] != foundation.Some("green") {
		t.Fatalf("onSelected calls = %v", selected)
	}
	if fieldText(ts) != "Green" {
		t.Errorf("field text = %q, want Green", fieldText(ts))
	}
	if ts.Exists(menuItem("Blue")) && ts.Count(gluttertest.ByType[*MenuItemButton]()) > 4 {
		t.Error("the menu is still open")
	}
}

func TestDropdownMenuInitialSelection(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{InitialSelection: str("blue")}))
	if fieldText(ts) != "Blue" {
		t.Errorf("field text = %q, want Blue", fieldText(ts))
	}
}

func TestDropdownMenuDisabledEntryIsNotSelectable(t *testing.T) {
	called := false
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{OnSelected: func(foundation.Opt[string]) { called = true }}))
	openMenu(ts)
	ts.Tap(menuItem("Black"))
	ts.PumpAndSettle()
	if called || fieldText(ts) != "" {
		t.Errorf("a disabled entry was selected: called %v, text %q", called, fieldText(ts))
	}
}

func TestDropdownMenuDisabled(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{Disabled: true}))
	ts.Tap(gluttertest.ByType[*TextField]())
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByType[*MenuItemButton]()) > len(colorEntries()) {
		t.Error("a disabled menu opened")
	}
}

func TestDropdownMenuFilter(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{EnableFilter: true}))
	openMenu(ts)
	ts.EnterText(gluttertest.ByType[*TextField](), "bl")
	ts.PumpAndSettle()
	// The menu has Blue and Black; the invisible copy of the entries has four.
	menuButtons := ts.Count(gluttertest.ByType[*MenuItemButton]()) - len(colorEntries())
	if menuButtons != 2 {
		t.Errorf("the menu has %d entries after the filter, want 2", menuButtons)
	}
}

func TestDropdownMenuCustomFilterCallback(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{
		EnableFilter: true,
		FilterCallback: func(entries []DropdownMenuEntry[string], filter string) []DropdownMenuEntry[string] {
			return entries[:1]
		},
	}))
	openMenu(ts)
	ts.EnterText(gluttertest.ByType[*TextField](), "x")
	ts.PumpAndSettle()
	if got := ts.Count(gluttertest.ByType[*MenuItemButton]()) - len(colorEntries()); got != 1 {
		t.Errorf("the menu has %d entries after the custom filter, want 1", got)
	}
}

func TestDropdownMenuKeyboardNavigation(t *testing.T) {
	var selected []foundation.Opt[string]
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{
		OnSelected: func(v foundation.Opt[string]) { selected = append(selected, v) },
	}))
	openMenu(ts)
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.Pump()
	if fieldText(ts) != "Red" {
		t.Errorf("text after one arrow down = %q, want Red", fieldText(ts))
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.Pump()
	if fieldText(ts) != "Blue" {
		t.Errorf("text after three arrows down = %q, want Blue", fieldText(ts))
	}
	// The next entry is disabled, so the highlight wraps to the first one.
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.Pump()
	if fieldText(ts) != "Red" {
		t.Errorf("text after wrapping = %q, want Red", fieldText(ts))
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowUp)
	ts.Pump()
	if fieldText(ts) != "Blue" {
		t.Errorf("text after arrow up = %q, want Blue", fieldText(ts))
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	ts.PumpAndSettle()
	if len(selected) == 0 || selected[len(selected)-1] != foundation.Some("blue") {
		t.Errorf("onSelected calls = %v, want blue last", selected)
	}
}

func TestDropdownMenuEscapeCloses(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries()))
	openMenu(ts)
	if ts.Count(gluttertest.ByType[*MenuItemButton]()) <= len(colorEntries()) {
		t.Fatal("the menu did not open")
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
	ts.PumpAndSettle()
	if ts.Count(gluttertest.ByType[*MenuItemButton]()) > len(colorEntries()) {
		t.Error("escape did not close the menu")
	}
}

func TestDropdownMenuSelectOnlyIsReadOnly(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{SelectOnly: true}))
	if !ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).ReadOnly() {
		t.Error("a select-only menu has an editable field")
	}
}

func TestDropdownMenuWidthAndTheme(t *testing.T) {
	theme := NewDropdownMenuThemeData(DropdownMenuThemeDataOpts{
		TextStyle: painting.NewTextStyle(painting.TextStyleOpts{Color: foundation.Some(ui.NewColor(0xFF123456)), FontSize: foundation.Some(20.0)}),
	})
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(
		NewDropdownMenuTheme(theme, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{Width: foundation.Some(250.0)})),
		widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})})}))
	if w := ts.RenderObject(gluttertest.ByType[*DropdownMenu[string]]()).(interface{ Size() ui.Size }).Size().Width; w != 250 {
		t.Errorf("width = %v, want 250", w)
	}
	style := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).Style()
	if style.Color != foundation.Some(ui.NewColor(0xFF123456)) || style.FontSize != foundation.Some(20.0) {
		t.Errorf("text style = %v, want the one of the theme", style)
	}
}

func TestDropdownMenuDisabledUsesTheDisabledColor(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{Disabled: true}))
	scheme := ThemeOf(ts.Element(gluttertest.ByType[*DropdownMenu[string]]())).ColorScheme
	style := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).Style()
	if want := scheme.OnSurface().WithOpacity(0.38); style.Color != foundation.Some(want) {
		t.Errorf("color = %v, want %v", style.Color, want)
	}
}

func TestDropdownMenuLeadingAndTrailingIcons(t *testing.T) {
	ts := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{LeadingIcon: NewIcon(Icons.Search())}))
	if !ts.Exists(gluttertest.ByIcon(Icons.Search())) {
		t.Error("no leading icon")
	}
	if !ts.Exists(gluttertest.ByIcon(Icons.ArrowDropDown())) {
		t.Error("no trailing icon")
	}
	openMenu(ts)
	if !ts.Exists(gluttertest.ByIcon(Icons.ArrowDropUp())) {
		t.Error("no selected trailing icon while the menu is open")
	}
	hidden := pumpDropdownMenu(t, NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{NoShowTrailingIcon: true}))
	if hidden.Exists(gluttertest.ByIcon(Icons.ArrowDropDown())) {
		t.Error("the trailing icon is shown")
	}
}

func TestDropdownMenuEntryAndThemeData(t *testing.T) {
	e := NewDropdownMenuEntry(3, "three", DropdownMenuEntryOpts{Disabled: true})
	if e.Value != 3 || e.Label != "three" || !e.Disabled {
		t.Errorf("entry = %+v", e)
	}
	a := NewDropdownMenuThemeData(DropdownMenuThemeDataOpts{DisabledColor: foundation.Some(ui.NewColor(0xFF000000))})
	b := NewDropdownMenuThemeData(DropdownMenuThemeDataOpts{DisabledColor: foundation.Some(ui.NewColor(0xFFFFFFFF))})
	if !a.Equal(a) || a.Equal(b) {
		t.Error("equality is wrong")
	}
	if got := DropdownMenuThemeDataLerp(a, b, 0.25).DisabledColor.Value(); got == a.DisabledColor.Value() || got == b.DisabledColor.Value() {
		t.Errorf("lerp = %v is not between the colors", got)
	}
	if got := a.CopyWith(DropdownMenuThemeDataCopy{DisabledColor: foundation.Some(ui.NewColor(0xFF00FF00))}); got.DisabledColor == a.DisabledColor {
		t.Error("CopyWith did not copy")
	}
	if s := a.String(); foundation.DebugMode && !containsStr(s, "disabledColor") {
		t.Errorf("description = %q", s)
	}
	for _, e := range []DropdownMenuCloseBehavior{DropdownMenuCloseBehaviorAll, DropdownMenuCloseBehaviorSelf, DropdownMenuCloseBehaviorNone} {
		if e.String() == "" {
			t.Errorf("%d has no name", int(e))
		}
	}
}

func TestDropdownMenuThemeInTheThemeData(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	td := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true),
		DropdownMenuTheme: NewDropdownMenuThemeData(DropdownMenuThemeDataOpts{DisabledColor: foundation.Some(ui.NewColor(0xFF00AA00))})})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: td, Home: NewScaffold(ScaffoldOpts{Body: NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{Disabled: true})})}))
	style := ts.Widget(gluttertest.ByType[*widgets.EditableText]()).(*widgets.EditableText).Style()
	if style.Color != foundation.Some(ui.NewColor(0xFF00AA00)) {
		t.Errorf("color = %v, want the disabled color of the theme", style.Color)
	}
}

func TestDropdownMenuErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"filter callback without filter", func() {
			NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{FilterCallback: func(e []DropdownMenuEntry[string], _ string) []DropdownMenuEntry[string] { return e }})
		}, "DropdownMenuFilterCallbackWithoutFilter"},
		{"trailing icon focus node without icon", func() {
			NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{TrailingIconFocusNode: widgets.NewFocusNode(widgets.FocusNodeOpts{}), NoShowTrailingIcon: true})
		}, "DropdownMenuTrailingIconFocusNodeWithoutIcon"},
		{"decoration builder with texts", func() {
			NewDropdownMenu(colorEntries(), DropdownMenuOpts[string]{HintText: str("h"),
				DecorationBuilder: func(widgets.BuildContext, *widgets.MenuController) InputDecoration { return InputDecoration{} }})
		}, "DropdownMenuDecorationBuilderWithTexts"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.run), tc.wantRoute)
		})
	}
}
