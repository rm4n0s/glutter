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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestInputDecorationApplyDefaults(t *testing.T) {
	fill := ui.NewColor(0xFF112233)
	theme := NewInputDecorationThemeData(InputDecorationThemeDataOpts{
		Filled:                true,
		FillColor:             widgets.NewWidgetStatePropertyAll(fill),
		IsDense:               true,
		HelperMaxLines:        foundation.Some(3),
		FocusColor:            foundation.Some(ui.NewColor(0xFF445566)),
		Border:                NewUnderlineInputBorder(),
		FloatingLabelBehavior: FloatingLabelBehaviorAlways,
	})
	tests := []struct {
		name string
		in   InputDecoration
		want func(t *testing.T, d InputDecoration)
	}{
		{"fills the unset fields", InputDecoration{}, func(t *testing.T, d InputDecoration) {
			if d.Filled != foundation.Some(true) || d.IsDense != foundation.Some(true) || d.HelperMaxLines != foundation.Some(3) {
				t.Errorf("filled %v, dense %v, helper lines %v", d.Filled, d.IsDense, d.HelperMaxLines)
			}
			if d.FillColor == nil || d.FillColor.Resolve(0) != fill {
				t.Error("no fill color")
			}
			if _, ok := d.Border.(UnderlineInputBorder); !ok {
				t.Errorf("border = %v", d.Border)
			}
			if d.FloatingLabelBehavior != FloatingLabelBehaviorAlways {
				t.Errorf("floating label behavior = %v", d.FloatingLabelBehavior)
			}
		}},
		{"keeps the set fields", InputDecoration{Filled: foundation.Some(false), HelperMaxLines: foundation.Some(1), Border: NewOutlineInputBorder(),
			FloatingLabelBehavior: FloatingLabelBehaviorNever}, func(t *testing.T, d InputDecoration) {
			if d.Filled != foundation.Some(false) || d.HelperMaxLines != foundation.Some(1) {
				t.Errorf("filled %v, helper lines %v", d.Filled, d.HelperMaxLines)
			}
			if _, ok := d.Border.(OutlineInputBorder); !ok {
				t.Errorf("border = %v", d.Border)
			}
			if d.FloatingLabelBehavior != FloatingLabelBehaviorNever {
				t.Errorf("floating label behavior = %v", d.FloatingLabelBehavior)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { tc.want(t, tc.in.ApplyDefaults(theme)) })
	}
}

func TestInputDecorationCopyWithAndEquality(t *testing.T) {
	a := InputDecoration{LabelText: str("a"), HelperText: str("h")}
	b := a.CopyWith(InputDecorationCopy{LabelText: str("b")})
	if b.LabelText != str("b") || b.HelperText != str("h") {
		t.Errorf("copy = %v", b)
	}
	if !a.Equal(a) || a.Equal(b) {
		t.Error("equality is wrong")
	}
	if s := a.String(); !containsStr(s, "labelText") || !containsStr(s, "helperText") {
		t.Errorf("description = %q", s)
	}
	if NewInputDecorationCollapsed(str("hint")).IsCollapsed != foundation.Some(true) {
		t.Error("a collapsed decoration is not collapsed")
	}
}

func TestInputDecorationThemeDataMergeAndEquality(t *testing.T) {
	a := NewInputDecorationThemeData(InputDecorationThemeDataOpts{HelperMaxLines: foundation.Some(2), Filled: true})
	b := NewInputDecorationThemeData(InputDecorationThemeDataOpts{HelperMaxLines: foundation.Some(5), HintMaxLines: foundation.Some(4),
		Border: NewOutlineInputBorder()})
	m := a.Merge(b)
	if m.HelperMaxLines != foundation.Some(2) || m.HintMaxLines != foundation.Some(4) || m.Border == nil {
		t.Errorf("merge = %+v", m)
	}
	if !a.Equal(a) || a.Equal(b) {
		t.Error("equality is wrong")
	}
	if got := InputDecorationThemeDataLerp(a, b, 0.25); !got.Equal(a) {
		t.Error("lerp before one half is not the first theme")
	}
	if got := InputDecorationThemeDataLerp(a, b, 0.75); !got.Equal(b) {
		t.Error("lerp after one half is not the second theme")
	}
	if s := a.String(); foundation.DebugMode && !containsStr(s, "helperMaxLines") {
		t.Errorf("description = %q", s)
	}
}

func TestInputDecorationThemeWidget(t *testing.T) {
	data := NewInputDecorationThemeData(InputDecorationThemeDataOpts{HelperMaxLines: foundation.Some(7)})
	var got InputDecorationThemeData
	ts := gluttertest.New(t)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewInputDecorationTheme(data, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		got = InputDecorationThemeOf(bc)
		return widgets.NewSizedBox(nil)
	}))}))
	if got.HelperMaxLines != foundation.Some(7) {
		t.Errorf("theme of = %+v", got)
	}
	w := NewInputDecorationTheme(data, widgets.NewSizedBox(nil))
	if w.UpdateShouldNotify(NewInputDecorationTheme(data, widgets.NewSizedBox(nil))) || !w.UpdateShouldNotify(NewInputDecorationTheme(InputDecorationThemeData{}, widgets.NewSizedBox(nil))) {
		t.Error("UpdateShouldNotify is wrong")
	}
}

func TestFloatingLabelEnums(t *testing.T) {
	if FloatingLabelAlignmentStart.String() != "FloatingLabelAlignment.start" || FloatingLabelAlignmentCenter.String() != "FloatingLabelAlignment.center" {
		t.Error("alignment names")
	}
	for _, b := range []FloatingLabelBehavior{FloatingLabelBehaviorNever, FloatingLabelBehaviorAuto, FloatingLabelBehaviorAlways} {
		if b.String() == "" {
			t.Errorf("%d has no name", int(b))
		}
	}
}

func TestInputDecorationInATextFieldUsesTheTheme(t *testing.T) {
	fill := ui.NewColor(0xFF336699)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 200}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
		Theme: NewThemeData(ThemeDataOpts{InputDecorationTheme: NewInputDecorationThemeData(InputDecorationThemeDataOpts{
			Filled: true, FillColor: widgets.NewWidgetStatePropertyAll(fill)})}),
		Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(widgets.NewSizedBox(NewTextField(), widgets.SizedBoxOpts{Width: foundation.Some(300.0)}),
			widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})}),
	}))
	dec := ts.Widget(gluttertest.ByType[*InputDecorator]()).(*InputDecorator).Decoration()
	if dec.Filled != foundation.Some(true) {
		t.Errorf("filled = %v: the theme was not applied", dec.Filled)
	}
}
