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
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/material/scrollbar_test.dart. The tests read
// what the scrollbar gives its painter instead of using the `paints` matcher.

func pumpScrollbar(t *testing.T, theme *ThemeData, opts ScrollbarOpts) (*gluttertest.Tester, *widgets.RawScrollbarState) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	controller := widgets.NewScrollController()
	t.Cleanup(controller.Dispose)
	opts.Controller = controller
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: widgets.NewMediaQuery(widgets.MediaQueryData{Size: ui.Size{Width: 800, Height: 600}},
		// The scroll behavior of the app would add a scrollbar of its own.
		widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			return widgets.NewScrollConfiguration(widgets.ScrollConfigurationOf(bc).CopyWith(widgets.ScrollBehaviorCopy{Scrollbars: foundation.Some(false)}),
				NewScrollbar(widgets.NewSingleChildScrollView(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(4000.0)}),
					widgets.SingleChildScrollViewOpts{Controller: controller}), opts))
		}))}))
	ts.PumpAndSettle()
	return ts, gluttertest.StateOf[*widgets.RawScrollbarState](ts, gluttertest.ByType[*widgets.RawScrollbar]())
}

func TestScrollbarPainterValues(t *testing.T) {
	colors := func(brightness ui.Brightness) *ColorScheme {
		return ColorSchemeFromSeed(ui.NewColor(0xFF2196F3), ColorSchemeFromSeedOpts{Brightness: brightness})
	}
	tests := []struct {
		name      string
		theme     *ThemeData
		opts      ScrollbarOpts
		color     func(cs *ColorScheme, th *ThemeData) ui.Color
		thickness float64
		radius    foundation.Opt[ui.Radius]
		margin    float64
	}{
		{"light, linux", NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux), ColorScheme: colors(ui.BrightnessLight)}), ScrollbarOpts{ThumbVisibility: foundation.Some(true)},
			func(cs *ColorScheme, _ *ThemeData) ui.Color { return cs.OnSurface().WithOpacity(0.1) }, 8, foundation.Some(ui.NewRadiusCircular(8)), 2},
		{"dark, linux", NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux), ColorScheme: colors(ui.BrightnessDark)}), ScrollbarOpts{ThumbVisibility: foundation.Some(true)},
			func(cs *ColorScheme, _ *ThemeData) ui.Color { return cs.OnSurface().WithOpacity(0.3) }, 8, foundation.Some(ui.NewRadiusCircular(8)), 2},
		{"android", NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformAndroid)}), ScrollbarOpts{ThumbVisibility: foundation.Some(true)},
			func(_ *ColorScheme, th *ThemeData) ui.Color { return th.HighlightColor.WithOpacity(1.0) }, 4, foundation.None[ui.Radius](), 0},
		{"thickness and radius", NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux)}),
			ScrollbarOpts{ThumbVisibility: foundation.Some(true), Thickness: foundation.Some(20.0), Radius: foundation.Some(ui.NewRadiusCircular(3))},
			func(cs *ColorScheme, _ *ThemeData) ui.Color { return cs.OnSurface().WithOpacity(0.1) }, 20, foundation.Some(ui.NewRadiusCircular(3)), 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, state := pumpScrollbar(t, tc.theme, tc.opts)
			p := state.ScrollbarPainter()
			scheme := tc.theme.ColorScheme
			if got, want := p.Color(), tc.color(scheme, tc.theme); got != want {
				t.Errorf("color %v, want %v", got, want)
			}
			if p.Thickness() != tc.thickness {
				t.Errorf("thickness %v, want %v", p.Thickness(), tc.thickness)
			}
			if p.Radius() != tc.radius {
				t.Errorf("radius %v, want %v", p.Radius(), tc.radius)
			}
			if p.CrossAxisMargin() != tc.margin {
				t.Errorf("cross axis margin %v, want %v", p.CrossAxisMargin(), tc.margin)
			}
			if p.MinLength() != 48 {
				t.Errorf("min length %v, want 48", p.MinLength())
			}
		})
	}
}

func TestScrollbarThemeOverrides(t *testing.T) {
	thumb, track, trackBorder := ui.NewColor(0xFF112233), ui.NewColor(0xFF445566), ui.NewColor(0xFF778899)
	theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux), ScrollbarTheme: ScrollbarThemeData{
		ThumbVisibility:  widgets.NewWidgetStatePropertyAll(foundation.Some(true)),
		TrackVisibility:  widgets.NewWidgetStatePropertyAll(foundation.Some(true)),
		Thickness:        widgets.NewWidgetStatePropertyAll(foundation.Some(30.0)),
		ThumbColor:       widgets.NewWidgetStatePropertyAll(foundation.Some(thumb)),
		TrackColor:       widgets.NewWidgetStatePropertyAll(foundation.Some(track)),
		TrackBorderColor: widgets.NewWidgetStatePropertyAll(foundation.Some(trackBorder)),
		Radius:           foundation.Some(ui.NewRadiusCircular(1)),
		CrossAxisMargin:  foundation.Some(5.0),
		MainAxisMargin:   foundation.Some(7.0),
		MinThumbLength:   foundation.Some(100.0),
	}})
	_, state := pumpScrollbar(t, theme, ScrollbarOpts{})
	p := state.ScrollbarPainter()
	if p.Color() != thumb || p.TrackColor() != track || p.TrackBorderColor() != trackBorder {
		t.Errorf("colors %v %v %v", p.Color(), p.TrackColor(), p.TrackBorderColor())
	}
	if p.Thickness() != 30 || p.CrossAxisMargin() != 5 || p.MainAxisMargin() != 7 || p.MinLength() != 100 || p.Radius() != foundation.Some(ui.NewRadiusCircular(1)) {
		t.Errorf("thickness %v margins %v %v min %v radius %v", p.Thickness(), p.CrossAxisMargin(), p.MainAxisMargin(), p.MinLength(), p.Radius())
	}
	// The options of the widget win over the theme.
	_, state = pumpScrollbar(t, theme, ScrollbarOpts{Thickness: foundation.Some(11.0), TrackVisibility: foundation.Some(false)})
	if got := state.ScrollbarPainter().Thickness(); got != 11 {
		t.Errorf("thickness %v, the option says 11", got)
	}
	if got := state.ScrollbarPainter().TrackColor(); got != ui.NewColor(0x00000000) {
		t.Errorf("track color %v, the option hides the track", got)
	}
}

func TestScrollbarHoverAndDrag(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux)})
	ts, state := pumpScrollbar(t, theme, ScrollbarOpts{ThumbVisibility: foundation.Some(true)})
	onSurface := theme.ColorScheme.OnSurface()
	idle := state.ScrollbarPainter().Color()
	if idle != onSurface.WithOpacity(0.1) {
		t.Fatalf("idle color %v", idle)
	}
	// The thumb is at the top right of the view, 8 wide with a margin of 2.
	over := ui.Offset{Dx: 800 - 2 - 4, Dy: 30}
	ts.Hover(over)
	ts.PumpAndSettle()
	if got := state.ScrollbarPainter().Color(); got != onSurface.WithOpacity(0.5) {
		t.Errorf("hover color %v, want %v", got, onSurface.WithOpacity(0.5))
	}
	ts.Hover(ui.Offset{Dx: 100, Dy: 300})
	ts.PumpAndSettle()
	if got := state.ScrollbarPainter().Color(); got != idle {
		t.Errorf("the thumb did not go back to %v: %v", idle, got)
	}
	// Dragging the thumb scrolls the view and darkens the thumb.
	g := ts.StartGesture(over)
	g.MoveBy(ui.Offset{Dx: 0, Dy: 60})
	ts.Pump()
	if got := state.ScrollbarPainter().Color(); got != onSurface.WithOpacity(0.6) {
		t.Errorf("drag color %v, want %v", got, onSurface.WithOpacity(0.6))
	}
	if offset := gluttertest.StateOf[*widgets.ScrollableState](ts, gluttertest.ByType[*widgets.Scrollable]()).Position().Pixels(); offset <= 0 {
		t.Errorf("the view did not scroll: %v", offset)
	}
	g.Up()
	ts.PumpAndSettle()
}

func TestScrollbarThumbVisibilityFromTheme(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux)})
	ts, state := pumpScrollbar(t, theme, ScrollbarOpts{})
	// A thumb that is not always visible has faded out.
	_ = state
	ts.Pump(2 * time.Second)
	if state.ShowScrollbar() {
		t.Error("the thumb is always visible by default")
	}
}

func TestScrollbarBehaviorOnDesktop(t *testing.T) {
	tests := []struct {
		platform foundation.TargetPlatform
		want     bool
	}{
		{foundation.TargetPlatformLinux, true}, {foundation.TargetPlatformWindows, true}, {foundation.TargetPlatformMacOS, true},
		{foundation.TargetPlatformAndroid, false}, {foundation.TargetPlatformFuchsia, false},
	}
	for _, tc := range tests {
		t.Run(tc.platform.String(), func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{Platform: foundation.Some(tc.platform)}),
				Home: NewScaffold(ScaffoldOpts{Body: widgets.NewListView([]widgets.Widget{widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(2000.0)})})})}))
			ts.PumpAndSettle()
			if got := ts.Exists(gluttertest.ByType[*Scrollbar]()); got != tc.want {
				t.Errorf("a Scrollbar is built: %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScrollbarThemeData(t *testing.T) {
	a := ScrollbarThemeData{Thickness: widgets.NewWidgetStatePropertyAll(foundation.Some(2.0)), MinThumbLength: foundation.Some(10.0), CrossAxisMargin: foundation.Some(1.0)}
	b := ScrollbarThemeData{Thickness: widgets.NewWidgetStatePropertyAll(foundation.Some(4.0)), MinThumbLength: foundation.Some(30.0), CrossAxisMargin: foundation.Some(3.0)}
	got := ScrollbarThemeDataLerp(a, b, 0.5)
	if got.MinThumbLength != foundation.Some(20.0) || got.CrossAxisMargin != foundation.Some(2.0) {
		t.Errorf("lerp %v %v", got.MinThumbLength, got.CrossAxisMargin)
	}
	if v := got.Thickness.Resolve(0).OrElse(0); v != 3 {
		t.Errorf("thickness lerp %v, want 3", v)
	}
	if !a.Equal(a) || a.Equal(b) {
		t.Error("Equal")
	}
	if c := a.CopyWith(ScrollbarThemeDataCopy{MinThumbLength: foundation.Some(99.0)}); c.MinThumbLength != foundation.Some(99.0) || c.CrossAxisMargin != a.CrossAxisMargin {
		t.Errorf("CopyWith %v", c)
	}
}

func TestScrollbarErrors(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	behavior := NewMaterialScrollBehavior()
	wantRoute(t, recoverError(func() {
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux)}),
			Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				return behavior.BuildScrollbar(bc, widgets.NewSizedBox(nil), widgets.ScrollableDetails{Direction: painting.AxisDirectionDown})
			})}))
		takeAndPanic(ts)
	}), "MaterialScrollbarControllerMissing")
}
