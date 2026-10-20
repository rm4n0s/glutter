// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestAppBarThemeData(t *testing.T) {
	t.Run("default values are all unset", func(t *testing.T) {
		d := NewAppBarThemeData()
		if d.BackgroundColor.IsSome() || d.Elevation.IsSome() || d.CenterTitle.IsSome() || d.ToolbarHeight.IsSome() || d.Shape != nil || d.IconTheme.IsSome() {
			t.Errorf("a field is set: %v", d)
		}
	})
	t.Run("equal and copy with", func(t *testing.T) {
		a := NewAppBarThemeData(AppBarThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFF112233)), Elevation: foundation.Some(2.0),
			SystemOverlayStyle: foundation.Some(services.SystemUiOverlayStyleDark)})
		b := a.CopyWith(AppBarThemeDataCopy{})
		if !a.Equal(b) {
			t.Error("a copy with nothing changed is not equal")
		}
		c := a.CopyWith(AppBarThemeDataCopy{Elevation: foundation.Some(4.0), CenterTitle: foundation.Some(true)})
		if a.Equal(c) || c.Elevation.Value() != 4 || !c.CenterTitle.Value() || c.BackgroundColor != a.BackgroundColor {
			t.Errorf("copy = %v", c)
		}
	})
	t.Run("lerp", func(t *testing.T) {
		tests := []struct {
			name      string
			a, b      AppBarThemeData
			t         float64
			wantElev  foundation.Opt[float64]
			wantColor foundation.Opt[ui.Color]
		}{
			{"numbers interpolate", NewAppBarThemeData(AppBarThemeDataOpts{Elevation: foundation.Some(0.0)}),
				NewAppBarThemeData(AppBarThemeDataOpts{Elevation: foundation.Some(10.0)}), 0.25, foundation.Some(2.5), foundation.None[ui.Color]()},
			{"colors interpolate", NewAppBarThemeData(AppBarThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFF000000))}),
				NewAppBarThemeData(AppBarThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF))}), 0.0, foundation.None[float64](),
				foundation.Some(ui.NewColor(0xFF000000))},
			{"both unset stay unset", NewAppBarThemeData(), NewAppBarThemeData(), 0.5, foundation.None[float64](), foundation.None[ui.Color]()},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				got := AppBarThemeDataLerp(tc.a, tc.b, tc.t)
				if got.Elevation != tc.wantElev || got.BackgroundColor != tc.wantColor {
					t.Errorf("lerp = %v", got)
				}
			})
		}
	})
}

// TestAppBarThemeIsUsed checks that an AppBarTheme above an AppBar gives it its
// values and that the properties of the widget win.
func TestAppBarThemeIsUsed(t *testing.T) {
	tests := []struct {
		name       string
		theme      AppBarThemeData
		o          AppBarOpts
		wantHeight float64
		wantColor  ui.Color
	}{
		{"theme only", NewAppBarThemeData(AppBarThemeDataOpts{ToolbarHeight: foundation.Some(80.0), BackgroundColor: foundation.Some(ui.NewColor(0xFFFF0000))}),
			AppBarOpts{}, 80, ui.NewColor(0xFFFF0000)},
		{"widget wins", NewAppBarThemeData(AppBarThemeDataOpts{ToolbarHeight: foundation.Some(80.0), BackgroundColor: foundation.Some(ui.NewColor(0xFFFF0000))}),
			AppBarOpts{ToolbarHeight: foundation.Some(40.0), BackgroundColor: foundation.Some(ui.NewColor(0xFF00FF00))}, 40, ui.NewColor(0xFF00FF00)},
		{"defaults", NewAppBarThemeData(), AppBarOpts{}, 56, NewThemeData().ColorScheme.Surface()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(), widgets.NewMediaQuery(widgets.NewMediaQueryData(),
				NewAppBarTheme(tc.theme, NewScaffold(ScaffoldOpts{AppBar: NewAppBar(tc.o)}))))))
			if got := ts.Size(gluttertest.ByType[*AppBar]()).Height; got != tc.wantHeight {
				t.Errorf("height = %v, want %v", got, tc.wantHeight)
			}
			if got := pixel(ts, 200, 1); got.R != uint8(tc.wantColor.Red()) || got.G != uint8(tc.wantColor.Green()) || got.B != uint8(tc.wantColor.Blue()) {
				t.Errorf("color = %v, want %v", got, tc.wantColor)
			}
		})
	}
}

func TestAppBarThemeFromThemeData(t *testing.T) {
	theme := NewThemeData(ThemeDataOpts{AppBarTheme: NewAppBarThemeData(AppBarThemeDataOpts{CenterTitle: foundation.Some(true)})})
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	var got AppBarThemeData
	ts.PumpWidget(NewTheme(theme, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		got = AppBarThemeOf(bc)
		return widgets.NewSizedBox(nil)
	})))
	if !got.CenterTitle.Value() {
		t.Errorf("AppBarThemeOf = %v, want the one of ThemeData", got)
	}
}

func TestBottomAppBarThemeData(t *testing.T) {
	t.Run("equal lerp and copy with", func(t *testing.T) {
		a := NewBottomAppBarThemeData(BottomAppBarThemeDataOpts{Color: foundation.Some(ui.NewColor(0xFF000000)), Elevation: foundation.Some(2.0), Height: foundation.Some(60.0)})
		b := NewBottomAppBarThemeData(BottomAppBarThemeDataOpts{Color: foundation.Some(ui.NewColor(0xFFFFFFFF)), Elevation: foundation.Some(6.0), Height: foundation.Some(100.0)})
		if !a.Equal(a.CopyWith(BottomAppBarThemeDataCopy{})) || a.Equal(b) {
			t.Error("equality")
		}
		mid := BottomAppBarThemeDataLerp(a, b, 0.5)
		if mid.Elevation.Value() != 4 || mid.Height.Value() != 80 {
			t.Errorf("lerp = %v", mid)
		}
		if got := BottomAppBarThemeDataLerp(a, b, 1); !got.Equal(b) {
			t.Errorf("lerp at 1 = %v", got)
		}
	})
	t.Run("shape switches at the middle", func(t *testing.T) {
		shape := painting.CircularNotchedRectangle{}
		a := NewBottomAppBarThemeData(BottomAppBarThemeDataOpts{Shape: shape})
		b := NewBottomAppBarThemeData()
		if got := BottomAppBarThemeDataLerp(a, b, 0.4); got.Shape == nil {
			t.Error("shape lost before the middle")
		}
		if got := BottomAppBarThemeDataLerp(a, b, 0.6); got.Shape != nil {
			t.Error("shape kept after the middle")
		}
	})
	t.Run("negative elevation", func(t *testing.T) {
		skipUnlessDebug(t)
		appErr, ok := errors.FromError(recoverError(func() {
			NewBottomAppBarThemeData(BottomAppBarThemeDataOpts{Elevation: foundation.Some(-1.0)})
		}))
		if !ok || !appErr.HasRoute("BottomAppBarThemeElevationNegative") {
			t.Errorf("error = %v", appErr)
		}
	})
}

func TestMaterialBannerThemeData(t *testing.T) {
	a := NewMaterialBannerThemeData(MaterialBannerThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFF000000)), Elevation: foundation.Some(0.0),
		Padding: painting.NewEdgeInsetsAll(0)})
	b := NewMaterialBannerThemeData(MaterialBannerThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), Elevation: foundation.Some(8.0),
		Padding: painting.NewEdgeInsetsAll(20)})
	mid := MaterialBannerThemeDataLerp(a, b, 0.5)
	if mid.Elevation.Value() != 4 || mid.Padding.(painting.EdgeInsets).Left != 10 {
		t.Errorf("lerp = %v", mid)
	}
	if !a.Equal(MaterialBannerThemeDataLerp(a, b, 0)) || !b.Equal(MaterialBannerThemeDataLerp(a, b, 1)) {
		t.Error("lerp at the ends is not the data")
	}
	if got := a.CopyWith(MaterialBannerThemeDataCopy{Elevation: foundation.Some(1.0)}); got.Elevation.Value() != 1 || got.BackgroundColor != a.BackgroundColor {
		t.Errorf("copy = %v", got)
	}
}

func TestSnackBarThemeData(t *testing.T) {
	t.Run("lerp and equality", func(t *testing.T) {
		a := NewSnackBarThemeData(SnackBarThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFF000000)), Elevation: foundation.Some(0.0)})
		b := NewSnackBarThemeData(SnackBarThemeDataOpts{BackgroundColor: foundation.Some(ui.NewColor(0xFFFFFFFF)), Elevation: foundation.Some(10.0)})
		mid := SnackBarThemeDataLerp(a, b, 0.5)
		if mid.Elevation.Value() != 5 {
			t.Errorf("lerp = %v", mid)
		}
		if !a.Equal(a.CopyWith(SnackBarThemeDataCopy{})) || a.Equal(b) {
			t.Error("equality")
		}
	})
	t.Run("behavior switches at the middle", func(t *testing.T) {
		a := NewSnackBarThemeData(SnackBarThemeDataOpts{Behavior: SnackBarBehaviorFloating})
		b := NewSnackBarThemeData(SnackBarThemeDataOpts{Behavior: SnackBarBehaviorFixed})
		if SnackBarThemeDataLerp(a, b, 0.4).Behavior != SnackBarBehaviorFloating || SnackBarThemeDataLerp(a, b, 0.6).Behavior != SnackBarBehaviorFixed {
			t.Error("behavior does not switch at 0.5")
		}
	})
	t.Run("errors", func(t *testing.T) {
		skipUnlessDebug(t)
		tests := []struct {
			name string
			o    SnackBarThemeDataOpts
			want string
		}{
			{"negative elevation", SnackBarThemeDataOpts{Elevation: foundation.Some(-1.0)}, "SnackBarThemeElevationNegative"},
			{"width without floating", SnackBarThemeDataOpts{Width: foundation.Some(300.0)}, "SnackBarThemeWidthNotFloating"},
			{"overflow threshold", SnackBarThemeDataOpts{ActionOverflowThreshold: foundation.Some(1.5)}, "SnackBarThemeOverflowThreshold"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				appErr, ok := errors.FromError(recoverError(func() { NewSnackBarThemeData(tc.o) }))
				if !ok || !appErr.HasRoute(tc.want) {
					t.Errorf("error = %v, want the route %s", appErr, tc.want)
				}
			})
		}
	})
	t.Run("the theme gives the defaults of a snack bar", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		key := widgets.NewGlobalKey[*ScaffoldMessengerState]()
		theme := NewThemeData(ThemeDataOpts{SnackBarTheme: NewSnackBarThemeData(SnackBarThemeDataOpts{
			BackgroundColor: foundation.Some(ui.NewColor(0xFF336699)), Behavior: SnackBarBehaviorFloating, Width: foundation.Some(200.0)})})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: theme, ScaffoldMessengerKey: key, Home: plainScaffold()}))
		m, _ := key.CurrentState()
		m.ShowSnackBar(NewSnackBar(widgets.NewText("hi"), SnackBarOpts{Persist: foundation.Some(true)}))
		ts.PumpAndSettle()
		// The snack bar is laid out loosely and aligned; its Material has the width.
		material := gluttertest.Descendant(gluttertest.ByType[*SnackBar](), gluttertest.ByType[*Material]())
		if got := ts.Size(gluttertest.First(material)).Width; got != 200 {
			t.Errorf("width = %v, want the 200 of the theme", got)
		}
	})
}
