// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	testHintText    = "hint"
	testInputText   = "text"
	testLabelText   = "label"
	testErrorText   = "error"
	testHelperText  = "helper"
	testCounterText = "counter"
)

// decoratorCase is the parameters of buildInputDecorator, Flutter's
// buildInputDecorator of input_decorator_test.dart.
type decoratorCase struct {
	decoration           InputDecoration
	theme                *ThemeData
	inputDecorationTheme *InputDecorationThemeData
	localInputTheme      *InputDecorationThemeData
	textDirection        ui.TextDirection
	expands              bool
	isEmpty              bool
	isFocused            bool
	isHovering           bool
	useIntrinsicWidth    bool
	baseStyle            *painting.TextStyle
	textAlignVertical    foundation.Opt[painting.TextAlignVertical]
	visualDensity        foundation.Opt[VisualDensity]
	child                widgets.Widget
	noChild              bool
}

func someStr(s string) foundation.Opt[string] { return foundation.Some(s) }

// buildInputDecorator is the app that Flutter's test builds: the decorator in the
// top left corner of a Material.
func buildInputDecorator(c decoratorCase) widgets.Widget {
	if c.textDirection == ui.TextDirectionUnspecified {
		c.textDirection = ui.TextDirectionLTR
	}
	child := c.child
	if child == nil && !c.noChild {
		// A text style compliant with the M3 specification (bodyLarge for text fields).
		child = widgets.NewText(testInputText, widgets.TextOpts{Style: painting.NewTextStyle(painting.TextStyleOpts{
			FontSize: foundation.Some(16.0), FontWeight: ui.FontWeightW400, LetterSpacing: foundation.Some(0.5), Height: foundation.Some(1.5)})})
	}
	var w widgets.Widget = NewInputDecorator(c.decoration, InputDecoratorOpts{
		Expands: c.expands, IsEmpty: c.isEmpty, IsFocused: c.isFocused, IsHovering: c.isHovering,
		BaseStyle: c.baseStyle, TextAlignVertical: c.textAlignVertical, Child: child})
	if c.useIntrinsicWidth {
		w = widgets.NewIntrinsicWidth(w)
	}
	if c.localInputTheme != nil {
		w = NewInputDecorationTheme(*c.localInputTheme, w)
	}
	return NewMaterialApp(MaterialAppOpts{Home: NewMaterial(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		theme := c.theme
		if theme == nil {
			theme = ThemeOf(bc)
		}
		copy := theme.CopyWith(ThemeDataCopy{})
		if c.inputDecorationTheme != nil {
			copy.InputDecorationTheme = *c.inputDecorationTheme
		}
		if v, ok := c.visualDensity.Get(); ok {
			copy.VisualDensity = v
		}
		return NewTheme(copy, widgets.NewAlign(widgets.NewDirectionality(c.textDirection, w), widgets.AlignOpts{Alignment: painting.AlignmentTopLeft}))
	}))})
}

func pumpDecorator(t *testing.T, c decoratorCase) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(buildInputDecorator(c))
	return ts
}

// m3Standard is the Material 3 theme with the standard density, as the tests of
// Flutter run on a mobile platform.
func m3Standard() *ThemeData {
	return NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true), VisualDensity: foundation.Some(VisualDensityStandard())})
}

func findBorderPainter() gluttertest.Finder {
	return gluttertest.Descendant(gluttertest.ByType[*borderContainer](), gluttertest.ByType[*widgets.CustomPaint]())
}

func borderPainter(ts *gluttertest.Tester) *inputBorderPainter {
	return ts.Widget(findBorderPainter()).(*widgets.CustomPaint).ForegroundPainter().(*inputBorderPainter)
}

func getBorder(ts *gluttertest.Tester) InputBorder {
	p := borderPainter(ts)
	return p.border.evaluate(p.borderAnimation)
}

func getBorderSide(ts *gluttertest.Tester) painting.BorderSide { return getBorder(ts).BorderSide() }

func getBorderRadius(ts *gluttertest.Tester) (painting.BorderRadius, bool) {
	switch b := getBorder(ts).(type) {
	case UnderlineInputBorder:
		return b.BorderRadius(), true
	case OutlineInputBorder:
		return b.BorderRadius(), true
	}
	return painting.BorderRadius{}, false
}

func containerBox(ts *gluttertest.Tester) rendering.Box {
	return InputDecoratorContainerOf(ts.Element(findBorderPainter()))
}

func containerHeight(ts *gluttertest.Tester) float64 { return containerBox(ts).Size().Height }

func findDecorator() gluttertest.Finder { return gluttertest.ByType[*InputDecorator]() }

func decoratorSize(ts *gluttertest.Tester) ui.Size {
	return ts.RenderObject(findDecorator()).(rendering.Box).Size()
}

// rectOf is the rectangle of the widget in global coordinates (Flutter's
// WidgetController.getRect: the corners go through the transforms).
func rectOf(ts *gluttertest.Tester, f gluttertest.Finder) ui.Rect {
	tl, br := ts.TopLeft(f), ts.BottomRight(f)
	return ui.Rect{Left: tl.Dx, Top: tl.Dy, Right: br.Dx, Bottom: br.Dy}
}

func labelRect(ts *gluttertest.Tester) ui.Rect { return rectOf(ts, findLabel()) }
func inputRect(ts *gluttertest.Tester) ui.Rect { return rectOf(ts, findInputText()) }
func hintRect(ts *gluttertest.Tester) ui.Rect  { return rectOf(ts, findHint()) }

func wantRect(t *testing.T, what string, got ui.Rect, left, top, right, bottom float64) {
	t.Helper()
	if !approxEq(got.Left, left) || !approxEq(got.Top, top) || !approxEq(got.Right, right) || !approxEq(got.Bottom, bottom) {
		t.Errorf("%s = %v, want [%v %v %v %v]", what, got, left, top, right, bottom)
	}
}

func findLabel() gluttertest.Finder {
	return gluttertest.Descendant(gluttertest.ByType[*widgets.MatrixTransition](), gluttertest.ByType[*widgets.Text]())
}

func findInputText() gluttertest.Finder { return gluttertest.ByText(testInputText) }
func findHint() gluttertest.Finder      { return gluttertest.ByText(testHintText) }
func findHelper() gluttertest.Finder    { return gluttertest.ByText(testHelperText) }
func findError() gluttertest.Finder     { return gluttertest.ByText(testErrorText) }
func findCounter() gluttertest.Finder   { return gluttertest.ByText(testCounterText) }

// opacityOf is the opacity of the closest FadeTransition above the text.
func opacityOf(ts *gluttertest.Tester, text string) float64 {
	f := gluttertest.First(gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*widgets.FadeTransition]()))
	return ts.Widget(f).(*widgets.FadeTransition).Opacity().Value()
}

func approxEq(a, b float64) bool {
	d := a - b
	return d > -0.01 && d < 0.01
}

func wantNear(t *testing.T, what string, got, want float64) {
	t.Helper()
	if !approxEq(got, want) {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestInputDecoratorSmoke(t *testing.T) {
	ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: InputDecoration{
		Filled: foundation.Some(true), LabelText: someStr(testLabelText), HelperText: someStr(testHelperText)}})
	wantNear(t, "container height", containerHeight(ts), 56)
	if _, ok := getBorder(ts).(UnderlineInputBorder); !ok {
		t.Errorf("border = %v", getBorder(ts))
	}
	_ = time.Second
}

// colorSchemeOf is the color scheme of the theme of the decorator.
func colorSchemeOf(ts *gluttertest.Tester) *ColorScheme {
	return ThemeOf(ts.Element(findDecorator())).ColorScheme
}

func TestInputDecoratorContainer(t *testing.T) {
	type states struct {
		name                     string
		disabled, hover, focused bool
		err                      bool
	}
	filled := []struct {
		states
		color  func(c *ColorScheme) ui.Color
		weight float64
	}{
		{states{name: "enabled"}, func(c *ColorScheme) ui.Color { return c.OnSurfaceVariant() }, 1},
		{states{name: "disabled", disabled: true}, func(c *ColorScheme) ui.Color { return c.OnSurface().WithOpacity(0.38) }, 1},
		{states{name: "hovered", hover: true}, func(c *ColorScheme) ui.Color { return c.OnSurface() }, 1},
		{states{name: "focused", focused: true}, func(c *ColorScheme) ui.Color { return c.Primary() }, 2},
		{states{name: "focused and hovered", focused: true, hover: true}, func(c *ColorScheme) ui.Color { return c.Primary() }, 2},
		{states{name: "error", err: true}, func(c *ColorScheme) ui.Color { return c.Error() }, 1},
		{states{name: "error and focused", err: true, focused: true}, func(c *ColorScheme) ui.Color { return c.Error() }, 2},
		{states{name: "error and hovered", err: true, hover: true}, func(c *ColorScheme) ui.Color { return c.OnErrorContainer() }, 1},
		{states{name: "error, focused and hovered", err: true, focused: true, hover: true}, func(c *ColorScheme) ui.Color { return c.Error() }, 2},
	}
	for _, outlined := range []bool{false, true} {
		for _, tc := range filled {
			name := "filled " + tc.name
			if outlined {
				name = "outlined " + tc.name
			}
			t.Run(name, func(t *testing.T) {
				dec := InputDecoration{LabelText: someStr(testLabelText), Disabled: tc.disabled}
				if tc.err {
					dec.ErrorText = someStr(testErrorText)
				} else {
					dec.HelperText = someStr(testHelperText)
				}
				if outlined {
					dec.Border = NewOutlineInputBorder()
				} else {
					dec.Filled = foundation.Some(true)
				}
				ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: dec, isFocused: tc.focused, isHovering: tc.hover})
				colors := colorSchemeOf(ts)
				wantHeight := 56.0
				wantNear(t, "container height", containerHeight(ts), wantHeight)
				side := getBorderSide(ts)
				wantColor := tc.color(colors)
				if outlined {
					switch {
					case tc.disabled:
						wantColor = colors.OnSurface().WithOpacity(0.12)
					case !tc.err && !tc.focused && !tc.hover:
						wantColor = colors.Outline()
					}
				}
				if side.Color != wantColor {
					t.Errorf("border color = %v, want %v", side.Color, wantColor)
				}
				wantNear(t, "border weight", side.Width, tc.weight)
				br, ok := getBorderRadius(ts)
				if !ok {
					t.Fatalf("border %v has no radius", getBorder(ts))
				}
				want := painting.BorderRadius{TopLeft: ui.NewRadiusCircular(4), TopRight: ui.NewRadiusCircular(4)}
				if outlined {
					want = painting.NewBorderRadiusAll(ui.NewRadiusCircular(4))
				}
				if br != want {
					t.Errorf("border radius = %v, want %v", br, want)
				}
				if _, isOutline := getBorder(ts).(OutlineInputBorder); isOutline != outlined {
					t.Errorf("border %v, outlined %v", getBorder(ts), outlined)
				}
			})
		}
	}
}

func TestInputDecoratorContainerHeights(t *testing.T) {
	desktop := NewThemeData(ThemeDataOpts{UseMaterial3: foundation.Some(true)})
	tests := []struct {
		name  string
		theme *ThemeData
		dec   InputDecoration
		vd    foundation.Opt[VisualDensity]
		local *InputDecorationThemeData
		want  float64
	}{
		{"filled, default density on the desktop", desktop, InputDecoration{Filled: foundation.Some(true)}, foundation.None[VisualDensity](), nil, 48},
		{"outlined, default density on the desktop", desktop, InputDecoration{Border: NewOutlineInputBorder()}, foundation.None[VisualDensity](), nil, 48},
		{"filled, standard density", m3Standard(), InputDecoration{Filled: foundation.Some(true)}, foundation.None[VisualDensity](), nil, 56},
		{"filled, compact density", m3Standard(), InputDecoration{Filled: foundation.Some(true)}, foundation.Some(VisualDensityCompact()), nil, 48},
		{"outlined, comfortable density", m3Standard(), InputDecoration{Border: NewOutlineInputBorder()}, foundation.Some(VisualDensityComfortable()), nil, 52},
		{"decoration density wins", m3Standard(), InputDecoration{Filled: foundation.Some(true),
			VisualDensity: foundation.Some(NewVisualDensity(VisualDensityOpts{Vertical: foundation.Some(-1.0)}))}, foundation.Some(VisualDensityCompact()), nil, 52},
		{"theme density wins", m3Standard(), InputDecoration{Border: NewOutlineInputBorder()}, foundation.Some(VisualDensityCompact()),
			&InputDecorationThemeData{VisualDensity: foundation.Some(NewVisualDensity(VisualDensityOpts{Vertical: foundation.Some(-1.0)}))}, 52},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.dec.LabelText = someStr(testLabelText)
			tc.dec.HelperText = someStr(testHelperText)
			ts := pumpDecorator(t, decoratorCase{theme: tc.theme, decoration: tc.dec, visualDensity: tc.vd, localInputTheme: tc.local})
			wantNear(t, "container height", containerHeight(ts), tc.want)
		})
	}
}

func TestInputDecoratorLabelLayout(t *testing.T) {
	t.Run("the label appears above the input", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: InputDecoration{LabelText: someStr(testLabelText)}})
		// 8 top padding, 12 floating label (font size 16 * 0.75, line height forced to 1), 4 gap between the label and the
		// input, 24 input text (font size 16, line height 1.5), 8 bottom padding.
		if got := decoratorSize(ts); got != (ui.Size{Width: 800, Height: 56}) {
			t.Errorf("size = %v, want 800 x 56", got)
		}
		wantNear(t, "label top", labelRect(ts).Top, 8)
		wantNear(t, "label bottom", labelRect(ts).Bottom, 20)
		wantNear(t, "input top", inputRect(ts).Top, 24)
		wantNear(t, "input bottom", inputRect(ts).Bottom, 48)
	})
	t.Run("the label appears within the input when there is no text", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: true, decoration: InputDecoration{LabelText: someStr(testLabelText)}})
		if got := decoratorSize(ts); got != (ui.Size{Width: 800, Height: 56}) {
			t.Errorf("size = %v, want 800 x 56", got)
		}
		// The label line height is forced to 1 and its font size is 16: it is vertically centered (20 above and below).
		wantNear(t, "label top", labelRect(ts).Top, 20)
		wantNear(t, "label bottom", labelRect(ts).Bottom, 36)
	})
	t.Run("the label floats when there is no content and the behavior is always", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: true, decoration: InputDecoration{
			LabelText: someStr(testLabelText), FloatingLabelBehavior: FloatingLabelBehaviorAlways}})
		wantNear(t, "label top", labelRect(ts).Top, 8)
		wantNear(t, "label bottom", labelRect(ts).Bottom, 20)
	})
	t.Run("the label stays inline when there is content and the behavior is never", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: InputDecoration{
			LabelText: someStr(testLabelText), FloatingLabelBehavior: FloatingLabelBehaviorNever}})
		wantNear(t, "label top", labelRect(ts).Top, 20)
		wantNear(t, "label bottom", labelRect(ts).Bottom, 36)
	})
	t.Run("the behavior of the ambient theme is used", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), inputDecorationTheme: &InputDecorationThemeData{FloatingLabelBehavior: FloatingLabelBehaviorNever},
			decoration: InputDecoration{Label: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("label"),
				Width: foundation.Some(50.0), Height: foundation.Some(16.0)})}})
		r := rectOf(ts, gluttertest.ByKey(foundation.NewValueKey("label")))
		wantNear(t, "label top", r.Top, 20)
		wantNear(t, "label bottom", r.Bottom, 36)
	})
}

func TestInputDecoratorFloatingLabelAnimation(t *testing.T) {
	pump := func(ts *gluttertest.Tester, focused bool) {
		ts.PumpWidget(buildInputDecorator(decoratorCase{theme: m3Standard(), isEmpty: true, isFocused: focused, decoration: InputDecoration{
			LabelText: someStr(testLabelText), FloatingLabelBehavior: FloatingLabelBehaviorAuto}}))
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	pump(ts, false)
	wantNear(t, "label top", labelRect(ts).Top, 20)
	// The label animates upwards and scales down: 167ms, fastOutSlowIn.
	pump(ts, true)
	for _, step := range []struct {
		ms   int
		want float64
	}{{42, 17.09}, {42, 10.66}, {42, 8.47}, {41, 8.0}} {
		ts.Pump(time.Duration(step.ms) * time.Millisecond)
		if got := labelRect(ts).Top; got < step.want-0.5 || got > step.want+0.5 {
			t.Errorf("label top going up = %v, want %v", got, step.want)
		}
	}
	// The pumpAndSettle keeps the animation from changing direction in the middle.
	ts.PumpAndSettle()
	pump(ts, false)
	for _, step := range []struct {
		ms   int
		want float64
	}{{42, 10.90}, {42, 17.34}, {42, 19.69}, {41, 20.0}} {
		ts.Pump(time.Duration(step.ms) * time.Millisecond)
		if got := labelRect(ts).Top; got < step.want-0.5 || got > step.want+0.5 {
			t.Errorf("label top going down = %v, want %v", got, step.want)
		}
	}
}

func TestInputDecoratorWithdrawsLabel(t *testing.T) {
	// Flutter's tests use a font whose "Label" is 82.5 wide; the width here is the one of the
	// inline label of the bundled font, measured below.
	const labelH = 16.0
	inline := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: true, decoration: InputDecoration{
		LabelText: someStr("Label"), FocusedBorder: NewOutlineInputBorder(), DisabledBorder: NewOutlineInputBorder(), Border: NewOutlineInputBorder()}})
	inline.Pump(167 * time.Millisecond)
	labelW := labelRect(inline).Right - labelRect(inline).Left
	tests := []struct {
		name                    string
		focused, enabled, empty bool
		floats                  bool
	}{
		{"unfocused and empty", false, true, true, false},
		{"unfocused and not empty", false, true, false, true},
		{"focused and empty", true, true, true, true},
		{"focused and not empty", true, true, false, true},
		{"disabled and empty", false, false, true, false},
		{"disabled and not empty", false, false, false, true},
		{"focused, disabled and empty", true, false, true, false},
		{"focused, disabled and not empty", true, false, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: tc.empty, isFocused: tc.focused, decoration: InputDecoration{
				LabelText: someStr("Label"), Disabled: !tc.enabled, FocusedBorder: NewOutlineInputBorder(), DisabledBorder: NewOutlineInputBorder(),
				Border: NewOutlineInputBorder()}})
			ts.Pump(167 * time.Millisecond)
			r := labelRect(ts)
			if tc.floats {
				wantRect(t, "floating label", r, 16, -5.5, 16+labelW*0.75, -5.5+labelH*0.75)
			} else {
				wantRect(t, "inline label", r, 16, 20, 16+labelW, 20+labelH)
			}
		})
	}
}

func TestInputDecoratorFloatingLabelY(t *testing.T) {
	// Regression test for flutter/flutter#54028: the label is centered on the top
	// border line, floatingLabelHeight = 12 (16 * 0.75), labelY = -12/2 +
	// borderWidth/2... with the stroke offset of the 4 pixel border.
	ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: true, decoration: InputDecoration{
		LabelText:             someStr(testLabelText),
		EnabledBorder:         NewOutlineInputBorder(OutlineInputBorderOpts{BorderSide: painting.NewBorderSide(painting.BorderSideOpts{Width: foundation.Some(4.0)})}),
		FloatingLabelBehavior: FloatingLabelBehaviorAlways,
	}})
	ts.Pump(167 * time.Millisecond)
	wantNear(t, "label top", labelRect(ts).Top, -4)
}

func hintOpacity(ts *gluttertest.Tester) float64 { return opacityOf(ts, testHintText) }

func TestInputDecoratorHint(t *testing.T) {
	type states struct {
		name                        string
		disabled, hover, focus, err bool
	}
	allStates := []states{
		{name: "enabled"}, {name: "disabled", disabled: true}, {name: "hovered", hover: true},
		{name: "focused", focus: true}, {name: "error", err: true},
	}
	for _, outlined := range []bool{false, true} {
		for _, withLabel := range []bool{false, true} {
			for _, st := range allStates {
				for _, empty := range []bool{true, false} {
					name := "filled"
					if outlined {
						name = "outlined"
					}
					if withLabel {
						name += " with label"
					} else {
						name += " without label"
					}
					name += ", " + st.name
					if empty {
						name += ", empty"
					} else {
						name += ", not empty"
					}
					t.Run(name, func(t *testing.T) {
						dec := InputDecoration{HintText: someStr(testHintText), Disabled: st.disabled}
						if st.err {
							dec.ErrorText = someStr(testErrorText)
						}
						if withLabel {
							dec.LabelText = someStr(testLabelText)
						}
						wantContainer, wantInputTop := 48.0, 12.0
						switch {
						case outlined:
							dec.Border = NewOutlineInputBorder()
							wantContainer, wantInputTop = 56, 16
						default:
							dec.Filled = foundation.Some(true)
							if withLabel {
								wantContainer, wantInputTop = 56, 24
							}
						}
						ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: empty, isFocused: st.focus, isHovering: st.hover, decoration: dec})
						wantNear(t, "container height", containerHeight(ts), wantContainer)
						wantNear(t, "input top", inputRect(ts).Top, wantInputTop)
						wantNear(t, "input bottom", inputRect(ts).Bottom, wantInputTop+24)
						wantNear(t, "hint top", hintRect(ts).Top, inputRect(ts).Top)
						wantNear(t, "hint bottom", hintRect(ts).Bottom, inputRect(ts).Bottom)
						// The hint is visible when the decorator is empty and the label, if any, floats.
						want := 0.0
						if empty && (!withLabel || st.focus) {
							want = 1.0
						}
						wantNear(t, "hint opacity", hintOpacity(ts), want)
					})
				}
			}
		}
	}
}

func TestInputDecoratorHintStyle(t *testing.T) {
	tests := []struct {
		name     string
		disabled bool
		color    func(c *ColorScheme) ui.Color
	}{
		{"enabled", false, func(c *ColorScheme) ui.Color { return c.OnSurfaceVariant() }},
		{"disabled", true, func(c *ColorScheme) ui.Color { return c.OnSurface().WithOpacity(0.38) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: true,
				decoration: InputDecoration{Filled: foundation.Some(true), HintText: someStr(testHintText), Disabled: tc.disabled}})
			theme := ThemeOf(ts.Element(findDecorator()))
			want := theme.TextTheme.BodyLarge.CopyWith(painting.TextStyleCopy{Color: foundation.Some(tc.color(theme.ColorScheme))})
			got := ts.Widget(findHint()).(*widgets.Text).Style()
			if !got.Equal(want) {
				t.Errorf("hint style = %v, want %v", got, want)
			}
		})
	}
}

func TestInputDecoratorHintSize(t *testing.T) {
	tests := []struct {
		name       string
		noMaintain bool
	}{
		{"the height stays at the hint height by default", false},
		{"the height collapses from the hint height when the size is not maintained", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// A tall hint, and a short input.
			ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: false, decoration: InputDecoration{
				Hint: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(48.0)}), NoMaintainHintSize: tc.noMaintain},
				child: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(24.0)})})
			got := decoratorSize(ts).Height
			want := 48.0 + 16 // hint 48 plus the vertical padding of 16
			if tc.noMaintain {
				want = 48.0 // the input of 24 plus the padding of 16 is less than the minimum height of 48
			}
			wantNear(t, "decorator height", got, want)
		})
	}
}

func TestInputDecoratorHelperCounterError(t *testing.T) {
	const (
		containerH   = 56.0
		helperGap    = 4.0
		helperHeight = 16.0
		fullHeight   = containerH + helperGap + helperHeight
	)
	states := []struct {
		name                   string
		disabled, hover, focus bool
		color                  func(c *ColorScheme) ui.Color
	}{
		{"enabled", false, false, false, func(c *ColorScheme) ui.Color { return c.OnSurfaceVariant() }},
		{"disabled", true, false, false, func(c *ColorScheme) ui.Color { return c.OnSurface().WithOpacity(0.38) }},
		{"hovered", false, true, false, func(c *ColorScheme) ui.Color { return c.OnSurfaceVariant() }},
		{"focused", false, false, true, func(c *ColorScheme) ui.Color { return c.OnSurfaceVariant() }},
	}
	for _, outlined := range []bool{false, true} {
		for _, st := range states {
			name := "filled, " + st.name
			if outlined {
				name = "outlined, " + st.name
			}
			t.Run(name, func(t *testing.T) {
				dec := InputDecoration{LabelText: someStr(testLabelText), HelperText: someStr(testHelperText), CounterText: someStr(testCounterText), Disabled: st.disabled}
				if outlined {
					dec.Border = NewOutlineInputBorder()
				} else {
					dec.Filled = foundation.Some(true)
				}
				ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isFocused: st.focus, isHovering: st.hover, decoration: dec})
				wantNear(t, "decorator height", decoratorSize(ts).Height, fullHeight)
				wantNear(t, "container height", containerHeight(ts), containerH)
				helper, counter := rectOf(ts, findHelper()), rectOf(ts, findCounter())
				wantNear(t, "helper top", helper.Top, containerH+helperGap)
				wantNear(t, "helper height", helper.Bottom-helper.Top, helperHeight)
				wantNear(t, "helper left", helper.Left, 16)
				wantNear(t, "counter top", counter.Top, containerH+helperGap)
				wantNear(t, "counter height", counter.Bottom-counter.Top, helperHeight)
				wantNear(t, "counter right", counter.Right, 800-16)
				theme := ThemeOf(ts.Element(findDecorator()))
				want := theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(st.color(theme.ColorScheme))})
				if got := ts.Widget(findHelper()).(*widgets.Text).Style(); !got.Equal(want) {
					t.Errorf("helper style = %v, want %v", got, want)
				}
				if got := ts.Widget(findCounter()).(*widgets.Text).Style(); !got.Equal(want) {
					t.Errorf("counter style = %v, want %v", got, want)
				}
			})
		}
	}
}

func TestInputDecoratorErrorReplacesHelper(t *testing.T) {
	pump := func(ts *gluttertest.Tester, errText foundation.Opt[string]) {
		ts.PumpWidget(buildInputDecorator(decoratorCase{theme: m3Standard(), decoration: InputDecoration{
			Filled: foundation.Some(true), LabelText: someStr(testLabelText), HelperText: someStr(testHelperText), ErrorText: errText,
			CounterText: someStr(testCounterText)}}))
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	pump(ts, foundation.None[string]())
	if !ts.Exists(findHelper()) || ts.Exists(findError()) {
		t.Fatal("the helper is not alone")
	}
	pump(ts, someStr(testErrorText))
	ts.Pump(20 * time.Millisecond)
	// While the fade is running, both are shown.
	if !ts.Exists(findHelper()) || !ts.Exists(findError()) {
		t.Errorf("during the fade: helper %v, error %v", ts.Exists(findHelper()), ts.Exists(findError()))
	}
	ts.PumpAndSettle()
	if ts.Exists(findHelper()) || !ts.Exists(findError()) {
		t.Errorf("after the fade: helper %v, error %v", ts.Exists(findHelper()), ts.Exists(findError()))
	}
	wantNear(t, "error opacity", opacityOf(ts, testErrorText), 1)
	wantNear(t, "error top", rectOf(ts, findError()).Top, 56+4)
	theme := ThemeOf(ts.Element(findDecorator()))
	want := theme.TextTheme.BodySmall.CopyWith(painting.TextStyleCopy{Color: foundation.Some(theme.ColorScheme.Error())})
	if got := ts.Widget(findError()).(*widgets.Text).Style(); !got.Equal(want) {
		t.Errorf("error style = %v, want %v", got, want)
	}
	pump(ts, foundation.None[string]())
	ts.PumpAndSettle()
	if !ts.Exists(findHelper()) || ts.Exists(findError()) {
		t.Errorf("after the error goes: helper %v, error %v", ts.Exists(findHelper()), ts.Exists(findError()))
	}
}

func containerRect(ts *gluttertest.Tester) ui.Rect {
	size := containerBox(ts).Size()
	tl := containerBox(ts).LocalToGlobal(ui.OffsetZero, nil)
	return ui.NewRectFromLTWH(tl.Dx, tl.Dy, size.Width, size.Height)
}

func TestInputDecoratorDense(t *testing.T) {
	const extra = 4.0
	t.Run("an outlined decoration", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: InputDecoration{
			Border: NewOutlineInputBorder(), LabelText: someStr(testLabelText), IsDense: foundation.Some(true)}})
		c, in := containerRect(ts), inputRect(ts)
		// The content padding is (12, 16, 12, 8): the vertical padding affects the container height, not the input
		// vertical position as the input is centered.
		wantNear(t, "container height", c.Height(), 48)
		wantNear(t, "input center", (in.Top+in.Bottom)/2, (c.Top+c.Bottom)/2)
		wantNear(t, "input left", in.Left, c.Left+12+extra)
		wantNear(t, "input right", in.Right, c.Right-12-extra)
	})
	t.Run("a filled decoration", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: InputDecoration{
			Filled: foundation.Some(true), LabelText: someStr(testLabelText), IsDense: foundation.Some(true)}})
		c, in, l := containerRect(ts), inputRect(ts), labelRect(ts)
		wantNear(t, "container height", c.Height(), 48)
		wantNear(t, "label top", l.Top, c.Top+4)
		wantNear(t, "input bottom", in.Bottom, c.Bottom-4)
		wantNear(t, "input left", in.Left, c.Left+12+extra)
		wantNear(t, "input right", in.Right, c.Right-12-extra)
	})
	t.Run("an underlined decoration", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: InputDecoration{
			LabelText: someStr(testLabelText), IsDense: foundation.Some(true)}})
		c, in, l := containerRect(ts), inputRect(ts), labelRect(ts)
		wantNear(t, "container height", c.Height(), 48)
		wantNear(t, "label top", l.Top, c.Top+4)
		wantNear(t, "input bottom", in.Bottom, c.Bottom-4)
		wantNear(t, "input left", in.Left, c.Left)
		wantNear(t, "input right", in.Right, c.Right)
	})
	t.Run("the ambient theme is used", func(t *testing.T) {
		ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), inputDecorationTheme: &InputDecorationThemeData{IsDense: true},
			decoration: InputDecoration{LabelText: someStr(testLabelText)}})
		wantNear(t, "container height", containerRect(ts).Height(), 48)
	})
}

func TestInputDecoratorCollapsed(t *testing.T) {
	const inputHeight = 24.0
	ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: NewInputDecorationCollapsed(someStr(testHintText))})
	if got := decoratorSize(ts); got != (ui.Size{Width: 800, Height: inputHeight}) {
		t.Errorf("size = %v, want 800 x 24", got)
	}
	wantNear(t, "input height", inputRect(ts).Bottom-inputRect(ts).Top, inputHeight)
	wantNear(t, "input top", inputRect(ts).Top, 0)
	wantNear(t, "hint opacity", hintOpacity(ts), 0)
	wantNear(t, "border weight", getBorderSide(ts).Width, 0)

	// The hint appears.
	ts.PumpWidget(buildInputDecorator(decoratorCase{theme: m3Standard(), isEmpty: true, isFocused: true,
		decoration: NewInputDecorationCollapsed(someStr(testHintText))}))
	ts.PumpAndSettle()
	if got := decoratorSize(ts); got != (ui.Size{Width: 800, Height: inputHeight}) {
		t.Errorf("size = %v, want 800 x 24", got)
	}
	wantNear(t, "hint opacity", hintOpacity(ts), 1)
	wantNear(t, "hint height", hintRect(ts).Bottom-hintRect(ts).Top, inputHeight)
	wantNear(t, "hint top", hintRect(ts).Top, 0)

	// The constraints are accepted.
	ts = pumpDecorator(t, decoratorCase{theme: m3Standard(), decoration: NewInputDecorationCollapsed(someStr(testHintText), InputDecorationCollapsedOpts{
		Constraints: foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(200.0), Height: foundation.Some(32.0)}))})})
	if got := decoratorSize(ts); got != (ui.Size{Width: 200, Height: 32}) {
		t.Errorf("size with constraints = %v, want 200 x 32", got)
	}
}

func TestInputDecoratorAffixIcons(t *testing.T) {
	prefixIcon, suffixIcon := widgets.IconData(Icons.Search()), widgets.IconData(Icons.CancelOutlined())
	for _, rtl := range []bool{false, true} {
		dirName := "LTR"
		dir := ui.TextDirectionLTR
		if rtl {
			dirName, dir = "RTL", ui.TextDirectionRTL
		}
		t.Run("the prefix icon is positioned, "+dirName, func(t *testing.T) {
			ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), textDirection: dir, decoration: InputDecoration{
				Filled: foundation.Some(true), LabelText: someStr(testLabelText), PrefixIcon: widgets.NewIcon(prefixIcon)}})
			icon := rectOf(ts, gluttertest.ByIcon(prefixIcon))
			// By default, the prefix icon is in a 48 x 48 box, vertically centered inside the container.
			wantNear(t, "icon width", icon.Right-icon.Left, 48)
			wantNear(t, "icon height", icon.Bottom-icon.Top, 48)
			wantNear(t, "icon center", (icon.Top+icon.Bottom)/2, (containerRect(ts).Top+containerRect(ts).Bottom)/2)
			in, dec := inputRect(ts), decoratorSize(ts)
			if rtl {
				// The padding is 12 and the gap between the icon glyph (24 in the 48 box) and the input is 16.
				wantNear(t, "right padding", dec.Width-icon.Right, 0)
				wantNear(t, "gap to the input", (icon.Left+12)-in.Right, 16)
			} else {
				wantNear(t, "left padding", icon.Left+12, 12)
				wantNear(t, "gap to the input", in.Left-(icon.Right-12), 16)
			}
		})
		t.Run("the suffix icon is positioned, "+dirName, func(t *testing.T) {
			ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), textDirection: dir, decoration: InputDecoration{
				Filled: foundation.Some(true), LabelText: someStr(testLabelText), SuffixIcon: widgets.NewIcon(suffixIcon)}})
			icon := rectOf(ts, gluttertest.ByIcon(suffixIcon))
			wantNear(t, "icon width", icon.Right-icon.Left, 48)
			wantNear(t, "icon height", icon.Bottom-icon.Top, 48)
			wantNear(t, "icon center", (icon.Top+icon.Bottom)/2, (containerRect(ts).Top+containerRect(ts).Bottom)/2)
			in, dec := inputRect(ts), decoratorSize(ts)
			if rtl {
				wantNear(t, "left padding", icon.Left, 0)
				wantNear(t, "gap to the input", in.Left-(icon.Right-12), 16)
			} else {
				wantNear(t, "right padding", dec.Width-icon.Right, 0)
				wantNear(t, "gap to the input", (icon.Left+12)-in.Right, 16)
			}
		})
	}
	colors := []struct {
		name     string
		disabled bool
		hover    bool
		want     func(c *ColorScheme) ui.Color
	}{
		{"enabled", false, false, func(c *ColorScheme) ui.Color { return c.OnSurfaceVariant() }},
		{"hovered", false, true, func(c *ColorScheme) ui.Color { return c.OnSurfaceVariant() }},
		{"disabled", true, false, func(c *ColorScheme) ui.Color { return c.OnSurface().WithOpacity(0.38) }},
	}
	for _, tc := range colors {
		t.Run("the icons have the color of the state, "+tc.name, func(t *testing.T) {
			ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isHovering: tc.hover, decoration: InputDecoration{
				Filled: foundation.Some(true), LabelText: someStr(testLabelText), Disabled: tc.disabled,
				PrefixIcon: widgets.NewIcon(prefixIcon), SuffixIcon: widgets.NewIcon(suffixIcon)}})
			want := tc.want(ThemeOf(ts.Element(findDecorator())).ColorScheme)
			for _, icon := range []widgets.IconData{prefixIcon, suffixIcon} {
				got := widgets.IconThemeOf(ts.Element(gluttertest.ByIcon(icon))).Color().OrElse(ui.Color{})
				if got != want {
					t.Errorf("icon %v color = %v, want %v", icon, got, want)
				}
			}
		})
	}
}

func TestInputDecoratorAffixText(t *testing.T) {
	ts := pumpDecorator(t, decoratorCase{theme: m3Standard(), isEmpty: true, decoration: InputDecoration{
		LabelText: someStr(testLabelText), PrefixText: someStr("pre"), SuffixText: someStr("suf")}})
	// The prefix and the suffix are not visible when the decorator is empty and the label is inline.
	for _, text := range []string{"pre", "suf"} {
		wantNear(t, text+" opacity", opacityOf2(ts, text), 0)
	}
	ts.PumpWidget(buildInputDecorator(decoratorCase{theme: m3Standard(), isEmpty: true, isFocused: true, decoration: InputDecoration{
		LabelText: someStr(testLabelText), PrefixText: someStr("pre"), SuffixText: someStr("suf")}}))
	ts.PumpAndSettle()
	for _, text := range []string{"pre", "suf"} {
		wantNear(t, text+" opacity", opacityOf2(ts, text), 1)
	}
}

// opacityOf2 is the opacity of the closest AnimatedOpacity above the text.
func opacityOf2(ts *gluttertest.Tester, text string) float64 {
	f := gluttertest.First(gluttertest.Ancestor(gluttertest.ByText(text), gluttertest.ByType[*widgets.AnimatedOpacity]()))
	// AnimatedOpacity builds a FadeTransition below it.
	g := gluttertest.First(gluttertest.Descendant(f, gluttertest.ByType[*widgets.FadeTransition]()))
	return ts.Widget(g).(*widgets.FadeTransition).Opacity().Value()
}
