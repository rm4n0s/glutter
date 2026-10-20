// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func fabIcon() widgets.Widget { return NewIcon(Icons.Add()) }

func TestFloatingActionButtonSizesAndShapes(t *testing.T) {
	noHero := FloatingActionButtonOpts{NoHero: true}
	tests := []struct {
		name       string
		material3  bool
		make       func() *FloatingActionButton
		wantSize   ui.Size
		wantRadius float64 // 0 for a circle or a stadium
		wantShape  string
	}{
		{"M3 regular", true, func() *FloatingActionButton { return NewFloatingActionButton(func() {}, fabIcon(), noHero) }, ui.Size{Width: 56, Height: 56}, 16, "rrect"},
		{"M3 small", true, func() *FloatingActionButton { return NewFloatingActionButtonSmall(func() {}, fabIcon(), noHero) }, ui.Size{Width: 40, Height: 40}, 12, "rrect"},
		{"M3 mini argument", true, func() *FloatingActionButton {
			return NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true, Mini: true})
		}, ui.Size{Width: 40, Height: 40}, 12, "rrect"},
		{"M3 large", true, func() *FloatingActionButton { return NewFloatingActionButtonLarge(func() {}, fabIcon(), noHero) }, ui.Size{Width: 96, Height: 96}, 28, "rrect"},
		{"M2 regular", false, func() *FloatingActionButton { return NewFloatingActionButton(func() {}, fabIcon(), noHero) }, ui.Size{Width: 56, Height: 56}, 0, "circle"},
		{"M2 large", false, func() *FloatingActionButton { return NewFloatingActionButtonLarge(func() {}, fabIcon(), noHero) }, ui.Size{Width: 96, Height: 96}, 0, "circle"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(tc.material3), tc.make())
			fab := gluttertest.ByType[*FloatingActionButton]()
			mat := buttonMaterial(ts, fab)
			if got := ts.Size(gluttertest.First(gluttertest.Descendant(fab, gluttertest.ByType[*Material]()))); got != tc.wantSize {
				t.Errorf("size = %v, want %v", got, tc.wantSize)
			}
			switch tc.wantShape {
			case "circle":
				if _, ok := mat.Shape().(painting.CircleBorder); !ok {
					t.Errorf("shape = %T, want a circle", mat.Shape())
				}
			default:
				rr, ok := mat.Shape().(painting.RoundedRectangleBorder)
				if !ok || !rr.BorderRadius.Equal(painting.NewBorderRadiusAll(ui.NewRadiusCircular(tc.wantRadius))) {
					t.Errorf("shape = %v, want a rounded rectangle of radius %v", mat.Shape(), tc.wantRadius)
				}
			}
		})
	}
}

func TestFloatingActionButtonColors(t *testing.T) {
	m3, m2 := standardTheme(true), standardTheme(false)
	tests := []struct {
		name     string
		theme    *ThemeData
		wantFill ui.Color
		wantFg   ui.Color
	}{
		{"M3", m3, m3.ColorScheme.PrimaryContainer(), m3.ColorScheme.OnPrimaryContainer()},
		{"M2", m2, m2.ColorScheme.Secondary(), m2.ColorScheme.OnSecondary()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, tc.theme, NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true}))
			mat := buttonMaterial(ts, gluttertest.ByType[*FloatingActionButton]())
			if mat.Color() != foundation.Some(tc.wantFill) {
				t.Errorf("fill = %v, want %v", mat.Color(), tc.wantFill)
			}
			if got := mat.TextStyle().Color; got != foundation.Some(tc.wantFg) {
				t.Errorf("foreground = %v, want %v", got, tc.wantFg)
			}
			it := widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*FloatingActionButton](), gluttertest.ByType[*widgets.Icon]()))))
			if it.Color() != foundation.Some(tc.wantFg) || it.Size() != foundation.Some(24.0) {
				t.Errorf("icon theme = %v, want size 24 and color %v", it, tc.wantFg)
			}
		})
	}
}

func TestFloatingActionButtonThemeOrder(t *testing.T) {
	red, green := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00)
	theme := NewThemeData(ThemeDataOpts{VisualDensity: foundation.Some(VisualDensityStandard()),
		FloatingActionButtonTheme: FloatingActionButtonThemeData{
			BackgroundColor: foundation.Some(green), ForegroundColor: foundation.Some(red), Elevation: foundation.Some(3.0),
			SizeConstraints: foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(70.0), Height: foundation.Some(70.0)})),
			Shape:           painting.NewCircleBorder(), IconSize: foundation.Some(30.0)}})
	ts := pumpButton(t, theme, NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true, BackgroundColor: foundation.Some(red)}))
	fab := gluttertest.ByType[*FloatingActionButton]()
	mat := buttonMaterial(ts, fab)
	if mat.Color() != foundation.Some(red) {
		t.Errorf("fill = %v, want the one of the widget", mat.Color())
	}
	if got := mat.TextStyle().Color; got != foundation.Some(red) {
		t.Errorf("foreground = %v, want the one of the theme", got)
	}
	if mat.Elevation() != 3 {
		t.Errorf("elevation = %v, want 3", mat.Elevation())
	}
	if got := ts.Size(gluttertest.First(gluttertest.Descendant(fab, gluttertest.ByType[*Material]()))); got != (ui.Size{Width: 70, Height: 70}) {
		t.Errorf("size = %v, want 70x70", got)
	}
	if _, ok := mat.Shape().(painting.CircleBorder); !ok {
		t.Errorf("shape = %T, want a circle", mat.Shape())
	}
	it := widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(fab, gluttertest.ByType[*widgets.Icon]()))))
	if it.Size() != foundation.Some(30.0) {
		t.Errorf("icon size = %v, want 30", it.Size())
	}

	// A FloatingActionButtonTheme above the button wins over the theme of the app.
	ts = pumpButton(t, theme, NewFloatingActionButtonTheme(FloatingActionButtonThemeData{Elevation: foundation.Some(9.0)},
		NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true})))
	if got := buttonMaterial(ts, gluttertest.ByType[*FloatingActionButton]()).Elevation(); got != 9 {
		t.Errorf("elevation = %v, want 9", got)
	}
}

func TestFloatingActionButtonElevationFollowsStates(t *testing.T) {
	for _, material3 := range []bool{true, false} {
		name := "M2"
		wantPressed := 12.0
		if material3 {
			name, wantPressed = "M3", 6.0
		}
		t.Run(name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(material3), NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true}))
			fab := gluttertest.ByType[*FloatingActionButton]()
			elevation := func() float64 { return buttonMaterial(ts, fab).Elevation() }
			if got := elevation(); got != 6 {
				t.Fatalf("elevation at rest = %v, want 6", got)
			}
			center := ts.Center(fab)
			ts.Hover(center)
			ts.PumpAndSettle()
			if got := elevation(); got != 8 {
				t.Errorf("hovered elevation = %v, want 8", got)
			}
			g := ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
			ts.PumpAndSettle()
			if got := elevation(); got != wantPressed {
				t.Errorf("pressed elevation = %v, want %v", got, wantPressed)
			}
			g.Up()
			ts.PumpAndSettle()
		})
	}
	t.Run("disabled", func(t *testing.T) {
		tests := []struct {
			name string
			opts FloatingActionButtonOpts
			want float64
		}{
			{"the elevation", FloatingActionButtonOpts{NoHero: true}, 6},
			{"given elevation", FloatingActionButtonOpts{NoHero: true, Elevation: foundation.Some(3.0)}, 3},
			{"disabled elevation", FloatingActionButtonOpts{NoHero: true, DisabledElevation: foundation.Some(1.0)}, 1},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				ts := pumpButton(t, standardTheme(true), NewFloatingActionButton(nil, fabIcon(), tc.opts))
				if got := buttonMaterial(ts, gluttertest.ByType[*FloatingActionButton]()).Elevation(); got != tc.want {
					t.Errorf("elevation = %v, want %v", got, tc.want)
				}
			})
		}
	})
}

func TestFloatingActionButtonExtended(t *testing.T) {
	tests := []struct {
		name     string
		material bool
		opts     FloatingActionButtonExtendedOpts
		wantH    float64
		wantW    float64
		hasLabel bool
	}{
		{"M3 with icon", true, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: FloatingActionButtonOpts{NoHero: true}, Icon: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)})}, 56, 16 + 24 + 8 + 50 + 20, true},
		{"M3 label only", true, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: FloatingActionButtonOpts{NoHero: true}}, 56, 20 + 50 + 20, true},
		{"M2 height", false, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: FloatingActionButtonOpts{NoHero: true}}, 48, 20 + 50 + 20, true},
		{"collapsed to the icon", true, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: FloatingActionButtonOpts{NoHero: true}, NoExtended: true,
			Icon: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)})}, 56, 16 + 24 + 20, false},
		{"custom spacing", true, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: FloatingActionButtonOpts{NoHero: true},
			ExtendedIconLabelSpacing: foundation.Some(20.0), Icon: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)})}, 56, 16 + 24 + 20 + 50 + 20, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			label := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(10.0)})
			ts := pumpButton(t, standardTheme(tc.material), NewFloatingActionButtonExtended(func() {}, label, tc.opts))
			fab := gluttertest.ByType[*FloatingActionButton]()
			if got := gluttertest.WidgetAs[*FloatingActionButton](ts, fab).IsExtended(); got != tc.hasLabel {
				t.Errorf("IsExtended = %v, want %v", got, tc.hasLabel)
			}
			if got := ts.Exists(gluttertest.ByWidget(label)); got != tc.hasLabel {
				t.Errorf("the label is shown = %v, want %v", got, tc.hasLabel)
			}
			got := ts.Size(gluttertest.First(gluttertest.Descendant(fab, gluttertest.ByType[*Material]())))
			if got != (ui.Size{Width: tc.wantW, Height: tc.wantH}) {
				t.Errorf("size = %v, want %vx%v", got, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestFloatingActionButtonHeroTooltipAndTap(t *testing.T) {
	taps := 0
	ts := pumpButton(t, nil, NewFloatingActionButton(func() { taps++ }, fabIcon(), FloatingActionButtonOpts{Tooltip: "Add"}))
	if !ts.Exists(gluttertest.ByType[*widgets.Hero]()) {
		t.Error("the default button has no hero")
	}
	fab := gluttertest.ByType[*FloatingActionButton]()
	ts.Tap(fab)
	ts.Pump()
	if taps != 1 {
		t.Errorf("taps = %d, want 1", taps)
	}
	if !ts.Exists(gluttertest.ByPredicate("the tooltip", func(w widgets.Widget) bool {
		s, ok := w.(*widgets.Semantics)
		return ok && s.Properties().Tooltip == foundation.Some("Add")
	})) {
		t.Error("the tooltip is missing")
	}
	hero := gluttertest.WidgetAs[*widgets.Hero](ts, gluttertest.ByType[*widgets.Hero]())
	if hero.Tag() == nil {
		t.Error("the hero has no tag")
	}

	ts = pumpButton(t, nil, NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true}))
	if ts.Exists(gluttertest.ByType[*widgets.Hero]()) {
		t.Error("a hero was made with NoHero")
	}
	ts = pumpButton(t, nil, NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{HeroTag: "mine"}))
	if got := gluttertest.WidgetAs[*widgets.Hero](ts, gluttertest.ByType[*widgets.Hero]()).Tag(); got != "mine" {
		t.Errorf("hero tag = %v, want mine", got)
	}
}

func TestFloatingActionButtonLocations(t *testing.T) {
	// A scaffold 400 by 600 with an app bar that is 56 high, and a button 56 by 56.
	g := ScaffoldPrelayoutGeometry{
		FloatingActionButtonSize: ui.Size{Width: 56, Height: 56}, ContentBottom: 600, ContentTop: 56,
		ScaffoldSize: ui.Size{Width: 400, Height: 600}, TextDirection: ui.TextDirectionLTR,
	}
	rtl := g
	rtl.TextDirection = ui.TextDirectionRTL
	snack := g
	snack.SnackBarSize = ui.Size{Width: 400, Height: 48}
	sheet := g
	sheet.BottomSheetSize = ui.Size{Width: 400, Height: 100}
	bar := g
	bar.ContentBottom = 540
	bar.MinViewPadding = painting.EdgeInsets{Bottom: 10}
	tests := []struct {
		name string
		loc  FloatingActionButtonLocation
		geo  ScaffoldPrelayoutGeometry
		want ui.Offset
	}{
		{"startTop", FloatingActionButtonLocationStartTop, g, ui.Offset{Dx: 16, Dy: 28}},
		{"miniStartTop", FloatingActionButtonLocationMiniStartTop, g, ui.Offset{Dx: 12, Dy: 28}},
		{"centerTop", FloatingActionButtonLocationCenterTop, g, ui.Offset{Dx: 172, Dy: 28}},
		{"miniCenterTop", FloatingActionButtonLocationMiniCenterTop, g, ui.Offset{Dx: 172, Dy: 28}},
		{"endTop", FloatingActionButtonLocationEndTop, g, ui.Offset{Dx: 328, Dy: 28}},
		{"miniEndTop", FloatingActionButtonLocationMiniEndTop, g, ui.Offset{Dx: 332, Dy: 28}},
		{"startFloat", FloatingActionButtonLocationStartFloat, g, ui.Offset{Dx: 16, Dy: 528}},
		{"miniStartFloat", FloatingActionButtonLocationMiniStartFloat, g, ui.Offset{Dx: 12, Dy: 532}},
		{"centerFloat", FloatingActionButtonLocationCenterFloat, g, ui.Offset{Dx: 172, Dy: 528}},
		{"miniCenterFloat", FloatingActionButtonLocationMiniCenterFloat, g, ui.Offset{Dx: 172, Dy: 532}},
		{"endFloat", FloatingActionButtonLocationEndFloat, g, ui.Offset{Dx: 328, Dy: 528}},
		{"miniEndFloat", FloatingActionButtonLocationMiniEndFloat, g, ui.Offset{Dx: 332, Dy: 532}},
		{"startDocked", FloatingActionButtonLocationStartDocked, g, ui.Offset{Dx: 16, Dy: 544}},
		{"miniStartDocked", FloatingActionButtonLocationMiniStartDocked, g, ui.Offset{Dx: 12, Dy: 544}},
		{"centerDocked", FloatingActionButtonLocationCenterDocked, g, ui.Offset{Dx: 172, Dy: 544}},
		{"miniCenterDocked", FloatingActionButtonLocationMiniCenterDocked, g, ui.Offset{Dx: 172, Dy: 544}},
		{"endDocked", FloatingActionButtonLocationEndDocked, g, ui.Offset{Dx: 328, Dy: 544}},
		{"miniEndDocked", FloatingActionButtonLocationMiniEndDocked, g, ui.Offset{Dx: 332, Dy: 544}},
		{"endContained", FloatingActionButtonLocationEndContained, g, ui.Offset{Dx: 328, Dy: 544}},
		{"startFloat in RTL is at the right", FloatingActionButtonLocationStartFloat, rtl, ui.Offset{Dx: 328, Dy: 528}},
		{"endFloat in RTL is at the left", FloatingActionButtonLocationEndFloat, rtl, ui.Offset{Dx: 16, Dy: 528}},
		{"floats over the snack bar", FloatingActionButtonLocationEndFloat, snack, ui.Offset{Dx: 328, Dy: 480}},
		{"top ignores the snack bar", FloatingActionButtonLocationEndTop, snack, ui.Offset{Dx: 328, Dy: 28}},
		{"floats over the bottom sheet", FloatingActionButtonLocationEndFloat, sheet, ui.Offset{Dx: 328, Dy: 600 - 100 - 28}},
		{"docked over a bar", FloatingActionButtonLocationCenterDocked, bar, ui.Offset{Dx: 172, Dy: 512}},
		{"contained in a bar", FloatingActionButtonLocationEndContained, bar, ui.Offset{Dx: 328, Dy: 534}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.loc.GetOffset(tc.geo); got != tc.want {
				t.Errorf("offset = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFloatingActionButtonLocationStrings(t *testing.T) {
	if got := FloatingActionButtonLocationEndFloat.(interface{ String() string }).String(); got != "FloatingActionButtonLocation.endFloat" {
		t.Errorf("String = %q", got)
	}
	if got := FloatingActionButtonLocationMiniCenterDocked.(interface{ String() string }).String(); got != "FloatingActionButtonLocation.miniCenterDocked" {
		t.Errorf("String = %q", got)
	}
}

// pumpFabScaffold shows a scaffold with a floating action button, in a window of
// 400 by 600.
func pumpFabScaffold(t *testing.T, ts *gluttertest.Tester, o ScaffoldOpts) *gluttertest.Tester {
	t.Helper()
	if ts == nil {
		ts = gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	}
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: NewScaffold(o)}))
	return ts
}

func TestScaffoldMovesTheButton(t *testing.T) {
	fab := gluttertest.ByType[*FloatingActionButton]()
	button := func() widgets.Widget {
		return NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true})
	}
	ts := pumpFabScaffold(t, nil, ScaffoldOpts{FloatingActionButton: button()})
	ts.PumpAndSettle()
	if got := ts.TopLeft(fab); !near(got, ui.Offset{Dx: 328, Dy: 528}, 0.01) {
		t.Fatalf("position at the start = %v, want the end float one", got)
	}
	scale := func() float64 {
		return gluttertest.WidgetAs[*widgets.ScaleTransition](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*Scaffold](), gluttertest.ByType[*widgets.ScaleTransition]()))).Scale().Value()
	}
	_ = scale

	// A new location: the button scales down where it was, and then up at the
	// new place.
	pumpFabScaffold(t, ts, ScaffoldOpts{FloatingActionButton: button(), FloatingActionButtonLocation: FloatingActionButtonLocationStartTop})
	ts.Pump()
	ts.Pump(50 * time.Millisecond)
	if got := ts.TopLeft(fab); got.Dx < 300 {
		t.Errorf("the button moved before the middle of the animation: %v", got)
	}
	ts.Pump(250 * time.Millisecond)
	if got := ts.TopLeft(fab); math.Abs(got.Dx-16) > 8 {
		t.Errorf("the button is at %v after the middle of the animation, want the new location", got)
	}
	ts.PumpAndSettle()
	if got := ts.TopLeft(fab); !near(got, ui.Offset{Dx: 16, Dy: 0}, 0.01) {
		t.Errorf("position at the end = %v, want the start top one", got)
	}
}

func TestScaffoldNoAnimationAnimatorMovesAtOnce(t *testing.T) {
	fab := gluttertest.ByType[*FloatingActionButton]()
	build := func(loc FloatingActionButtonLocation) ScaffoldOpts {
		return ScaffoldOpts{FloatingActionButton: NewFloatingActionButton(func() {}, fabIcon(), FloatingActionButtonOpts{NoHero: true}),
			FloatingActionButtonLocation: loc, FloatingActionButtonAnimator: FloatingActionButtonAnimatorNoAnimation}
	}
	ts := pumpFabScaffold(t, nil, build(FloatingActionButtonLocationEndFloat))
	pumpFabScaffold(t, ts, build(FloatingActionButtonLocationCenterTop))
	ts.Pump()
	if got := ts.TopLeft(fab); !near(got, ui.Offset{Dx: 172, Dy: 0}, 0.01) {
		t.Errorf("position = %v, want the center top one at once", got)
	}
}

func TestFloatingActionButtonAnimatorValues(t *testing.T) {
	a := FloatingActionButtonAnimatorScaling
	begin, end := ui.Offset{Dx: 1, Dy: 2}, ui.Offset{Dx: 10, Dy: 20}
	tests := []struct {
		progress float64
		want     ui.Offset
	}{{0, begin}, {0.49, begin}, {0.5, end}, {1, end}}
	for _, tc := range tests {
		if got := a.GetOffset(begin, end, tc.progress); got != tc.want {
			t.Errorf("offset at %v = %v, want %v", tc.progress, got, tc.want)
		}
	}
	if got := a.GetAnimationRestart(0.25); got != 0.25 {
		t.Errorf("restart(0.25) = %v, want 0.25", got)
	}
	if got := a.GetAnimationRestart(0.75); got != 0.25 {
		t.Errorf("restart(0.75) = %v, want 0.25", got)
	}
	no := FloatingActionButtonAnimatorNoAnimation
	if got := no.GetOffset(begin, end, 0); got != end {
		t.Errorf("no animation offset = %v, want the end", got)
	}
	if got := no.GetAnimationRestart(0.7); got != 0 {
		t.Errorf("no animation restart = %v", got)
	}
}

func TestFloatingActionButtonTransition(t *testing.T) {
	fabs := func(label string) widgets.Widget {
		return NewFloatingActionButton(func() {}, NewIcon(Icons.Add()), FloatingActionButtonOpts{NoHero: true, Key: foundation.NewValueKey(label)})
	}
	scales := func(ts *gluttertest.Tester) []float64 {
		f := gluttertest.Descendant(gluttertest.ByType[*Scaffold](), gluttertest.ByType[*widgets.ScaleTransition]())
		var out []float64
		for i := 0; i < ts.Count(f); i++ {
			out = append(out, gluttertest.WidgetAs[*widgets.ScaleTransition](ts, gluttertest.At(f, i)).Scale().Value())
		}
		return out
	}
	ts := pumpFabScaffold(t, nil, ScaffoldOpts{})
	if ts.Exists(gluttertest.ByType[*FloatingActionButton]()) {
		t.Fatal("a button is shown without one")
	}
	// A button that is added grows from nothing.
	pumpFabScaffold(t, ts, ScaffoldOpts{FloatingActionButton: fabs("a")})
	ts.Pump()
	if got := scales(ts); len(got) != 1 || got[0] > 0.2 {
		t.Errorf("scale at the first frame = %v, want one near zero", got)
	}
	ts.Pump(100 * time.Millisecond)
	if got := scales(ts); len(got) != 1 || got[0] <= 0.2 || got[0] >= 1 {
		t.Errorf("scale in the middle = %v, want one between 0.2 and 1", got)
	}
	ts.PumpAndSettle()
	if got := scales(ts); len(got) != 1 || got[0] != 1 {
		t.Errorf("scale at the end = %v, want 1", got)
	}

	// A button with another key replaces the first: the first leaves while the
	// second waits, and then it enters.
	pumpFabScaffold(t, ts, ScaffoldOpts{FloatingActionButton: fabs("b")})
	ts.Pump()
	ts.Pump(50 * time.Millisecond)
	if got := scales(ts); len(got) != 2 {
		t.Errorf("scales while the old button leaves = %v, want two", got)
	}
	ts.PumpAndSettle()
	if got := scales(ts); len(got) != 1 || got[0] != 1 {
		t.Errorf("scale at the end = %v, want 1", got)
	}
	if n := ts.Count(gluttertest.ByType[*FloatingActionButton]()); n != 1 {
		t.Errorf("%d buttons are left, want 1", n)
	}

	// With the same key nothing animates.
	pumpFabScaffold(t, ts, ScaffoldOpts{FloatingActionButton: fabs("b")})
	ts.Pump()
	if got := scales(ts); len(got) != 1 || got[0] != 1 {
		t.Errorf("scale after a rebuild = %v, want 1", got)
	}

	// Removing the button leaves nothing.
	pumpFabScaffold(t, ts, ScaffoldOpts{})
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByType[*FloatingActionButton]()) {
		t.Error("the button was not removed")
	}
}

func TestFloatingActionButtonErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		build func()
		route string
	}{
		{"negative elevation", func() {
			NewFloatingActionButton(nil, nil, FloatingActionButtonOpts{Elevation: foundation.Some(-1.0)})
		}, "FloatingActionButtonElevationNegative"},
		{"negative disabled elevation", func() {
			NewFloatingActionButtonExtended(nil, nil, FloatingActionButtonExtendedOpts{FloatingActionButtonOpts: FloatingActionButtonOpts{DisabledElevation: foundation.Some(-0.5)}})
		}, "FloatingActionButtonElevationNegative"},
		{"raw button negative elevation", func() {
			NewRawMaterialButton(nil, nil, RawMaterialButtonOpts{HoverElevation: foundation.Some(-1.0)})
		}, "RawMaterialButtonElevationNegative"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoute(t, recoverError(tc.build), tc.route)
		})
	}
}
