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

// standardTheme is the theme with the standard density: the tests that measure
// buttons do not depend on the platform.
func standardTheme(material3 bool) *ThemeData {
	return NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(material3), VisualDensity: foundation.Some(VisualDensityStandard())})
}

// pumpButton shows the widget in the middle of a Material app 400 by 600 wide.
func pumpButton(t *testing.T, theme *ThemeData, child widgets.Widget) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(child)})}))
	return ts
}

// buttonKinds are the four buttons that are made of a ButtonStyle: the
// constructors with a label, and the ones with an icon too.
var buttonKinds = []struct {
	name string
	make func(onPressed func(), opts ButtonStyleButtonOpts) widgets.Widget
	icon func(onPressed func(), opts ButtonStyleButtonOpts) widgets.Widget
	typ  gluttertest.Finder
}{
	{"TextButton",
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewTextButton(p, widgets.NewText("Go"), o)
		},
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewTextButtonIcon(p, widgets.NewText("Go"), TextButtonIconOpts{ButtonStyleButtonOpts: o, Icon: NewIcon(Icons.Add())})
		},
		gluttertest.ByType[*TextButton]()},
	{"ElevatedButton",
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewElevatedButton(p, widgets.NewText("Go"), o)
		},
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewElevatedButtonIcon(p, widgets.NewText("Go"), ElevatedButtonIconOpts{ButtonStyleButtonOpts: o, Icon: NewIcon(Icons.Add())})
		},
		gluttertest.ByType[*ElevatedButton]()},
	{"FilledButton",
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewFilledButton(p, widgets.NewText("Go"), o)
		},
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewFilledButtonIcon(p, widgets.NewText("Go"), FilledButtonIconOpts{ButtonStyleButtonOpts: o, Icon: NewIcon(Icons.Add())})
		},
		gluttertest.ByType[*FilledButton]()},
	{"FilledButton.tonal",
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewFilledButtonTonal(p, widgets.NewText("Go"), o)
		},
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewFilledButtonTonalIcon(p, widgets.NewText("Go"), FilledButtonIconOpts{ButtonStyleButtonOpts: o, Icon: NewIcon(Icons.Add())})
		},
		gluttertest.ByType[*FilledButton]()},
	{"OutlinedButton",
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewOutlinedButton(p, widgets.NewText("Go"), o)
		},
		func(p func(), o ButtonStyleButtonOpts) widgets.Widget {
			return NewOutlinedButtonIcon(p, widgets.NewText("Go"), OutlinedButtonIconOpts{ButtonStyleButtonOpts: o, Icon: NewIcon(Icons.Add())})
		},
		gluttertest.ByType[*OutlinedButton]()},
}

// buttonMaterial is the Material of the button.
func buttonMaterial(ts *gluttertest.Tester, button gluttertest.Finder) *Material {
	return gluttertest.WidgetAs[*Material](ts, gluttertest.First(gluttertest.Descendant(button, gluttertest.ByType[*Material]())))
}

func TestButtonsTapAndDisable(t *testing.T) {
	for _, k := range buttonKinds {
		t.Run(k.name, func(t *testing.T) {
			taps, longPresses := 0, 0
			ts := pumpButton(t, nil, k.make(func() { taps++ }, ButtonStyleButtonOpts{OnLongPress: func() { longPresses++ }}))
			ts.Tap(k.typ)
			ts.Pump()
			if taps != 1 {
				t.Errorf("taps = %d, want 1", taps)
			}
			ts.LongPress(k.typ)
			ts.PumpAndSettle()
			if longPresses != 1 {
				t.Errorf("long presses = %d, want 1", longPresses)
			}

			// With a nil onPressed and no long press callback the button is disabled.
			ts = pumpButton(t, nil, k.make(nil, ButtonStyleButtonOpts{}))
			b := gluttertest.WidgetAs[ButtonStyleButtonSpec](ts, k.typ).ButtonStyleButtonBase()
			if b.Enabled() {
				t.Error("a button without callbacks is enabled")
			}
			ts.Tap(k.typ)
			ts.Pump()
			// A button with only a long press callback is enabled.
			ts = pumpButton(t, nil, k.make(nil, ButtonStyleButtonOpts{OnLongPress: func() {}}))
			if !gluttertest.WidgetAs[ButtonStyleButtonSpec](ts, k.typ).ButtonStyleButtonBase().Enabled() {
				t.Error("a button with only OnLongPress is disabled")
			}
		})
	}
}

func TestButtonsSizeMaterial3(t *testing.T) {
	tests := []struct {
		name      string
		button    func() widgets.Widget
		finder    gluttertest.Finder
		wantWidth float64 // 0 means only checked to be at least the minimum
		wantH     float64
	}{
		{"text button", func() widgets.Widget {
			return NewTextButton(func() {}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(10.0)}))
		},
			gluttertest.ByType[*TextButton](), 64, 40},
		{"elevated button grows with its padding", func() widgets.Widget {
			return NewElevatedButton(func() {}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(10.0)}))
		}, gluttertest.ByType[*ElevatedButton](), 148, 40},
		{"filled button padding", func() widgets.Widget {
			return NewFilledButton(func() {}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(10.0)}))
		}, gluttertest.ByType[*FilledButton](), 148, 40},
		{"outlined button padding", func() widgets.Widget {
			return NewOutlinedButton(func() {}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(10.0)}))
		}, gluttertest.ByType[*OutlinedButton](), 148, 40},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, standardTheme(true), tc.button())
			// The Material has the visible size: the input padding around it is only
			// for the tap target.
			got := ts.Size(gluttertest.First(gluttertest.Descendant(tc.finder, gluttertest.ByType[*Material]())))
			if got != (ui.Size{Width: tc.wantWidth, Height: tc.wantH}) {
				t.Errorf("size = %v, want %vx%v", got, tc.wantWidth, tc.wantH)
			}
		})
	}
}

func TestButtonsColorsMaterial3(t *testing.T) {
	theme := standardTheme(true)
	cs := theme.ColorScheme
	tests := []struct {
		name       string
		kind       int
		disabled   bool
		wantColor  foundation.Opt[ui.Color]
		wantText   ui.Color
		wantElev   float64
		wantShadow foundation.Opt[ui.Color]
	}{
		{"text", 0, false, foundation.Some(Colors.Transparent), cs.Primary(), 0, foundation.Some(Colors.Transparent)},
		{"text disabled", 0, true, foundation.Some(Colors.Transparent), cs.OnSurface().WithOpacity(0.38), 0, foundation.Some(Colors.Transparent)},
		{"elevated", 1, false, foundation.Some(cs.SurfaceContainerLow()), cs.Primary(), 1, foundation.Some(cs.Shadow())},
		{"elevated disabled", 1, true, foundation.Some(cs.OnSurface().WithOpacity(0.12)), cs.OnSurface().WithOpacity(0.38), 0, foundation.Some(cs.Shadow())},
		{"filled", 2, false, foundation.Some(cs.Primary()), cs.OnPrimary(), 0, foundation.Some(cs.Shadow())},
		{"filled tonal", 3, false, foundation.Some(cs.SecondaryContainer()), cs.OnSecondaryContainer(), 0, foundation.Some(cs.Shadow())},
		{"outlined", 4, false, foundation.Some(Colors.Transparent), cs.Primary(), 0, foundation.Some(Colors.Transparent)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k := buttonKinds[tc.kind]
			var onPressed func()
			if !tc.disabled {
				onPressed = func() {}
			}
			ts := pumpButton(t, theme, k.make(onPressed, ButtonStyleButtonOpts{}))
			mat := buttonMaterial(ts, k.typ)
			if mat.Color() != tc.wantColor {
				t.Errorf("color = %v, want %v", mat.Color(), tc.wantColor)
			}
			if mat.Elevation() != tc.wantElev {
				t.Errorf("elevation = %v, want %v", mat.Elevation(), tc.wantElev)
			}
			if mat.ShadowColor() != tc.wantShadow {
				t.Errorf("shadow color = %v, want %v", mat.ShadowColor(), tc.wantShadow)
			}
			if got := mat.TextStyle().Color; got != foundation.Some(tc.wantText) {
				t.Errorf("text color = %v, want %v", got, tc.wantText)
			}
			if _, ok := mat.Shape().(painting.StadiumBorder); !ok {
				t.Errorf("shape = %T, want a stadium", mat.Shape())
			}
		})
	}
}

func TestButtonsMaterial2Defaults(t *testing.T) {
	theme := standardTheme(false)
	cs := theme.ColorScheme
	tests := []struct {
		name     string
		kind     int
		wantSize ui.Size
		wantFill foundation.Opt[ui.Color]
		wantElev float64
	}{
		{"text", 0, ui.Size{Width: 64, Height: 36}, foundation.Some(Colors.Transparent), 0},
		{"elevated", 1, ui.Size{Width: 64, Height: 36}, foundation.Some(cs.Primary()), 2},
		{"outlined", 4, ui.Size{Width: 64, Height: 36}, foundation.Some(Colors.Transparent), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k := buttonKinds[tc.kind]
			ts := pumpButton(t, theme, k.make(func() {}, ButtonStyleButtonOpts{}))
			mat := buttonMaterial(ts, k.typ)
			if got := ts.Size(gluttertest.First(gluttertest.Descendant(k.typ, gluttertest.ByType[*Material]()))); got.Height != tc.wantSize.Height {
				t.Errorf("height = %v, want %v", got.Height, tc.wantSize.Height)
			}
			if mat.Color() != tc.wantFill {
				t.Errorf("color = %v, want %v", mat.Color(), tc.wantFill)
			}
			if mat.Elevation() != tc.wantElev {
				t.Errorf("elevation = %v, want %v", mat.Elevation(), tc.wantElev)
			}
			rr, ok := mat.Shape().(painting.RoundedRectangleBorder)
			if !ok {
				t.Fatalf("shape = %T, want a rounded rectangle", mat.Shape())
			}
			if rr.BorderRadius.Equal(painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))) == false {
				t.Errorf("radius = %v, want 4", rr.BorderRadius)
			}
		})
	}
}

func TestElevatedButtonElevationFollowsStates(t *testing.T) {
	tests := []struct {
		name   string
		states []widgets.WidgetState
		want   float64
	}{
		{"enabled", nil, 1},
		{"hovered", []widgets.WidgetState{widgets.WidgetStateHovered}, 3},
		{"focused", []widgets.WidgetState{widgets.WidgetStateFocused}, 1},
		{"pressed", []widgets.WidgetState{widgets.WidgetStatePressed}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controller := widgets.NewWidgetStatesController(tc.states...)
			ts := pumpButton(t, nil, NewElevatedButton(func() {}, widgets.NewText("Go"), ElevatedButtonOpts{StatesController: controller}))
			ts.PumpAndSettle()
			if got := buttonMaterial(ts, gluttertest.ByType[*ElevatedButton]()).Elevation(); got != tc.want {
				t.Errorf("elevation = %v, want %v", got, tc.want)
			}
		})
	}
	t.Run("filled hovered", func(t *testing.T) {
		controller := widgets.NewWidgetStatesController(widgets.WidgetStateHovered)
		ts := pumpButton(t, nil, NewFilledButton(func() {}, widgets.NewText("Go"), FilledButtonOpts{StatesController: controller}))
		if got := buttonMaterial(ts, gluttertest.ByType[*FilledButton]()).Elevation(); got != 1 {
			t.Errorf("elevation = %v, want 1", got)
		}
	})
}

func TestButtonStatesFromThePointer(t *testing.T) {
	for _, k := range buttonKinds {
		t.Run(k.name, func(t *testing.T) {
			controller := widgets.NewWidgetStatesController()
			var hovers []bool
			ts := pumpButton(t, nil, k.make(func() {}, ButtonStyleButtonOpts{StatesController: controller, OnHover: func(v bool) { hovers = append(hovers, v) }}))
			if got := controller.Value(); got != widgets.NewWidgetStates() {
				t.Fatalf("states at rest = %v", got)
			}
			center := ts.Center(k.typ)
			ts.Hover(center)
			ts.Pump()
			if !controller.Value().Has(widgets.WidgetStateHovered) {
				t.Errorf("states over the button = %v, want hovered", controller.Value())
			}
			g := ts.StartGesture(center, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
			ts.Pump()
			if !controller.Value().Has(widgets.WidgetStatePressed) {
				t.Errorf("states while pressed = %v, want pressed", controller.Value())
			}
			g.Up()
			ts.PumpAndSettle()
			if controller.Value().Has(widgets.WidgetStatePressed) {
				t.Errorf("states after the press = %v", controller.Value())
			}
			ts.Hover(ui.Offset{Dx: 5, Dy: 5})
			ts.Pump()
			if controller.Value().Has(widgets.WidgetStateHovered) {
				t.Errorf("states away from the button = %v", controller.Value())
			}
			if len(hovers) != 2 || !hovers[0] || hovers[1] {
				t.Errorf("OnHover calls = %v, want [true false]", hovers)
			}
		})
	}
}

func TestButtonStatesControllerFollowsEnabled(t *testing.T) {
	controller := widgets.NewWidgetStatesController()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	build := func(onPressed func()) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewTextButton(onPressed, widgets.NewText("Go"), TextButtonOpts{StatesController: controller}))})})
	}
	ts.PumpWidget(build(func() {}))
	if controller.Value().Has(widgets.WidgetStateDisabled) {
		t.Error("the enabled button is disabled")
	}
	ts.PumpWidget(build(nil))
	if !controller.Value().Has(widgets.WidgetStateDisabled) {
		t.Error("the button was not disabled")
	}
	// A press that is under way when the button is disabled is cancelled.
	controller.Update(widgets.WidgetStatePressed, true)
	ts.PumpWidget(build(func() {}))
	ts.PumpWidget(build(nil))
	if controller.Value().Has(widgets.WidgetStatePressed) {
		t.Errorf("states of the disabled button = %v, want no pressed", controller.Value())
	}
	ts.PumpWidget(build(func() {}))
	if controller.Value().Has(widgets.WidgetStateDisabled) {
		t.Error("the button was not enabled again")
	}
}

func TestButtonStyleOrder(t *testing.T) {
	// The style of the widget wins over the style of the theme, which wins over
	// the defaults; what a style does not say comes from the next.
	red, green, blue := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF)
	theme := NewThemeData(ThemeDataOpts{
		VisualDensity: foundation.Some(VisualDensityStandard()),
		TextButtonTheme: TextButtonThemeData{Style: TextButtonStyleFrom(ButtonStyleFromOpts{
			BackgroundColor: foundation.Some(green), ForegroundColor: foundation.Some(blue), MinimumSize: foundation.Some(ui.Size{Width: 120, Height: 50})})},
	})
	ts := pumpButton(t, theme, NewTextButton(func() {}, widgets.NewText("Go"),
		TextButtonOpts{Style: TextButtonStyleFrom(ButtonStyleFromOpts{BackgroundColor: foundation.Some(red)})}))
	mat := buttonMaterial(ts, gluttertest.ByType[*TextButton]())
	if mat.Color() != foundation.Some(red) {
		t.Errorf("color = %v, want the one of the widget", mat.Color())
	}
	if got := mat.TextStyle().Color; got != foundation.Some(blue) {
		t.Errorf("text color = %v, want the one of the theme", got)
	}
	if got := ts.Size(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*TextButton](), gluttertest.ByType[*Material]()))); got != (ui.Size{Width: 120, Height: 50}) {
		t.Errorf("size = %v, want the minimum of the theme", got)
	}

	// A TextButtonTheme above the button wins over the theme of the app.
	ts = pumpButton(t, theme, NewTextButtonTheme(TextButtonThemeData{Style: TextButtonStyleFrom(ButtonStyleFromOpts{ForegroundColor: foundation.Some(red)})},
		NewTextButton(func() {}, widgets.NewText("Go"))))
	if got := buttonMaterial(ts, gluttertest.ByType[*TextButton]()).TextStyle().Color; got != foundation.Some(red) {
		t.Errorf("text color = %v, want the one of the TextButtonTheme", got)
	}
}

func TestButtonStyleSizes(t *testing.T) {
	tests := []struct {
		name  string
		style *ButtonStyle
		want  ui.Size
	}{
		{"fixed size", NewButtonStyle(ButtonStyleOpts{FixedSize: allOpt(ui.Size{Width: 200, Height: 70})}), ui.Size{Width: 200, Height: 70}},
		{"fixed width only", NewButtonStyle(ButtonStyleOpts{FixedSize: allOpt(ui.NewSizeFromWidth(150)),
			Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.EdgeInsetsZero)}), ui.Size{Width: 150, Height: 100}},
		{"maximum size", NewButtonStyle(ButtonStyleOpts{MinimumSize: allOpt(ui.Size{Width: 10, Height: 10}), MaximumSize: allOpt(ui.Size{Width: 50, Height: 30}),
			Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.EdgeInsetsZero)}), ui.Size{Width: 50, Height: 30}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, nil, NewTextButton(func() {}, widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
				TextButtonOpts{Style: tc.style}))
			got := ts.Size(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*TextButton](), gluttertest.ByType[*Material]())))
			// The child is 100 by 100: the maximum and the fixed width win over it.
			if got != tc.want {
				t.Errorf("size = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestButtonTextStyleMergesWithTheLabelStyle checks that a text style given to a
// button, or to its theme, is merged onto the default label style: the label keeps
// its font (with the bundled fonts alone, a style with no family drew no label) and
// the foreground color.
func TestButtonTextStyleMergesWithTheLabelStyle(t *testing.T) {
	const textKind, filledKind, outlinedKind = 0, 2, 4
	size := func(v float64) *painting.TextStyle {
		return painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(v)})
	}
	bold := painting.NewTextStyle(painting.TextStyleOpts{FontWeight: ui.FontWeightW700})
	boldTwenty := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(20.0), FontWeight: ui.FontWeightW700})
	noInherit := painting.NewTextStyle(painting.TextStyleOpts{Inherit: foundation.Some(false), FontFamily: "Roboto", FontSize: foundation.Some(20.0)})
	red := Colors.Red.Shade700()
	theme := standardTheme(true)
	cs := theme.ColorScheme
	// label is the default label style: the localized LabelLarge of the theme.
	label := buttonMaterial(pumpButton(t, theme, buttonKinds[filledKind].make(func() {}, ButtonStyleButtonOpts{})), buttonKinds[filledKind].typ).TextStyle()
	styleFrom := [...]func(ButtonStyleFromOpts) *ButtonStyle{textKind: TextButtonStyleFrom, filledKind: FilledButtonStyleFrom, outlinedKind: OutlinedButtonStyleFrom}
	themed := func(kind int, st *ButtonStyle) ThemeDataOpts {
		switch kind {
		case filledKind:
			return ThemeDataOpts{FilledButtonTheme: NewFilledButtonThemeData(FilledButtonThemeData{Style: st})}
		case outlinedKind:
			return ThemeDataOpts{OutlinedButtonTheme: NewOutlinedButtonThemeData(OutlinedButtonThemeData{Style: st})}
		}
		return ThemeDataOpts{TextButtonTheme: NewTextButtonThemeData(TextButtonThemeData{Style: st})}
	}
	tests := []struct {
		name       string
		kind       int
		onButton   ButtonStyleFromOpts
		onTheme    *ButtonStyleFromOpts
		wantFamily string
		wantSize   float64
		wantWeight ui.FontWeight
		wantColor  ui.Color
	}{
		{"filled, size on the button", filledKind, ButtonStyleFromOpts{TextStyle: size(20)}, nil,
			label.FontFamily, 20, label.FontWeight, cs.OnPrimary()},
		{"filled, size and foreground on the button", filledKind, ButtonStyleFromOpts{TextStyle: size(20), ForegroundColor: foundation.Some(red)}, nil,
			label.FontFamily, 20, label.FontWeight, red},
		{"filled, weight on the button", filledKind, ButtonStyleFromOpts{TextStyle: bold}, nil,
			label.FontFamily, label.FontSize.Value(), ui.FontWeightW700, cs.OnPrimary()},
		{"outlined, size on the button", outlinedKind, ButtonStyleFromOpts{TextStyle: size(20)}, nil,
			label.FontFamily, 20, label.FontWeight, cs.Primary()},
		{"text, size on the button", textKind, ButtonStyleFromOpts{TextStyle: size(20)}, nil,
			label.FontFamily, 20, label.FontWeight, cs.Primary()},
		{"filled, size and weight on the theme", filledKind, ButtonStyleFromOpts{}, &ButtonStyleFromOpts{TextStyle: boldTwenty},
			label.FontFamily, 20, ui.FontWeightW700, cs.OnPrimary()},
		{"outlined, size on the theme", outlinedKind, ButtonStyleFromOpts{}, &ButtonStyleFromOpts{TextStyle: size(20)},
			label.FontFamily, 20, label.FontWeight, cs.Primary()},
		{"text, weight on the theme and size on the button", textKind, ButtonStyleFromOpts{TextStyle: size(24)}, &ButtonStyleFromOpts{TextStyle: bold},
			label.FontFamily, 24, ui.FontWeightW700, cs.Primary()},
		{"a style that does not inherit replaces the label style", filledKind, ButtonStyleFromOpts{TextStyle: noInherit}, nil,
			"Roboto", 20, ui.FontWeight(0), cs.OnPrimary()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			to := ThemeDataOpts{}
			if tc.onTheme != nil {
				to = themed(tc.kind, styleFrom[tc.kind](*tc.onTheme))
			}
			to.UseMaterial3 = foundation.Some(true)
			to.VisualDensity = foundation.Some(VisualDensityStandard())
			k := buttonKinds[tc.kind]
			ts := pumpButton(t, NewThemeData(to), k.make(func() {}, ButtonStyleButtonOpts{Style: styleFrom[tc.kind](tc.onButton)}))
			got := buttonMaterial(ts, k.typ).TextStyle()
			if got.FontFamily != tc.wantFamily || got.FontSize != foundation.Some(tc.wantSize) || got.FontWeight != tc.wantWeight {
				t.Errorf("text style = family %q, size %v, weight %v; want %q, %v, %v",
					got.FontFamily, got.FontSize, got.FontWeight, tc.wantFamily, tc.wantSize, tc.wantWeight)
			}
			if got.Color != foundation.Some(tc.wantColor) {
				t.Errorf("text color = %v, want %v", got.Color, tc.wantColor)
			}
			if w := ts.Size(gluttertest.ByType[*widgets.RichText]()).Width; w <= 0 {
				t.Errorf("the label is %v wide: no font has its glyphs", w)
			}
		})
	}
}

func allOpt[T any](v T) widgets.WidgetStateProperty[foundation.Opt[T]] {
	return widgets.NewWidgetStatePropertyAll(foundation.Some(v))
}

func TestButtonTapTarget(t *testing.T) {
	// On a theme with the padded tap target a small button can be tapped around
	// its visible area, in the 48 pixels that the input padding makes.
	small := NewButtonStyle(ButtonStyleOpts{
		MinimumSize: allOpt(ui.Size{Width: 20, Height: 20}), Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.EdgeInsetsZero)})
	for _, size := range []MaterialTapTargetSize{MaterialTapTargetSizePadded, MaterialTapTargetSizeShrinkWrap} {
		t.Run(size.Name(), func(t *testing.T) {
			taps := 0
			ts := pumpButton(t, NewThemeData(ThemeDataOpts{MaterialTapTargetSize: size, VisualDensity: foundation.Some(VisualDensityStandard())}), NewTextButton(func() { taps++ },
				widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(1.0), Height: foundation.Some(1.0)}), TextButtonOpts{Style: small}))
			button := gluttertest.ByType[*TextButton]()
			visible := ts.Size(gluttertest.First(gluttertest.Descendant(button, gluttertest.ByType[*Material]())))
			if visible != (ui.Size{Width: 20, Height: 20}) {
				t.Fatalf("visible size = %v, want 20x20", visible)
			}
			center := ts.Center(button)
			ts.TapAt(center.Add(ui.Offset{Dx: 20, Dy: 0}))
			ts.Pump()
			wantTaps := 1
			if size == MaterialTapTargetSizeShrinkWrap {
				wantTaps = 0
			}
			if taps != wantTaps {
				t.Errorf("taps outside the visible area = %d, want %d", taps, wantTaps)
			}
		})
	}
}

func TestButtonIconColorAndSize(t *testing.T) {
	cs := NewThemeData().ColorScheme
	ts := pumpButton(t, nil, NewElevatedButtonIcon(func() {}, widgets.NewText("Go"), ElevatedButtonIconOpts{Icon: NewIcon(Icons.Add())}))
	icon := gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*ElevatedButton](), gluttertest.ByType[*widgets.Icon]()))
	theme := widgets.IconThemeOf(ts.Element(icon))
	if theme.Size() != foundation.Some(18.0) {
		t.Errorf("icon size = %v, want 18", theme.Size())
	}
	if theme.Color() != foundation.Some(cs.Primary()) {
		t.Errorf("icon color = %v, want the primary color", theme.Color())
	}
	// With IconColor and IconSize in the style.
	ts = pumpButton(t, nil, NewTextButton(func() {}, NewIcon(Icons.Add()), TextButtonOpts{Style: NewButtonStyle(ButtonStyleOpts{
		IconColor: allOpt(ui.NewColor(0xFF123456)), IconSize: allOpt(30.0)})}))
	icon = gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*TextButton](), gluttertest.ByType[*widgets.Icon]()))
	theme = widgets.IconThemeOf(ts.Element(icon))
	if theme.Size() != foundation.Some(30.0) || theme.Color() != foundation.Some(ui.NewColor(0xFF123456)) {
		t.Errorf("icon theme = %v, want size 30 and color 0xFF123456", theme)
	}
}

func TestButtonIconAlignment(t *testing.T) {
	tests := []struct {
		name      string
		alignment IconAlignment
		style     *ButtonStyle
		iconFirst bool
	}{
		{"default", IconAlignmentUnspecified, nil, true},
		{"start", IconAlignmentStart, nil, true},
		{"end", IconAlignmentEnd, nil, false},
		{"style says end", IconAlignmentUnspecified, NewButtonStyle(ButtonStyleOpts{IconAlignment: IconAlignmentEnd}), false},
		{"argument wins over the style", IconAlignmentStart, NewButtonStyle(ButtonStyleOpts{IconAlignment: IconAlignmentEnd}), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, nil, NewTextButtonIcon(func() {}, widgets.NewText("Label"), TextButtonIconOpts{
				ButtonStyleButtonOpts: ButtonStyleButtonOpts{Style: tc.style}, Icon: NewIcon(Icons.Add()), IconAlignment: tc.alignment}))
			icon := ts.Center(gluttertest.ByType[*widgets.Icon]())
			label := ts.Center(gluttertest.ByText("Label"))
			if got := icon.Dx < label.Dx; got != tc.iconFirst {
				t.Errorf("icon before the label = %v, want %v", got, tc.iconFirst)
			}
		})
	}
}

func TestButtonWithoutIconIsJustTheLabel(t *testing.T) {
	ts := pumpButton(t, nil, NewFilledButtonIcon(func() {}, widgets.NewText("Label")))
	if ts.Exists(gluttertest.ByType[*widgets.Icon]()) {
		t.Error("an icon was made for a nil icon")
	}
	if !ts.Exists(gluttertest.ByText("Label")) {
		t.Error("the label is missing")
	}
}

func TestButtonTooltipAndSemantics(t *testing.T) {
	ts := pumpButton(t, nil, NewTextButton(func() {}, widgets.NewText("Go"), TextButtonOpts{Tooltip: "Does it"}))
	sem := gluttertest.WidgetAs[*widgets.Semantics](ts, gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*TextButton](), gluttertest.ByType[*widgets.Semantics]())))
	_ = sem
	if !ts.Exists(gluttertest.ByPredicate("a tooltip semantics", func(w widgets.Widget) bool {
		s, ok := w.(*widgets.Semantics)
		return ok && s.Properties().Tooltip == foundation.Some("Does it")
	})) {
		t.Error("the tooltip is not in the semantics")
	}
}

func TestButtonAnimationDelaysTranslucentBackground(t *testing.T) {
	// When a button loses its elevation and the background gets translucent the
	// opaque background stays until the animation has ended.
	opaque, translucent := ui.NewColor(0xFF336699), ui.NewColor(0x80336699)
	style := func(enabled bool) *ButtonStyle {
		return NewButtonStyle(ButtonStyleOpts{
			BackgroundColor: widgets.WidgetStatePropertyResolveWith(func(s widgets.WidgetStates) foundation.Opt[ui.Color] {
				if s.Has(widgets.WidgetStateDisabled) {
					return foundation.Some(translucent)
				}
				return foundation.Some(opaque)
			}),
			Elevation: widgets.WidgetStatePropertyResolveWith(func(s widgets.WidgetStates) foundation.Opt[float64] {
				if s.Has(widgets.WidgetStateDisabled) {
					return foundation.Some(0.0)
				}
				return foundation.Some(4.0)
			}),
			AnimationDuration: foundation.Some(200 * time.Millisecond),
		})
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	build := func(onPressed func()) widgets.Widget {
		return NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewCenter(
			NewElevatedButton(onPressed, widgets.NewText("Go"), ElevatedButtonOpts{Style: style(onPressed != nil)}))})})
	}
	ts.PumpWidget(build(func() {}))
	ts.PumpWidget(build(nil))
	if got := buttonMaterial(ts, gluttertest.ByType[*ElevatedButton]()).Color(); got != foundation.Some(opaque) {
		t.Errorf("color while the elevation goes = %v, want the opaque one", got)
	}
	ts.Pump(300 * time.Millisecond)
	ts.PumpAndSettle()
	if got := buttonMaterial(ts, gluttertest.ByType[*ElevatedButton]()).Color(); got != foundation.Some(translucent) {
		t.Errorf("color at the end = %v, want the translucent one", got)
	}
}
