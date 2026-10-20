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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestBottomAppBarLayoutAndColor(t *testing.T) {
	cs := NewThemeData().ColorScheme
	tests := []struct {
		name       string
		o          BottomAppBarOpts
		padding    painting.EdgeInsets
		wantHeight float64
	}{
		{"material 3 defaults", BottomAppBarOpts{}, painting.EdgeInsets{}, 80},
		{"height", BottomAppBarOpts{Height: foundation.Some(50.0)}, painting.EdgeInsets{}, 50},
		{"safe area", BottomAppBarOpts{}, painting.EdgeInsets{Bottom: 34}, 80 + 34},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mq := widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{Padding: tc.padding, ViewPadding: tc.padding})
			ts := pumpScaffold(t, ui.TextDirectionLTR, mq, NewScaffold(ScaffoldOpts{
				Body: bodyBox(), BottomNavigationBar: NewBottomAppBar(tc.o)}))
			bar := gluttertest.ByType[*BottomAppBar]()
			if got := ts.Size(bar).Height; got != tc.wantHeight {
				t.Errorf("height = %v, want %v", got, tc.wantHeight)
			}
			if got := ts.TopLeft(bar).Dy; got != 600-tc.wantHeight {
				t.Errorf("top = %v, want %v", got, 600-tc.wantHeight)
			}
			want := cs.SurfaceContainer()
			// Material 3 has an elevation of 3 with a transparent surface tint: its color.
			got := pixel(ts, 5, 599)
			if got.R != uint8(want.Red()) || got.G != uint8(want.Green()) || got.B != uint8(want.Blue()) {
				t.Errorf("color = %v, want %v", got, want)
			}
		})
	}
}

func TestBottomAppBarNotch(t *testing.T) {
	tests := []struct {
		name     string
		o        BottomAppBarOpts
		wantHole bool
	}{
		// A Material 3 bar has no notch by default.
		{"default shape", BottomAppBarOpts{}, false},
		{"circular notch", BottomAppBarOpts{Shape: painting.CircularNotchedRectangle{}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.o
			o.Height = foundation.Some(60.0)
			// A docked button: the button is centered on the top of the bar.
			ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{
				Body: bodyBox(), BottomNavigationBar: NewBottomAppBar(o), BackgroundColor: foundation.Some(ui.NewColor(0xFFFF0000)),
				FloatingActionButton:         NewFloatingActionButton(func() {}, nil, FloatingActionButtonOpts{}),
				FloatingActionButtonLocation: dockedLocation{},
			}))
			bar := gluttertest.ByType[*BottomAppBar]()
			barTop := ts.TopLeft(bar).Dy
			// The button of 56 sits on the top of the bar, and the notch is 4 wider than it.
			// Just below the button, inside the notch, the bar is cut away and the red
			// background of the scaffold shows through (darkened by the shadow of the
			// button); without a notch the grayish bar is there.
			p := pixel(ts, 200, int(barTop)+30)
			isBackground := int(p.R) > int(p.G)+80
			if isBackground != tc.wantHole {
				t.Errorf("the pixel below the button is %v; the background shows through: %v, want %v", p, isBackground, tc.wantHole)
			}
		})
	}
}

// dockedLocation docks the button at the center of the top of the bottom bar.
type dockedLocation struct{}

func (dockedLocation) GetOffset(g ScaffoldPrelayoutGeometry) ui.Offset {
	return ui.Offset{Dx: (g.ScaffoldSize.Width - g.FloatingActionButtonSize.Width) / 2, Dy: g.ContentBottom - g.FloatingActionButtonSize.Height/2}
}

func TestBottomAppBarNegativeElevation(t *testing.T) {
	skipUnlessDebug(t)
	appErr, ok := errors.FromError(recoverError(func() { NewBottomAppBar(BottomAppBarOpts{Elevation: foundation.Some(-1.0)}) }))
	if !ok || !appErr.HasRoute("BottomAppBarElevationNegative") {
		t.Errorf("error = %v", appErr)
	}
}

func TestFlexibleSpaceBar(t *testing.T) {
	background := widgets.NewSizedBoxExpand(nil, widgets.WidgetOpts{Key: foundation.NewValueKey("background")})
	pump := func(t *testing.T, minExtent, maxExtent, current float64, o FlexibleSpaceBarOpts) *gluttertest.Tester {
		t.Helper()
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(), widgets.NewMediaQuery(widgets.NewMediaQueryData(),
			widgets.NewAlign(widgets.NewSizedBox(FlexibleSpaceBarCreateSettings(current, NewFlexibleSpaceBar(o),
				FlexibleSpaceBarSettingsOpts{MinExtent: foundation.Some(minExtent), MaxExtent: foundation.Some(maxExtent)}),
				widgets.SizedBoxOpts{Width: foundation.Some(400.0), Height: foundation.Some(current)}),
				widgets.AlignOpts{Alignment: painting.AlignmentTopLeft})))))
		return ts
	}
	tests := []struct {
		name       string
		current    float64
		o          FlexibleSpaceBarOpts
		wantBgSize ui.Size
	}{
		// The background has the height of the whole bar, and the width of the
		// bar, when it is expanded and while it collapses.
		{"expanded", 200, FlexibleSpaceBarOpts{Background: background}, ui.Size{Width: 400, Height: 200}},
		{"collapsed halfway, parallax", 100, FlexibleSpaceBarOpts{Background: background}, ui.Size{Width: 400, Height: 200}},
		{"collapsed halfway, pin", 100, FlexibleSpaceBarOpts{Background: background, CollapseMode: CollapseModePin}, ui.Size{Width: 400, Height: 200}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pump(t, 56, 200, tc.current, tc.o)
			bg := gluttertest.ByKey(foundation.NewValueKey("background"))
			if !ts.Exists(bg) {
				t.Fatal("no background")
			}
			if got := ts.Size(bg); got.Width != tc.wantBgSize.Width || got.Height < tc.current {
				t.Errorf("background size = %v, want a width of %v and at least the height %v", got, tc.wantBgSize.Width, tc.current)
			}
		})
	}
	t.Run("title is shown", func(t *testing.T) {
		ts := pump(t, 56, 200, 200, FlexibleSpaceBarOpts{Title: widgets.NewText("Title")})
		if !ts.Exists(gluttertest.ByText("Title")) {
			t.Error("no title")
		}
	})
	t.Run("errors", func(t *testing.T) {
		skipUnlessDebug(t)
		tests := []struct {
			name string
			run  func()
			want string
		}{
			{"scale", func() { NewFlexibleSpaceBar(FlexibleSpaceBarOpts{ExpandedTitleScale: foundation.Some(0.5)}) }, "FlexibleSpaceBarExpandedTitleScale"},
			{"extents", func() { NewFlexibleSpaceBarSettings(1, 100, 50, 60, nil) }, "FlexibleSpaceBarSettingsInvalid"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				appErr, ok := errors.FromError(recoverError(tc.run))
				if !ok || !appErr.HasRoute(tc.want) {
					t.Errorf("error = %v, want the route %s", appErr, tc.want)
				}
			})
		}
		t.Run("no settings", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(), widgets.NewMediaQuery(widgets.NewMediaQueryData(),
				NewFlexibleSpaceBar()))))
			err := ts.TakeError()
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute("FlexibleSpaceBarNoSettings") {
				t.Errorf("error = %v", err)
			}
		})
	})
}
