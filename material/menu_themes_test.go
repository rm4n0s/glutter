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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestMenuStyle(t *testing.T) {
	states := widgets.NewWidgetStates()
	a := NewMenuStyle(MenuStyleOpts{
		BackgroundColor: allOpt(colorA), Elevation: allOpt(2.0), VisualDensity: foundation.Some(VisualDensityCompact()),
		Alignment: painting.AlignmentDirectionalTopEnd,
	})
	b := NewMenuStyle(MenuStyleOpts{BackgroundColor: allOpt(colorB), ShadowColor: allOpt(colorB), Elevation: allOpt(6.0), Alignment: painting.AlignmentBottomLeft})

	t.Run("copyWith: the argument wins", func(t *testing.T) {
		got := a.CopyWith(MenuStyleOpts{BackgroundColor: allOpt(colorB), Padding: widgets.NewWidgetStatePropertyAll[painting.EdgeInsetsGeometry](painting.NewEdgeInsetsAll(3))})
		if v := got.BackgroundColor.Resolve(states); v != foundation.Some(colorB) {
			t.Errorf("background = %v, want %v", v, colorB)
		}
		if got.Elevation.Resolve(states) != foundation.Some(2.0) || got.Padding == nil {
			t.Errorf("copy lost or dropped properties: %v", got)
		}
		if v := a.BackgroundColor.Resolve(states); v != foundation.Some(colorA) {
			t.Error("the original style was changed")
		}
	})
	t.Run("merge: the receiver wins", func(t *testing.T) {
		got := a.Merge(b)
		if v := got.BackgroundColor.Resolve(states); v != foundation.Some(colorA) {
			t.Errorf("background = %v, want the one of the receiver", v)
		}
		if v := got.ShadowColor.Resolve(states); v != foundation.Some(colorB) {
			t.Errorf("shadow = %v, want the one of the argument", v)
		}
		if got.Alignment != painting.AlignmentGeometry(painting.AlignmentDirectionalTopEnd) {
			t.Errorf("alignment = %v, want the one of the receiver", got.Alignment)
		}
		if got := a.Merge(nil); !got.Equal(a) {
			t.Errorf("a.Merge(nil) = %v, want a", got)
		}
	})
	t.Run("equality", func(t *testing.T) {
		tests := []struct {
			name string
			x, y *MenuStyle
			want bool
		}{
			{"both nil", nil, nil, true},
			{"nil and style", a, nil, false},
			{"equal but distinct", NewMenuStyle(MenuStyleOpts{Elevation: allOpt(2.0)}), NewMenuStyle(MenuStyleOpts{Elevation: allOpt(2.0)}), true},
			{"different", a, b, false},
		}
		for _, tc := range tests {
			if got := tc.x.Equal(tc.y); got != tc.want {
				t.Errorf("%s: Equal = %v, want %v", tc.name, got, tc.want)
			}
		}
	})
	t.Run("lerp", func(t *testing.T) {
		got := MenuStyleLerp(a, b, 0.5)
		if v := got.Elevation.Resolve(states); v != foundation.Some(4.0) {
			t.Errorf("elevation = %v, want 4", v)
		}
		if MenuStyleLerp(nil, nil, 0.5) != nil {
			t.Error("the lerp of nothing is not nil")
		}
		if v := MenuStyleLerp(a, b, 0.4).Alignment; v == nil || v.Equal(a.Alignment) || v.Equal(b.Alignment) {
			t.Errorf("alignment at 0.4 = %v, want a mix of both", v)
		}
	})
	t.Run("diagnostics", func(t *testing.T) {
		skipUnlessDebug(t)
		s := a.String()
		for _, want := range []string{"backgroundColor", "elevation", "visualDensity", "alignment"} {
			if !strings.Contains(s, want) {
				t.Errorf("the description lacks %q:\n%s", want, s)
			}
		}
	})
}

func TestMenuThemeData(t *testing.T) {
	style := NewMenuStyle(MenuStyleOpts{Elevation: allOpt(5.0)})
	icon := widgets.NewWidgetStatePropertyAll[widgets.Widget](widgets.NewSizedBox(nil))
	d := MenuThemeData{Style: style, SubmenuIcon: icon}
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"equal", d.Equal(MenuThemeData{Style: NewMenuStyle(MenuStyleOpts{Elevation: allOpt(5.0)}), SubmenuIcon: icon}), true},
		{"different style", d.Equal(MenuThemeData{Style: NewMenuStyle(MenuStyleOpts{Elevation: allOpt(6.0)}), SubmenuIcon: icon}), false},
		{"no icon", d.Equal(MenuThemeData{Style: style}), false},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	other := MenuThemeData{Style: NewMenuStyle(MenuStyleOpts{Elevation: allOpt(9.0)})}
	if got := MenuThemeDataLerp(d, other, 0.5).Style.Elevation.Resolve(widgets.NewWidgetStates()); got != foundation.Some(7.0) {
		t.Errorf("lerp elevation = %v, want 7", got)
	}
	if MenuThemeDataLerp(d, other, 0.4).SubmenuIcon == nil || MenuThemeDataLerp(d, other, 0.6).SubmenuIcon != nil {
		t.Error("the submenu icon does not change at the middle")
	}
	bar := MenuBarThemeData{Style: style}
	if !bar.Equal(MenuBarThemeData{Style: style}) || bar.Equal(MenuBarThemeData{}) {
		t.Error("MenuBarThemeData.Equal is wrong")
	}
	if MenuBarThemeDataLerp(bar, MenuBarThemeData{}, 0.5).Style == nil {
		t.Error("the lerp of a bar theme lost its style")
	}
	button := MenuButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(1.0)})}
	if !button.Equal(MenuButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(1.0)})}) || button.Equal(MenuButtonThemeData{}) {
		t.Error("MenuButtonThemeData.Equal is wrong")
	}
	if got := MenuButtonThemeDataLerp(button, MenuButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{Elevation: allOpt(3.0)})}, 0.5).Style.Elevation.Resolve(widgets.NewWidgetStates()); got != foundation.Some(2.0) {
		t.Errorf("button theme lerp elevation = %v, want 2", got)
	}
}

// TestMenuThemesAreUsed checks that the panel of a menu takes its style from the
// MenuTheme, the MenuBarTheme and the MenuStyle of the widget, in that order of
// precedence, and the items from the MenuButtonTheme.
func TestMenuThemesAreUsed(t *testing.T) {
	tests := []struct {
		name  string
		theme func(d *ThemeData) *ThemeData
		bar   func(m *menuTester) widgets.Widget
		open  string
		// check is run with the open menu.
		check func(t *testing.T, ts *menuTester)
	}{
		{"MenuBarTheme styles the bar",
			func(d *ThemeData) *ThemeData {
				return d.CopyWith(ThemeDataCopy{MenuBarTheme: foundation.Some(MenuBarThemeData{Style: NewMenuStyle(MenuStyleOpts{BackgroundColor: allOpt(colorA)})})})
			},
			func(m *menuTester) widgets.Widget { return NewMenuBar([]widgets.Widget{m.item("A")}) }, "",
			func(t *testing.T, ts *menuTester) {
				if got := ts.barColor(); got != colorA {
					t.Errorf("bar color = %v, want %v", got, colorA)
				}
			}},
		{"the style of the widget wins over the theme",
			func(d *ThemeData) *ThemeData {
				return d.CopyWith(ThemeDataCopy{MenuBarTheme: foundation.Some(MenuBarThemeData{Style: NewMenuStyle(MenuStyleOpts{BackgroundColor: allOpt(colorA)})})})
			},
			func(m *menuTester) widgets.Widget {
				return NewMenuBar([]widgets.Widget{m.item("A")}, MenuBarOpts{Style: NewMenuStyle(MenuStyleOpts{BackgroundColor: allOpt(colorB)})})
			}, "",
			func(t *testing.T, ts *menuTester) {
				if got := ts.barColor(); got != colorB {
					t.Errorf("bar color = %v, want %v", got, colorB)
				}
			}},
		{"MenuTheme styles the submenu panel",
			func(d *ThemeData) *ThemeData {
				return d.CopyWith(ThemeDataCopy{MenuTheme: foundation.Some(MenuThemeData{Style: NewMenuStyle(MenuStyleOpts{BackgroundColor: allOpt(colorB)})})})
			},
			func(m *menuTester) widgets.Widget {
				return NewMenuBar([]widgets.Widget{NewSubmenuButton([]widgets.Widget{m.item("Inside")}, widgets.NewText("Menu"))})
			}, "Menu",
			func(t *testing.T, ts *menuTester) {
				if got := ts.panelColorOf("Inside"); got != colorB {
					t.Errorf("panel color = %v, want %v", got, colorB)
				}
			}},
		{"MenuButtonTheme styles the items",
			func(d *ThemeData) *ThemeData {
				return d.CopyWith(ThemeDataCopy{MenuButtonTheme: foundation.Some(MenuButtonThemeData{Style: NewButtonStyle(ButtonStyleOpts{
					ForegroundColor: allOpt(colorA)})})})
			},
			func(m *menuTester) widgets.Widget {
				return NewMenuBar([]widgets.Widget{NewSubmenuButton([]widgets.Widget{m.item("Inside")}, widgets.NewText("Menu"))})
			}, "Menu",
			func(t *testing.T, ts *menuTester) {
				if got := ts.textColorOf("Inside"); got != colorA {
					t.Errorf("item text color = %v, want %v", got, colorA)
				}
			}},
		{"the submenu icon of the theme",
			func(d *ThemeData) *ThemeData {
				return d.CopyWith(ThemeDataCopy{MenuTheme: foundation.Some(MenuThemeData{
					SubmenuIcon: widgets.NewWidgetStatePropertyAll[widgets.Widget](widgets.NewSizedBox(widgets.NewText("SUB"), widgets.SizedBoxOpts{Width: foundation.Some(30.0)}))})})
			},
			func(m *menuTester) widgets.Widget {
				return NewMenuBar([]widgets.Widget{NewSubmenuButton([]widgets.Widget{
					NewSubmenuButton([]widgets.Widget{m.item("Deep")}, widgets.NewText("Nested"))}, widgets.NewText("Menu"))})
			}, "Menu",
			func(t *testing.T, ts *menuTester) {
				if !ts.Exists(gluttertest.ByText("SUB")) {
					t.Error("the submenu icon of the theme is not shown")
				}
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &menuTester{t: t}
			m.Tester = gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			m.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: tc.theme(standardTheme(true)), Home: NewScaffold(ScaffoldOpts{Body: tc.bar(m)})}))
			if tc.open != "" {
				m.Tap(gluttertest.ByText(tc.open))
				m.PumpAndSettle()
			}
			tc.check(t, m)
		})
	}
}
