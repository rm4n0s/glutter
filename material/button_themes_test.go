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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// styleTheme is what the button themes that hold only a style have in common.
type styleTheme struct {
	name    string
	make    func(*ButtonStyle) any
	lerp    func(a, b *ButtonStyle, t float64) *ButtonStyle
	equal   func(a, b *ButtonStyle) bool
	styled  func(*ButtonStyle) widgets.Widget // a widget of the kind with the theme above it
	of      func(bc widgets.BuildContext) *ButtonStyle
	inTheme func(*ThemeData, *ButtonStyle) *ThemeData
	str     func(*ButtonStyle) string
	button  gluttertest.Finder
}

var styleThemes = []styleTheme{
	{"TextButtonTheme",
		func(s *ButtonStyle) any { return NewTextButtonThemeData(TextButtonThemeData{Style: s}) },
		func(a, b *ButtonStyle, t float64) *ButtonStyle {
			return TextButtonThemeDataLerp(TextButtonThemeData{Style: a}, TextButtonThemeData{Style: b}, t).Style
		},
		func(a, b *ButtonStyle) bool {
			return TextButtonThemeData{Style: a}.Equal(TextButtonThemeData{Style: b})
		},
		func(s *ButtonStyle) widgets.Widget {
			return NewTextButtonTheme(TextButtonThemeData{Style: s}, NewTextButton(func() {}, widgets.NewText("Go")))
		},
		func(bc widgets.BuildContext) *ButtonStyle { return TextButtonThemeOf(bc).Style },
		func(d *ThemeData, s *ButtonStyle) *ThemeData {
			return d.CopyWith(ThemeDataCopy{TextButtonTheme: foundation.Some(TextButtonThemeData{Style: s})})
		},
		func(s *ButtonStyle) string { return TextButtonThemeData{Style: s}.String() },
		gluttertest.ByType[*TextButton]()},
	{"ElevatedButtonTheme",
		func(s *ButtonStyle) any { return NewElevatedButtonThemeData(ElevatedButtonThemeData{Style: s}) },
		func(a, b *ButtonStyle, t float64) *ButtonStyle {
			return ElevatedButtonThemeDataLerp(ElevatedButtonThemeData{Style: a}, ElevatedButtonThemeData{Style: b}, t).Style
		},
		func(a, b *ButtonStyle) bool {
			return ElevatedButtonThemeData{Style: a}.Equal(ElevatedButtonThemeData{Style: b})
		},
		func(s *ButtonStyle) widgets.Widget {
			return NewElevatedButtonTheme(ElevatedButtonThemeData{Style: s}, NewElevatedButton(func() {}, widgets.NewText("Go")))
		},
		func(bc widgets.BuildContext) *ButtonStyle { return ElevatedButtonThemeOf(bc).Style },
		func(d *ThemeData, s *ButtonStyle) *ThemeData {
			return d.CopyWith(ThemeDataCopy{ElevatedButtonTheme: foundation.Some(ElevatedButtonThemeData{Style: s})})
		},
		func(s *ButtonStyle) string { return ElevatedButtonThemeData{Style: s}.String() },
		gluttertest.ByType[*ElevatedButton]()},
	{"FilledButtonTheme",
		func(s *ButtonStyle) any { return NewFilledButtonThemeData(FilledButtonThemeData{Style: s}) },
		func(a, b *ButtonStyle, t float64) *ButtonStyle {
			return FilledButtonThemeDataLerp(FilledButtonThemeData{Style: a}, FilledButtonThemeData{Style: b}, t).Style
		},
		func(a, b *ButtonStyle) bool {
			return FilledButtonThemeData{Style: a}.Equal(FilledButtonThemeData{Style: b})
		},
		func(s *ButtonStyle) widgets.Widget {
			return NewFilledButtonTheme(FilledButtonThemeData{Style: s}, NewFilledButton(func() {}, widgets.NewText("Go")))
		},
		func(bc widgets.BuildContext) *ButtonStyle { return FilledButtonThemeOf(bc).Style },
		func(d *ThemeData, s *ButtonStyle) *ThemeData {
			return d.CopyWith(ThemeDataCopy{FilledButtonTheme: foundation.Some(FilledButtonThemeData{Style: s})})
		},
		func(s *ButtonStyle) string { return FilledButtonThemeData{Style: s}.String() },
		gluttertest.ByType[*FilledButton]()},
	{"OutlinedButtonTheme",
		func(s *ButtonStyle) any { return NewOutlinedButtonThemeData(OutlinedButtonThemeData{Style: s}) },
		func(a, b *ButtonStyle, t float64) *ButtonStyle {
			return OutlinedButtonThemeDataLerp(OutlinedButtonThemeData{Style: a}, OutlinedButtonThemeData{Style: b}, t).Style
		},
		func(a, b *ButtonStyle) bool {
			return OutlinedButtonThemeData{Style: a}.Equal(OutlinedButtonThemeData{Style: b})
		},
		func(s *ButtonStyle) widgets.Widget {
			return NewOutlinedButtonTheme(OutlinedButtonThemeData{Style: s}, NewOutlinedButton(func() {}, widgets.NewText("Go")))
		},
		func(bc widgets.BuildContext) *ButtonStyle { return OutlinedButtonThemeOf(bc).Style },
		func(d *ThemeData, s *ButtonStyle) *ThemeData {
			return d.CopyWith(ThemeDataCopy{OutlinedButtonTheme: foundation.Some(OutlinedButtonThemeData{Style: s})})
		},
		func(s *ButtonStyle) string { return OutlinedButtonThemeData{Style: s}.String() },
		gluttertest.ByType[*OutlinedButton]()},
	{"IconButtonTheme",
		func(s *ButtonStyle) any { return NewIconButtonThemeData(IconButtonThemeData{Style: s}) },
		func(a, b *ButtonStyle, t float64) *ButtonStyle {
			return IconButtonThemeDataLerp(IconButtonThemeData{Style: a}, IconButtonThemeData{Style: b}, t).Style
		},
		func(a, b *ButtonStyle) bool {
			return IconButtonThemeData{Style: a}.Equal(IconButtonThemeData{Style: b})
		},
		func(s *ButtonStyle) widgets.Widget {
			return NewIconButtonTheme(IconButtonThemeData{Style: s}, NewIconButton(func() {}, NewIcon(Icons.Add())))
		},
		func(bc widgets.BuildContext) *ButtonStyle { return IconButtonThemeOf(bc).Style },
		func(d *ThemeData, s *ButtonStyle) *ThemeData {
			return d.CopyWith(ThemeDataCopy{IconButtonTheme: foundation.Some(IconButtonThemeData{Style: s})})
		},
		func(s *ButtonStyle) string { return IconButtonThemeData{Style: s}.String() },
		gluttertest.ByType[*IconButton]()},
}

func TestStyleThemes(t *testing.T) {
	a := NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(2.0)})
	b := NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(6.0)})
	states := widgets.NewWidgetStates()
	for _, th := range styleThemes {
		t.Run(th.name, func(t *testing.T) {
			if !th.equal(nil, nil) || th.equal(a, nil) || th.equal(a, b) || !th.equal(a, NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(2.0)})) {
				t.Error("Equal does not compare the styles")
			}
			if got := th.lerp(a, b, 0.5); got.Elevation.Resolve(states) != foundation.Some(4.0) {
				t.Errorf("lerp elevation = %v, want 4", got.Elevation.Resolve(states))
			}
			if got := th.lerp(nil, nil, 0.5); got != nil {
				t.Errorf("lerp of nothing = %v", got)
			}
			if foundation.DebugMode {
				if !strings.Contains(th.str(a), "style") {
					t.Errorf("the description has no style: %q", th.str(a))
				}
				if got := th.str(nil); strings.Contains(got, "style:") {
					t.Errorf("the description of an empty theme has a style: %q", got)
				}
			}
		})
	}
}

func TestStyleThemesAreUsed(t *testing.T) {
	style := NewButtonStyle(ButtonStyleOpts{MinimumSize: allOpt(ui.Size{Width: 200, Height: 80})})
	other := NewButtonStyle(ButtonStyleOpts{MinimumSize: allOpt(ui.Size{Width: 120, Height: 60})})
	size := func(ts *gluttertest.Tester, th styleTheme) ui.Size {
		return ts.Size(gluttertest.First(gluttertest.Descendant(th.button, gluttertest.ByType[*Material]())))
	}
	for _, th := range styleThemes {
		t.Run(th.name, func(t *testing.T) {
			// A theme above the button.
			ts := pumpButton(t, standardTheme(true), th.styled(style))
			if got := size(ts, th); got != (ui.Size{Width: 200, Height: 80}) {
				t.Errorf("size with the inherited theme = %v, want 200x80", got)
			}
			// The ThemeData has one too, and the inherited theme wins over it.
			var fromContext *ButtonStyle
			pumpButton(t, th.inTheme(standardTheme(true), style), widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				fromContext = th.of(bc)
				return widgets.NewSizedBox(nil)
			}))
			if !style.Equal(fromContext) {
				t.Errorf("the style of the ThemeData is %v, want %v", fromContext, style)
			}
			ts = pumpButton(t, th.inTheme(standardTheme(true), other), th.styled(style))
			if got := size(ts, th); got != (ui.Size{Width: 200, Height: 80}) {
				t.Errorf("size = %v, want the one of the inherited theme", got)
			}
		})
	}
}

func TestTextButtonThemeWinsOverThemeData(t *testing.T) {
	inner := NewButtonStyle(ButtonStyleOpts{MinimumSize: allOpt(ui.Size{Width: 100, Height: 70})})
	outer := NewButtonStyle(ButtonStyleOpts{MinimumSize: allOpt(ui.Size{Width: 300, Height: 90})})
	theme := standardTheme(true).CopyWith(ThemeDataCopy{TextButtonTheme: foundation.Some(TextButtonThemeData{Style: outer})})
	ts := pumpButton(t, theme, NewTextButtonTheme(TextButtonThemeData{Style: inner}, NewTextButton(func() {}, widgets.NewText("Go"))))
	if got := ts.Size(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*TextButton](), gluttertest.ByType[*Material]()))); got != (ui.Size{Width: 100, Height: 70}) {
		t.Errorf("size = %v, want the one of the TextButtonTheme", got)
	}
	ts = pumpButton(t, theme, NewTextButton(func() {}, widgets.NewText("Go")))
	if got := ts.Size(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*TextButton](), gluttertest.ByType[*Material]()))); got != (ui.Size{Width: 300, Height: 90}) {
		t.Errorf("size = %v, want the one of the ThemeData", got)
	}
	// The theme wraps itself again for a route that is shown below it.
	wrapped := NewTextButtonTheme(TextButtonThemeData{Style: inner}, widgets.NewSizedBox(nil)).Wrap(nil, widgets.NewSizedBox(nil))
	if w, ok := wrapped.(*TextButtonTheme); !ok || !w.Data().Style.Equal(inner) {
		t.Errorf("Wrap = %v", wrapped)
	}
}

func TestStyleThemeUpdateShouldNotify(t *testing.T) {
	a := NewTextButtonTheme(TextButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(1.0)})}, widgets.NewSizedBox(nil))
	same := NewTextButtonTheme(TextButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(1.0)})}, widgets.NewSizedBox(nil))
	other := NewTextButtonTheme(TextButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(2.0)})}, widgets.NewSizedBox(nil))
	if a.UpdateShouldNotify(same) {
		t.Error("an equal theme notifies")
	}
	if !a.UpdateShouldNotify(other) {
		t.Error("a different theme does not notify")
	}
}

func TestFloatingActionButtonThemeData(t *testing.T) {
	a := FloatingActionButtonThemeData{
		BackgroundColor: foundation.Some(ui.NewColor(0xFF000000)), Elevation: foundation.Some(2.0), IconSize: foundation.Some(20.0),
		SizeConstraints: foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(50.0), Height: foundation.Some(50.0)})),
		ExtendedPadding: painting.NewEdgeInsetsAll(10), EnableFeedback: foundation.Some(false), Shape: painting.NewCircleBorder()}
	b := FloatingActionButtonThemeData{
		BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), Elevation: foundation.Some(6.0), IconSize: foundation.Some(40.0),
		SizeConstraints: foundation.Some(rendering.NewBoxConstraintsTightFor(rendering.TightForOpts{Width: foundation.Some(90.0), Height: foundation.Some(90.0)})),
		ExtendedPadding: painting.NewEdgeInsetsAll(30), EnableFeedback: foundation.Some(true), Shape: painting.NewCircleBorder()}
	t.Run("equal and copyWith", func(t *testing.T) {
		if !a.Equal(a.CopyWith(FloatingActionButtonThemeDataCopy{})) || a.Equal(b) {
			t.Error("Equal is wrong")
		}
		c := a.CopyWith(FloatingActionButtonThemeDataCopy{Elevation: foundation.Some(9.0)})
		if c.Elevation.Value() != 9 || c.IconSize != a.IconSize || a.Equal(c) {
			t.Errorf("copy = %v", c)
		}
	})
	t.Run("lerp", func(t *testing.T) {
		got := FloatingActionButtonThemeDataLerp(a, b, 0.5)
		if got.Elevation != foundation.Some(4.0) || got.IconSize != foundation.Some(30.0) {
			t.Errorf("lerp = %v", got)
		}
		if got.BackgroundColor.Value() != ui.NewColor(0xFF000000).Lerp(ui.NewColor(0xFFFFFFFF), 0.5) {
			t.Errorf("lerp color = %v", got.BackgroundColor)
		}
		if c := got.SizeConstraints.Value(); c.MinWidth != 70 || c.MaxHeight != 70 {
			t.Errorf("lerp constraints = %v", c)
		}
		if got.ExtendedPadding.(painting.EdgeInsets).Left != 20 {
			t.Errorf("lerp padding = %v", got.ExtendedPadding)
		}
		if got := FloatingActionButtonThemeDataLerp(a, b, 0.4); got.EnableFeedback != foundation.Some(false) {
			t.Errorf("a bool changes at the middle: %v", got.EnableFeedback)
		}
		if got := FloatingActionButtonThemeDataLerp(a, b, 0.6); got.EnableFeedback != foundation.Some(true) {
			t.Errorf("a bool changes at the middle: %v", got.EnableFeedback)
		}
	})
	t.Run("diagnostics", func(t *testing.T) {
		skipUnlessDebug(t)
		s := a.String()
		for _, want := range []string{"backgroundColor", "elevation: 2.0", "iconSize: 20.0", "sizeConstraints"} {
			if !strings.Contains(s, want) {
				t.Errorf("the description lacks %q:\n%s", want, s)
			}
		}
	})
}

func TestSegmentedButtonAndActionIconThemes(t *testing.T) {
	icon := NewIcon(Icons.Add())
	a := SegmentedButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(2.0)}), SelectedIcon: icon}
	b := SegmentedButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(6.0)})}
	states := widgets.NewWidgetStates()
	if !a.Equal(a.CopyWith(SegmentedButtonThemeDataCopy{})) || a.Equal(b) {
		t.Error("Equal is wrong")
	}
	if got := SegmentedButtonThemeDataLerp(a, b, 0.5); got.Style.Elevation.Resolve(states) != foundation.Some(4.0) {
		t.Errorf("lerp = %v", got)
	}
	if SegmentedButtonThemeDataLerp(a, b, 0.4).SelectedIcon != icon || SegmentedButtonThemeDataLerp(a, b, 0.6).SelectedIcon != nil {
		t.Error("the selected icon does not change at the middle")
	}
	// The theme of the app gives the selected icon.
	custom := NewIcon(Icons.Favorite())
	theme := standardTheme(true).CopyWith(ThemeDataCopy{SegmentedButtonTheme: foundation.Some(SegmentedButtonThemeData{SelectedIcon: custom})})
	ts := pumpButton(t, theme, NewSegmentedButton(textSegments("A", "B"), []string{"A"}, func([]string) {}))
	if !ts.Exists(gluttertest.ByWidget(custom)) {
		t.Error("the selected icon of the theme is not shown")
	}

	// ActionIconTheme.
	builder := func(widgets.BuildContext) widgets.Widget { return widgets.NewText("custom back") }
	x := ActionIconThemeData{BackButtonIconBuilder: builder}
	if !x.Equal(x.CopyWith(ActionIconThemeDataCopy{})) || x.Equal(ActionIconThemeData{}) {
		t.Error("ActionIconThemeData.Equal is wrong")
	}
	if got := ActionIconThemeDataLerp(x, ActionIconThemeData{}, 0.6); got.BackButtonIconBuilder != nil {
		t.Error("the builder did not go at the middle")
	}
}

func TestActionButtonsUseIconButtonAndActionIconTheme(t *testing.T) {
	custom := ActionIconThemeData{
		CloseButtonIconBuilder:     func(widgets.BuildContext) widgets.Widget { return widgets.NewText("custom close") },
		DrawerButtonIconBuilder:    func(widgets.BuildContext) widgets.Widget { return widgets.NewText("custom drawer") },
		EndDrawerButtonIconBuilder: func(widgets.BuildContext) widgets.Widget { return widgets.NewText("custom end drawer") },
		BackButtonIconBuilder:      func(widgets.BuildContext) widgets.Widget { return widgets.NewText("custom back") },
	}
	theme := standardTheme(true).CopyWith(ThemeDataCopy{ActionIconTheme: foundation.Some(custom)})
	tests := []struct {
		name   string
		button widgets.Widget
		want   string
	}{
		{"back", NewBackButton(), "custom back"},
		{"close", NewCloseButton(), "custom close"},
		{"drawer", NewDrawerButton(), "custom drawer"},
		{"end drawer", NewEndDrawerButton(), "custom end drawer"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpButton(t, theme, tc.button)
			if !ts.Exists(gluttertest.ByText(tc.want)) {
				t.Errorf("the icon %q of the ActionIconTheme is not shown", tc.want)
			}
			if !ts.Exists(gluttertest.ByType[*IconButton]()) {
				t.Error("the button is not an IconButton")
			}
		})
	}
	t.Run("the style reaches the icon button", func(t *testing.T) {
		ts := pumpButton(t, standardTheme(true), NewCloseButton(ActionButtonOpts{
			Style: IconButtonStyleFrom(IconButtonStyleFromOpts{IconSize: foundation.Some(31.0)})}))
		it := widgets.IconThemeOf(ts.Element(gluttertest.First(gluttertest.Descendant(gluttertest.ByType[*IconButton](), gluttertest.ByType[*widgets.Icon]()))))
		if it.Size() != foundation.Some(31.0) {
			t.Errorf("icon size = %v, want 31", it.Size())
		}
	})
}
