// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

var fruits = []string{"apple", "banana", "grape", "orange", "pineapple"}

// fruitOptions filters the fruits by the text, ignoring the case.
func fruitOptions(value services.TextEditingValue, deliver func([]string)) {
	var out []string
	for _, f := range fruits {
		if value.Text != "" && strings.Contains(f, strings.ToLower(value.Text)) {
			out = append(out, f)
		}
	}
	deliver(out)
}

// fruitSuggestions makes a list tile-like text for each fruit that has the text of
// the controller; a tap closes the view with the fruit.
func fruitSuggestions(bc widgets.BuildContext, c *SearchController, deliver func([]widgets.Widget)) {
	var out []widgets.Widget
	for _, f := range fruits {
		f := f
		if strings.Contains(f, strings.ToLower(c.Text())) {
			out = append(out, NewInkWell(widgets.NewPadding(painting.NewEdgeInsetsAll(16), widgets.NewText(f)),
				InkWellOpts{OnTap: func() { c.CloseView(foundation.Some(f)) }}))
		}
	}
	deliver(out)
}

func linuxTheme() *ThemeData {
	return NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true), VisualDensity: foundation.Some(VisualDensityStandard()),
		Platform: foundation.Some(foundation.TargetPlatformLinux)})
}

func TestAutocomplete(t *testing.T) {
	var selected []string
	ts := pumpButton(t, linuxTheme(), widgets.NewSizedBox(NewAutocomplete(fruitOptions, AutocompleteOpts[string]{
		OnSelected: func(s string) { selected = append(selected, s) }}), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))
	field := gluttertest.ByType[*TextFormField]()
	ts.EnterText(field, "ap")
	ts.PumpAndSettle()
	for _, want := range []string{"apple", "grape", "pineapple"} {
		if !ts.Exists(gluttertest.ByText(want)) {
			t.Errorf("the option %q is not shown", want)
		}
	}
	if ts.Exists(gluttertest.ByText("banana")) {
		t.Error("an option that does not match is shown")
	}
	ts.Tap(gluttertest.ByText("grape"))
	ts.PumpAndSettle()
	if len(selected) != 1 || selected[0] != "grape" {
		t.Errorf("selected = %v, want [grape]", selected)
	}
	if got := gluttertest.WidgetAs[*TextFormField](ts, field).Controller(); got != nil && got.Text() != "grape" {
		t.Errorf("the field has %q", got.Text())
	}
	if ts.Exists(gluttertest.ByText("apple")) {
		t.Error("the options are still shown after one was selected")
	}
}

func TestAutocompleteOptionsStyle(t *testing.T) {
	ts := pumpButton(t, linuxTheme(), widgets.NewSizedBox(NewAutocomplete(fruitOptions, AutocompleteOpts[string]{
		OptionsMaxHeight: foundation.Some(100.0)}), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))
	ts.EnterText(gluttertest.ByType[*TextFormField](), "e")
	ts.PumpAndSettle()
	// apple, grape, orange and pineapple have an "e": more than 100 pixels of options.
	list := gluttertest.ByType[*widgets.ListView]()
	if h := ts.Size(list).Height; h != 100 {
		t.Errorf("the options are %v high, want 100 (the maximum)", h)
	}
	mat := gluttertest.WidgetAs[*Material](ts, gluttertest.First(gluttertest.Ancestor(list, gluttertest.ByType[*Material]())))
	if mat.Elevation() != 4 {
		t.Errorf("elevation = %v, want 4", mat.Elevation())
	}
}

func TestAutocompleteKeyboard(t *testing.T) {
	var selected []string
	ts := pumpButton(t, linuxTheme(), widgets.NewSizedBox(NewAutocomplete(fruitOptions, AutocompleteOpts[string]{
		OnSelected: func(s string) { selected = append(selected, s) }}), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}))
	ts.EnterText(gluttertest.ByType[*TextFormField](), "ap")
	ts.PumpAndSettle()
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.Pump()
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	ts.PumpAndSettle()
	if len(selected) != 1 || selected[0] != "grape" {
		t.Errorf("selected = %v, want [grape] (the second option)", selected)
	}
}

func TestSearchBar(t *testing.T) {
	t.Run("size and parts", func(t *testing.T) {
		var changed []string
		ts := pumpButton(t, linuxTheme(), NewSearchBar(SearchBarOpts{
			HintText: foundation.Some("Search here"), Leading: widgets.NewIcon(Icons.Search()),
			Trailing: []widgets.Widget{widgets.NewIcon(Icons.Mic())}, OnChanged: func(s string) { changed = append(changed, s) }}))
		bar := gluttertest.ByType[*SearchBar]()
		size := ts.Size(bar)
		// The bar is as wide as the window of the Center allows, and 56 high.
		if size.Width != 400 || size.Height != 56 {
			t.Errorf("size = %v, want 400 by 56", size)
		}
		for _, text := range []string{"Search here"} {
			if !ts.Exists(gluttertest.ByText(text)) {
				t.Errorf("the hint %q is not shown", text)
			}
		}
		if !ts.Exists(gluttertest.ByIcon(Icons.Search())) || !ts.Exists(gluttertest.ByIcon(Icons.Mic())) {
			t.Error("the icons are not shown")
		}
		ts.EnterText(bar, "hello")
		ts.Pump()
		if len(changed) == 0 || changed[len(changed)-1] != "hello" {
			t.Errorf("onChanged = %v", changed)
		}
	})
	t.Run("colors and shape", func(t *testing.T) {
		theme := linuxTheme()
		ts := pumpButton(t, theme, NewSearchBar())
		mat := gluttertest.WidgetAs[*Material](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*SearchBar](), gluttertest.ByType[*Material]())))
		if mat.Color() != foundation.Some(theme.ColorScheme.SurfaceContainerHigh()) || mat.Elevation() != 6 {
			t.Errorf("color = %v, elevation = %v", mat.Color(), mat.Elevation())
		}
		if _, ok := mat.Shape().(painting.StadiumBorder); !ok {
			t.Errorf("shape = %T, want a stadium", mat.Shape())
		}
	})
	t.Run("disabled", func(t *testing.T) {
		tapped := 0
		ts := pumpButton(t, linuxTheme(), NewSearchBar(SearchBarOpts{Disabled: true, OnTap: func() { tapped++ }}))
		ts.Tap(gluttertest.ByType[*SearchBar]())
		ts.Pump()
		if tapped != 0 {
			t.Error("a disabled search bar was tapped")
		}
		op := gluttertest.WidgetAs[*widgets.Opacity](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*SearchBar](), gluttertest.ByType[*widgets.Opacity]())))
		if op.Opacity() != 0.38 {
			t.Errorf("opacity = %v, want 0.38", op.Opacity())
		}
	})
	t.Run("a tap focuses", func(t *testing.T) {
		focus := widgets.NewFocusNode()
		defer focus.Dispose()
		taps := 0
		ts := pumpButton(t, linuxTheme(), NewSearchBar(SearchBarOpts{FocusNode: focus, OnTap: func() { taps++ }}))
		ts.Tap(gluttertest.ByType[*SearchBar]())
		ts.Pump()
		if !focus.HasFocus() || taps == 0 {
			t.Errorf("focus = %v, taps = %d", focus.HasFocus(), taps)
		}
	})
	t.Run("theme", func(t *testing.T) {
		theme := NewSearchBarTheme(NewSearchBarThemeData(SearchBarThemeDataOpts{
			Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
				MinWidth: foundation.Some(200.0), MaxWidth: foundation.Some(300.0), MinHeight: foundation.Some(80.0)}))}), NewSearchBar())
		ts := pumpButton(t, linuxTheme(), theme)
		if size := ts.Size(gluttertest.ByType[*SearchBar]()); size.Width != 300 || size.Height != 80 {
			t.Errorf("size = %v, want 300 by 80", size)
		}
	})
}

// pumpSearchAnchor shows a search anchor bar in a window of 800 by 600.
func pumpSearchAnchor(t *testing.T, theme *ThemeData, anchor widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(
		widgets.NewPadding(painting.NewEdgeInsetsAll(24), anchor), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})})}))
	return ts
}

func TestSearchAnchorBar(t *testing.T) {
	var opened, closed int
	controller := NewSearchController()
	defer controller.Dispose()
	ts := pumpSearchAnchor(t, linuxTheme(), NewSearchAnchorBar(fruitSuggestions, SearchAnchorBarOpts{
		SearchController: controller, BarHintText: foundation.Some("Search fruits"), OnOpen: func() { opened++ }, OnClose: func() { closed++ }}))
	if controller.IsOpen() {
		t.Fatal("the view is open at the start")
	}
	ts.Tap(gluttertest.ByType[*SearchBar]())
	ts.PumpAndSettle()
	if !controller.IsOpen() || opened != 1 {
		t.Fatalf("open = %v, onOpen %d times, want an open view", controller.IsOpen(), opened)
	}
	// All the fruits are suggestions of the empty text, in the view.
	for _, f := range fruits {
		if !ts.Exists(gluttertest.ByText(f)) {
			t.Errorf("the suggestion %q is not shown", f)
		}
	}
	// The view has its own search bar with the hint, a back button and no clear button.
	if !ts.Exists(gluttertest.ByIcon(Icons.ArrowBack())) {
		t.Error("the view has no back button")
	}
	if ts.Exists(gluttertest.ByIcon(Icons.Close())) {
		t.Error("the view has a clear button with no text")
	}
	ts.EnterText(gluttertest.Last(gluttertest.ByType[*SearchBar]()), "pine")
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("pineapple")) || ts.Exists(gluttertest.ByText("banana")) {
		t.Error("the suggestions did not follow the text")
	}
	if !ts.Exists(gluttertest.ByIcon(Icons.Close())) {
		t.Error("the view has no clear button with a text")
	}
	ts.Tap(gluttertest.ByText("pineapple"))
	ts.PumpAndSettle()
	if controller.IsOpen() || closed != 1 {
		t.Errorf("open = %v, onClose %d times, want a closed view", controller.IsOpen(), closed)
	}
	if controller.Text() != "pineapple" {
		t.Errorf("text = %q, want pineapple", controller.Text())
	}
}

func TestSearchAnchorViewGeometry(t *testing.T) {
	tests := []struct {
		name     string
		platform foundation.TargetPlatform
		full     foundation.Opt[bool]
		wantSize ui.Size
	}{
		{"desktop view grows from the anchor", foundation.TargetPlatformLinux, foundation.None[bool](), ui.Size{Width: 752, Height: 400}},
		{"android view is full screen", foundation.TargetPlatformAndroid, foundation.None[bool](), ui.Size{Width: 800, Height: 600}},
		{"full screen on the desktop", foundation.TargetPlatformLinux, foundation.Some(true), ui.Size{Width: 800, Height: 600}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true), VisualDensity: foundation.Some(VisualDensityStandard()),
				Platform: foundation.Some(tc.platform)})
			ts := pumpSearchAnchor(t, theme, NewSearchAnchorBar(fruitSuggestions, SearchAnchorBarOpts{IsFullScreen: tc.full}))
			ts.Tap(gluttertest.ByType[*SearchBar]())
			ts.PumpAndSettle()
			view := gluttertest.First(gluttertest.Ancestor(gluttertest.Last(gluttertest.ByType[*SearchBar]()), gluttertest.ByType[*Material]()))
			if got := ts.Size(view); got != tc.wantSize {
				t.Errorf("view size = %v, want %v", got, tc.wantSize)
			}
		})
	}
}

func TestSearchAnchor(t *testing.T) {
	t.Run("a custom anchor", func(t *testing.T) {
		var seen []string
		ts := pumpSearchAnchor(t, linuxTheme(), NewSearchAnchor(func(_ widgets.BuildContext, c *SearchController) widgets.Widget {
			return NewIconButton(func() { c.OpenView() }, widgets.NewIcon(Icons.Search()))
		}, func(_ widgets.BuildContext, c *SearchController, deliver func([]widgets.Widget)) {
			seen = append(seen, c.Text())
			deliver([]widgets.Widget{widgets.NewText("only suggestion")})
		}, SearchAnchorOpts{ViewHintText: foundation.Some("Find")}))
		ts.Tap(gluttertest.ByIcon(Icons.Search()))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("only suggestion")) || !ts.Exists(gluttertest.ByText("Find")) {
			t.Error("the view does not show the suggestion and the hint")
		}
		if len(seen) == 0 {
			t.Error("the suggestions were not asked for")
		}
		// The back button closes the view.
		ts.Tap(gluttertest.ByIcon(Icons.ArrowBack()))
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByText("only suggestion")) {
			t.Error("the view is still open after the back button")
		}
	})
	t.Run("a tap outside closes the view", func(t *testing.T) {
		controller := NewSearchController()
		defer controller.Dispose()
		ts := pumpSearchAnchor(t, linuxTheme(), NewSearchAnchorBar(fruitSuggestions, SearchAnchorBarOpts{SearchController: controller}))
		ts.Tap(gluttertest.ByType[*SearchBar]())
		ts.PumpAndSettle()
		ts.TapAt(ui.Offset{Dx: 780, Dy: 580})
		ts.PumpAndSettle()
		if controller.IsOpen() {
			t.Error("the view is still open")
		}
	})
	t.Run("a view builder", func(t *testing.T) {
		ts := pumpSearchAnchor(t, linuxTheme(), NewSearchAnchorBar(fruitSuggestions, SearchAnchorBarOpts{
			ViewBuilder: func(s []widgets.Widget) widgets.Widget {
				return widgets.NewText("count " + string(rune('0'+len(s))))
			}}))
		ts.Tap(gluttertest.ByType[*SearchBar]())
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByText("count 5")) {
			t.Error("the view is not the one that the view builder made")
		}
	})
	t.Run("shrink wrap", func(t *testing.T) {
		controller := NewSearchController()
		defer controller.Dispose()
		ts := pumpSearchAnchor(t, linuxTheme(), NewSearchAnchorBar(func(_ widgets.BuildContext, c *SearchController, deliver func([]widgets.Widget)) {
			if c.Text() == "" {
				deliver(nil)
				return
			}
			deliver([]widgets.Widget{widgets.NewSizedBox(widgets.NewText("one"), widgets.SizedBoxOpts{Height: foundation.Some(48.0)})})
		}, SearchAnchorBarOpts{SearchController: controller, ShrinkWrap: foundation.Some(true),
			ViewConstraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MinWidth: foundation.Some(360.0)}))}))
		ts.Tap(gluttertest.ByType[*SearchBar]())
		ts.PumpAndSettle()
		view := gluttertest.First(gluttertest.Ancestor(gluttertest.Last(gluttertest.ByType[*SearchBar]()), gluttertest.ByType[*Material]()))
		empty := ts.Size(view).Height
		ts.EnterText(gluttertest.Last(gluttertest.ByType[*SearchBar]()), "x")
		ts.PumpAndSettle()
		if full := ts.Size(view).Height; full <= empty {
			t.Errorf("the view is %v high with a suggestion and %v without: it does not shrink wrap", full, empty)
		}
	})
	t.Run("the anchor fades while the view is open", func(t *testing.T) {
		ts := pumpSearchAnchor(t, linuxTheme(), NewSearchAnchorBar(fruitSuggestions))
		anchor := gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*SearchAnchor](), gluttertest.ByType[*widgets.AnimatedOpacity]()))
		if got := gluttertest.WidgetAs[*widgets.AnimatedOpacity](ts, anchor).Opacity(); got != 1 {
			t.Errorf("opacity = %v, want 1", got)
		}
		ts.Tap(gluttertest.ByType[*SearchBar]())
		ts.PumpAndSettle()
		if got := gluttertest.WidgetAs[*widgets.AnimatedOpacity](ts, anchor).Opacity(); got != 0 {
			t.Errorf("opacity with the view open = %v, want 0", got)
		}
	})
	t.Run("a disabled anchor", func(t *testing.T) {
		ts := pumpSearchAnchor(t, linuxTheme(), NewSearchAnchorBar(fruitSuggestions, SearchAnchorBarOpts{Disabled: true}))
		ts.Tap(gluttertest.ByType[*SearchBar]())
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByIcon(Icons.ArrowBack())) {
			t.Error("a disabled anchor opened the view")
		}
	})
}

func TestSearchControllerErrors(t *testing.T) {
	skipUnlessDebug(t)
	c := NewSearchController()
	defer c.Dispose()
	if c.IsAttached() {
		t.Error("a new controller is attached")
	}
	tests := []struct {
		name string
		run  func()
	}{
		{"is open", func() { c.IsOpen() }},
		{"open view", func() { c.OpenView() }},
		{"close view", func() { c.CloseView(foundation.None[string]()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.run), "SearchControllerNotAttached")
		})
	}
}

func TestSearchThemeData(t *testing.T) {
	states := widgets.NewWidgetStates()
	t.Run("search bar", func(t *testing.T) {
		a := NewSearchBarThemeData(SearchBarThemeDataOpts{Elevation: allOpt(2.0), BackgroundColor: allOpt(colorA), TextCapitalization: services.TextCapitalizationWords})
		b := NewSearchBarThemeData(SearchBarThemeDataOpts{Elevation: allOpt(6.0), BackgroundColor: allOpt(colorB)})
		if !a.Equal(a.CopyWith(SearchBarThemeDataCopy{})) || a.Equal(b) {
			t.Error("equality is wrong")
		}
		got := SearchBarThemeDataLerp(a, b, 0.5)
		if got.Elevation.Resolve(states) != foundation.Some(4.0) {
			t.Errorf("elevation = %v, want 4", got.Elevation.Resolve(states))
		}
		if got := SearchBarThemeDataLerp(a, b, 0.25); got.TextCapitalization != services.TextCapitalizationWords {
			t.Errorf("capitalization at 0.25 = %v", got.TextCapitalization)
		}
	})
	t.Run("search view", func(t *testing.T) {
		a := NewSearchViewThemeData(SearchViewThemeDataOpts{Elevation: foundation.Some(2.0), HeaderHeight: foundation.Some(40.0), ShrinkWrap: foundation.Some(true),
			HeaderTextStyle: painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(10.0)})})
		b := NewSearchViewThemeData(SearchViewThemeDataOpts{Elevation: foundation.Some(6.0), HeaderHeight: foundation.Some(80.0),
			HeaderTextStyle: painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(30.0)})})
		if !a.Equal(a.CopyWith(SearchViewThemeDataCopy{})) || a.Equal(b) {
			t.Error("equality is wrong")
		}
		got := SearchViewThemeDataLerp(a, b, 0.5)
		if got.Elevation != foundation.Some(4.0) || got.HeaderHeight != foundation.Some(60.0) {
			t.Errorf("lerp = %v", got)
		}
		if got := SearchViewThemeDataLerp(a, b, 0.25); got.ShrinkWrap != foundation.Some(true) {
			t.Errorf("shrinkWrap at 0.25 = %v", got.ShrinkWrap)
		}
		// Flutter's lerp of the hint style interpolates the text styles of the header.
		if got.HeaderHintStyle == nil || got.HeaderHintStyle.FontSize != foundation.Some(20.0) {
			t.Errorf("header hint style = %v, want the lerp of the header text styles", got.HeaderHintStyle)
		}
	})
}

func TestSearchViewTheme(t *testing.T) {
	// The theme of the view gives the colors and the header height of the view.
	viewTheme := NewSearchViewTheme(NewSearchViewThemeData(SearchViewThemeDataOpts{
		BackgroundColor: foundation.Some(ui.NewColor(0xFFFF0000)), HeaderHeight: foundation.Some(100.0)}), NewSearchAnchorBar(fruitSuggestions))
	ts := pumpSearchAnchor(t, linuxTheme(), viewTheme)
	ts.Tap(gluttertest.ByType[*SearchBar]())
	ts.PumpAndSettle()
	view := gluttertest.WidgetAs[*Material](ts, gluttertest.First(gluttertest.Ancestor(gluttertest.Last(gluttertest.ByType[*SearchBar]()), gluttertest.ByType[*Material]())))
	if view.Color() != foundation.Some(ui.NewColor(0xFFFF0000)) {
		t.Errorf("view color = %v, want red", view.Color())
	}
	if h := ts.Size(gluttertest.Last(gluttertest.ByType[*SearchBar]())).Height; h != 100 {
		t.Errorf("the header is %v high, want 100", h)
	}
}
