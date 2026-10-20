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
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func iconFor(name string) widgets.Widget { return NewIcon(Icons.Favorite()) }

var iconButtonVariants = []struct {
	name string
	make func(onPressed func(), icon widgets.Widget, o IconButtonOpts) *IconButton
}{
	{"standard", func(p func(), i widgets.Widget, o IconButtonOpts) *IconButton { return NewIconButton(p, i, o) }},
	{"filled", func(p func(), i widgets.Widget, o IconButtonOpts) *IconButton { return NewIconButtonFilled(p, i, o) }},
	{"filledTonal", func(p func(), i widgets.Widget, o IconButtonOpts) *IconButton {
		return NewIconButtonFilledTonal(p, i, o)
	}},
	{"outlined", func(p func(), i widgets.Widget, o IconButtonOpts) *IconButton { return NewIconButtonOutlined(p, i, o) }},
}

func TestIconButtonTapsAndTooltip(t *testing.T) {
	for _, v := range iconButtonVariants {
		t.Run(v.name, func(t *testing.T) {
			taps, longs := 0, 0
			ts := pumpButton(t, standardTheme(true), v.make(func() { taps++ }, iconFor(v.name), IconButtonOpts{OnLongPress: func() { longs++ }, Tooltip: "Like"}))
			ts.Tap(gluttertest.ByType[*IconButton]())
			ts.Pump()
			ts.LongPress(gluttertest.ByType[*IconButton]())
			ts.PumpAndSettle()
			if taps != 1 || longs != 1 {
				t.Errorf("taps = %d, long presses = %d, want 1 and 1", taps, longs)
			}
			if got := ts.Size(gluttertest.ByType[*IconButton]()); got != (ui.Size{Width: 48, Height: 48}) && got != (ui.Size{Width: 40, Height: 40}) {
				t.Errorf("size = %v, want 40x40 (the tap target of a padded theme is 48x48)", got)
			}

			// A disabled button has no long press either.
			longs = 0
			ts = pumpButton(t, standardTheme(true), v.make(nil, iconFor(v.name), IconButtonOpts{OnLongPress: func() { longs++ }}))
			ts.LongPress(gluttertest.ByType[*IconButton]())
			ts.PumpAndSettle()
			if longs != 0 {
				t.Error("the disabled button was long pressed")
			}
		})
	}
}

func TestIconButtonVariantsColorsMaterial3(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	tests := []struct {
		name       string
		variant    int
		selected   foundation.Opt[bool]
		disabled   bool
		wantBg     foundation.Opt[ui.Color]
		wantFg     ui.Color
		wantBorder bool
	}{
		{"standard", 0, foundation.None[bool](), false, foundation.Some(Colors.Transparent), cs.OnSurfaceVariant(), false},
		{"standard selected", 0, foundation.Some(true), false, foundation.Some(Colors.Transparent), cs.Primary(), false},
		{"standard disabled", 0, foundation.None[bool](), true, foundation.Some(Colors.Transparent), cs.OnSurface().WithOpacity(0.38), false},
		{"filled", 1, foundation.None[bool](), false, foundation.Some(cs.Primary()), cs.OnPrimary(), false},
		{"filled toggle unselected", 1, foundation.Some(false), false, foundation.Some(cs.SurfaceContainerHighest()), cs.Primary(), false},
		{"filled toggle selected", 1, foundation.Some(true), false, foundation.Some(cs.Primary()), cs.OnPrimary(), false},
		{"filled disabled", 1, foundation.None[bool](), true, foundation.Some(cs.OnSurface().WithOpacity(0.12)), cs.OnSurface().WithOpacity(0.38), false},
		{"tonal", 2, foundation.None[bool](), false, foundation.Some(cs.SecondaryContainer()), cs.OnSecondaryContainer(), false},
		{"tonal toggle unselected", 2, foundation.Some(false), false, foundation.Some(cs.SurfaceContainerHighest()), cs.OnSurfaceVariant(), false},
		{"outlined", 3, foundation.None[bool](), false, foundation.Some(Colors.Transparent), cs.OnSurfaceVariant(), true},
		{"outlined selected", 3, foundation.Some(true), false, foundation.Some(cs.InverseSurface()), cs.OnInverseSurface(), false},
		{"outlined disabled", 3, foundation.None[bool](), true, foundation.Some(Colors.Transparent), cs.OnSurface().WithOpacity(0.38), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var onPressed func()
			if !tc.disabled {
				onPressed = func() {}
			}
			v := iconButtonVariants[tc.variant]
			ts := pumpButton(t, theme, v.make(onPressed, iconFor(v.name), IconButtonOpts{IsSelected: tc.selected}))
			mat := buttonMaterial(ts, gluttertest.ByType[*IconButton]())
			if mat.Color() != tc.wantBg {
				t.Errorf("background = %v, want %v", mat.Color(), tc.wantBg)
			}
			icon := gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*widgets.Icon]()))
			it := widgets.IconThemeOf(ts.Element(icon))
			if it.Color() != foundation.Some(tc.wantFg) {
				t.Errorf("icon color = %v, want %v", it.Color(), tc.wantFg)
			}
			if it.Size() != foundation.Some(24.0) {
				t.Errorf("icon size = %v, want 24", it.Size())
			}
			side := mat.Shape().(painting.OutlinedBorder).Side()
			if hasBorder := side.Style == painting.BorderStyleSolid && side.Width > 0; hasBorder != tc.wantBorder {
				t.Errorf("border = %v, want %v (side %v)", hasBorder, tc.wantBorder, side)
			}
		})
	}
}

func TestIconButtonSelectedIcon(t *testing.T) {
	selectedIcon, normalIcon := NewIcon(Icons.Favorite()), NewIcon(Icons.FavoriteBorder())
	tests := []struct {
		name     string
		selected foundation.Opt[bool]
		selIcon  widgets.Widget
		want     widgets.Widget
	}{
		{"not a toggle", foundation.None[bool](), selectedIcon, normalIcon},
		{"unselected", foundation.Some(false), selectedIcon, normalIcon},
		{"selected", foundation.Some(true), selectedIcon, selectedIcon},
		{"selected without selected icon", foundation.Some(true), nil, normalIcon},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, nil, NewIconButton(func() {}, normalIcon, IconButtonOpts{IsSelected: tc.selected, SelectedIcon: tc.selIcon}))
			got := gluttertest.WidgetAs[*widgets.Icon](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*widgets.Icon]())))
			if got != tc.want.(*widgets.Icon) {
				t.Errorf("shown icon %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIconButtonToggleStatesController(t *testing.T) {
	controller := widgets.NewWidgetStatesController()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	build := func(selected bool) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewIconButton(func() {}, iconFor(""), IconButtonOpts{IsSelected: foundation.Some(selected), StatesController: controller}))})})
	}
	ts.PumpWidget(build(false))
	if controller.Value().Has(widgets.WidgetStateSelected) {
		t.Error("an unselected button has the selected state")
	}
	ts.PumpWidget(build(true))
	if !controller.Value().Has(widgets.WidgetStateSelected) {
		t.Error("a selected button does not have the selected state")
	}
	ts.PumpWidget(build(false))
	if controller.Value().Has(widgets.WidgetStateSelected) {
		t.Error("the selected state was not removed")
	}
}

func TestIconButtonOptions(t *testing.T) {
	tests := []struct {
		name string
		opts IconButtonOpts
		want ui.Size
	}{
		{"default", IconButtonOpts{}, ui.Size{Width: 40, Height: 40}},
		{"icon size and padding", IconButtonOpts{IconSize: foundation.Some(40.0), Padding: painting.NewEdgeInsetsAll(10)}, ui.Size{Width: 60, Height: 60}},
		{"constraints", IconButtonOpts{Constraints: foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(80.0), Height: foundation.Some(70.0)}))}, ui.Size{Width: 80, Height: 70}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(true), NewIconButton(func() {}, iconFor(""), tc.opts))
			got := ts.Size(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*Material]())))
			if got != tc.want {
				t.Errorf("size = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIconButtonStyleFromAndThemeOrder(t *testing.T) {
	red, green := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00)
	theme := NewThemeData(ThemeDataOpts{VisualDensity: foundation.Some(VisualDensityStandard()),
		IconButtonTheme: IconButtonThemeData{Style: IconButtonStyleFrom(IconButtonStyleFromOpts{ForegroundColor: foundation.Some(green), IconSize: foundation.Some(30.0)})}})
	// The theme gives the foreground and the size; the argument Color wins over it.
	ts := pumpButton(t, theme, NewIconButton(func() {}, iconFor(""), IconButtonOpts{Color: foundation.Some(red)}))
	it := widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*widgets.Icon]()))))
	if it.Color() != foundation.Some(red) || it.Size() != foundation.Some(30.0) {
		t.Errorf("icon theme = %v, want color red and size 30", it)
	}
	ts = pumpButton(t, theme, NewIconButton(func() {}, iconFor("")))
	it = widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*widgets.Icon]()))))
	if it.Color() != foundation.Some(green) {
		t.Errorf("icon color = %v, want the one of the theme", it.Color())
	}

	// The style of the widget wins over the arguments and the theme.
	style := IconButtonStyleFrom(IconButtonStyleFromOpts{ForegroundColor: foundation.Some(red)})
	ts = pumpButton(t, theme, NewIconButton(func() {}, iconFor(""), IconButtonOpts{Style: style, Color: foundation.Some(green)}))
	it = widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*widgets.Icon]()))))
	if it.Color() != foundation.Some(red) {
		t.Errorf("icon color = %v, want the one of the style", it.Color())
	}
}

func TestIconButtonStyleFromOverlay(t *testing.T) {
	blue := ui.NewColor(0xFF0000FF)
	tests := []struct {
		name    string
		opts    IconButtonStyleFromOpts
		states  widgets.WidgetStates
		want    foundation.Opt[ui.Color]
		wantNil bool
	}{
		{"no colors has no overlay", IconButtonStyleFromOpts{}, widgets.NewWidgetStates(widgets.WidgetStateHovered), foundation.None[ui.Color](), true},
		{"foreground makes the hover", IconButtonStyleFromOpts{ForegroundColor: foundation.Some(blue)}, widgets.NewWidgetStates(widgets.WidgetStateHovered), foundation.Some(blue.WithOpacity(0.08)), false},
		{"foreground makes the press", IconButtonStyleFromOpts{ForegroundColor: foundation.Some(blue)}, widgets.NewWidgetStates(widgets.WidgetStatePressed), foundation.Some(blue.WithOpacity(0.1)), false},
		{"hover color", IconButtonStyleFromOpts{HoverColor: foundation.Some(blue)}, widgets.NewWidgetStates(widgets.WidgetStateHovered), foundation.Some(blue), false},
		{"transparent overlay turns it off", IconButtonStyleFromOpts{ForegroundColor: foundation.Some(blue), OverlayColor: foundation.Some(Colors.Transparent)},
			widgets.NewWidgetStates(widgets.WidgetStateHovered), foundation.Some(Colors.Transparent), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := IconButtonStyleFrom(tc.opts)
			if tc.wantNil {
				if s.OverlayColor != nil {
					t.Errorf("overlay = %v, want none", s.OverlayColor)
				}
				return
			}
			if got := s.OverlayColor.Resolve(tc.states); got != tc.want {
				t.Errorf("overlay = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIconButtonMaterial2(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(false), VisualDensity: foundation.Some(VisualDensityStandard())})
	taps := 0
	ts := pumpButton(t, theme, NewIconButton(func() { taps++ }, iconFor(""), IconButtonOpts{Tooltip: "M2"}))
	if got := ts.Size(gluttertest.ByType[*widgets.Icon]()); got != (ui.Size{Width: 24, Height: 24}) {
		t.Errorf("icon = %v, want 24x24", got)
	}
	if got := ts.Size(gluttertest.ByType[*IconButton]()); got != (ui.Size{Width: 48, Height: 48}) {
		t.Errorf("size = %v, want 48x48", got)
	}
	ts.Tap(gluttertest.ByType[*IconButton]())
	ts.Pump()
	if taps != 1 {
		t.Errorf("taps = %d", taps)
	}
	// The disabled button has the disabled color of the theme.
	ts = pumpButton(t, theme, NewIconButton(nil, iconFor("")))
	it := widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*widgets.Icon]()))))
	if it.Color() != foundation.Some(theme.DisabledColor) {
		t.Errorf("disabled icon color = %v, want %v", it.Color(), theme.DisabledColor)
	}
}

func TestIconButtonErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		build func()
		route string
	}{
		{"zero splash radius", func() { NewIconButton(nil, nil, IconButtonOpts{SplashRadius: foundation.Some(0.0)}) }, "IconButtonSplashRadius"},
		{"negative splash radius", func() { NewIconButtonFilled(nil, nil, IconButtonOpts{SplashRadius: foundation.Some(-4.0)}) }, "IconButtonSplashRadius"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.build), tc.route)
		})
	}
}
