// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestScaffoldBottomWidgets(t *testing.T) {
	const w, h = 400.0, 600.0
	nav := func() widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(80.0), Key: foundation.NewValueKey("nav")})
	}
	footer := []widgets.Widget{widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(30.0), Key: foundation.NewValueKey("button")})}
	tests := []struct {
		name         string
		mq           widgets.MediaQueryData
		o            ScaffoldOpts
		wantBody     ui.Rect
		wantNavTop   float64 // NaN when there is no navigation bar
		wantFooterAt float64 // top of the footer container, NaN for none
	}{
		{"navigation bar", widgets.NewMediaQueryData(), ScaffoldOpts{BottomNavigationBar: nav()}, ui.NewRectFromLTWH(0, 0, w, h-80), h - 80, math.NaN()},
		{"extended body", widgets.NewMediaQueryData(), ScaffoldOpts{BottomNavigationBar: nav(), ExtendBody: true}, ui.NewRectFromLTWH(0, 0, w, h), h - 80, math.NaN()},
		{"footer buttons", widgets.NewMediaQueryData(), ScaffoldOpts{PersistentFooterButtons: footer}, ui.NewRectFromLTWH(0, 0, w, h-(30+16+1)), math.NaN(), h - (30 + 16 + 1)},
		{"navigation bar and footer", widgets.NewMediaQueryData(), ScaffoldOpts{BottomNavigationBar: nav(), PersistentFooterButtons: footer},
			ui.NewRectFromLTWH(0, 0, w, h-80-(30+16+1)), h - 80, h - 80 - (30 + 16 + 1)},
		{"keyboard over the navigation bar", widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{ViewInsets: painting.NewEdgeInsetsFromLTRB(0, 0, 0, 300)}),
			ScaffoldOpts{BottomNavigationBar: nav()}, ui.NewRectFromLTWH(0, 0, w, h-300), h - 80, math.NaN()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := tc.o
			o.Body = bodyBox()
			ts := pumpScaffold(t, ui.TextDirectionLTR, tc.mq, NewScaffold(o))
			if got := rectOf(ts, gluttertest.ByType[*widgets.ColoredBox]()); got != tc.wantBody {
				t.Errorf("body = %v, want %v", got, tc.wantBody)
			}
			if !math.IsNaN(tc.wantNavTop) {
				if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("nav"))).Dy; got != tc.wantNavTop {
					t.Errorf("navigation bar top = %v, want %v", got, tc.wantNavTop)
				}
			}
			if !math.IsNaN(tc.wantFooterAt) {
				// The footer is aligned at the end, with 8 around and a 1 divider on top.
				if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("button"))); got.Dy != tc.wantFooterAt+1+8 || got.Dx != w-8-50 {
					t.Errorf("footer button at %v, want (%v, %v)", got, w-8-50, tc.wantFooterAt+1+8)
				}
			}
		})
	}
}

func TestScaffoldExtendBodyBehindAppBar(t *testing.T) {
	tests := []struct {
		name       string
		extend     bool
		wantBodyY  float64
		wantTopPad float64 // the padding that the body gets from its MediaQuery
	}{
		{"not extended", false, 56 + 24, 0},
		{"extended", true, 0, 56 + 24},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pad painting.EdgeInsets
			body := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				pad = widgets.MediaQueryPaddingOf(bc)
				return widgets.NewColoredBox(ui.NewColor(0xFF00FF00), widgets.NewSizedBoxExpand(nil))
			})
			mq := widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{
				Padding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 0), ViewPadding: painting.NewEdgeInsetsFromLTRB(0, 24, 0, 0)})
			ts := pumpScaffold(t, ui.TextDirectionLTR, mq, NewScaffold(ScaffoldOpts{
				AppBar: NewAppBar(), Body: body, ExtendBodyBehindAppBar: tc.extend}))
			if got := ts.TopLeft(gluttertest.ByType[*widgets.ColoredBox]()).Dy; got != tc.wantBodyY {
				t.Errorf("body y = %v, want %v", got, tc.wantBodyY)
			}
			if pad.Top != tc.wantTopPad {
				t.Errorf("padding of the body = %v, want a top of %v", pad, tc.wantTopPad)
			}
		})
	}
}

func TestScaffoldState(t *testing.T) {
	var state *ScaffoldState
	var maybeNone *ScaffoldState
	probe := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		state = ScaffoldOf(bc)
		return widgets.NewSizedBox(nil)
	})
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(), widgets.NewMediaQuery(widgets.NewMediaQueryData(),
		widgets.NewColumn([]widgets.Widget{
			widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				maybeNone = ScaffoldMaybeOf(bc)
				return widgets.NewSizedBox(nil)
			}),
			widgets.NewSizedBox(NewScaffold(ScaffoldOpts{AppBar: NewAppBar(), Body: probe, FloatingActionButton: NewFloatingActionButton(func() {}, nil)}),
				widgets.SizedBoxOpts{Height: foundation.Some(300.0)}),
		})))))
	if maybeNone != nil {
		t.Error("ScaffoldMaybeOf found a scaffold above a sibling")
	}
	if !state.HasAppBar() || !state.HasFloatingActionButton() || state.HasDrawer() || state.HasEndDrawer() || state.IsDrawerOpen() || state.IsEndDrawerOpen() {
		t.Error("the state does not describe the scaffold")
	}
	if got := state.AppBarMaxHeight().Value(); got != 56 {
		t.Errorf("max height of the app bar = %v, want 56", got)
	}
}

func TestScaffoldGeometry(t *testing.T) {
	var geometry foundation.ValueListenable[ScaffoldGeometry]
	probe := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		geometry = ScaffoldGeometryOf(bc)
		return widgets.NewSizedBox(nil)
	})
	nav := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(80.0)})
	ts := pumpScaffold(t, ui.TextDirectionLTR, widgets.NewMediaQueryData(), NewScaffold(ScaffoldOpts{
		Body: probe, BottomNavigationBar: nav, FloatingActionButton: NewFloatingActionButton(func() {}, nil)}))
	_ = ts
	g := geometry.Value()
	if got := g.BottomNavigationBarTop.OrElse(-1); got != 520 {
		t.Errorf("top of the bottom navigation bar = %v, want 520", got)
	}
	// The button floats 16 above the navigation bar, at the end.
	want := ui.NewRectFromLTWH(400-16-56, 520-16-56, 56, 56)
	if got := g.FloatingActionButtonArea.Value(); got != want {
		t.Errorf("button area = %v, want %v", got, want)
	}
}

func TestScaffoldErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		run  func(bc widgets.BuildContext)
		want string
	}{
		{"of without a scaffold", func(bc widgets.BuildContext) { ScaffoldOf(bc) }, "ScaffoldOf.ScaffoldMissing"},
		{"geometry without a scaffold", func(bc widgets.BuildContext) { ScaffoldGeometryOf(bc) }, "ScaffoldGeometryOf.ScaffoldGeometryMissing"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			var err error
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				err = recoverError(func() { tc.run(bc) })
				return widgets.NewSizedBox(nil)
			})))
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tc.want) {
				t.Errorf("error = %v, want the route %s", err, tc.want)
			}
		})
	}
}

func TestFloatingActionButtonLocationEndFloat(t *testing.T) {
	g := ScaffoldPrelayoutGeometry{
		FloatingActionButtonSize: ui.Size{Width: 56, Height: 56}, ScaffoldSize: ui.Size{Width: 400, Height: 600}, ContentBottom: 600,
		TextDirection: ui.TextDirectionLTR,
	}
	tests := []struct {
		name string
		edit func(g *ScaffoldPrelayoutGeometry)
		want ui.Offset
	}{
		{"plain", func(*ScaffoldPrelayoutGeometry) {}, ui.Offset{Dx: 328, Dy: 528}},
		{"right to left", func(g *ScaffoldPrelayoutGeometry) { g.TextDirection = ui.TextDirectionRTL }, ui.Offset{Dx: 16, Dy: 528}},
		{"min insets", func(g *ScaffoldPrelayoutGeometry) { g.MinInsets = painting.NewEdgeInsetsFromLTRB(10, 0, 20, 0) }, ui.Offset{Dx: 308, Dy: 528}},
		{"view padding at the bottom", func(g *ScaffoldPrelayoutGeometry) { g.MinViewPadding = painting.NewEdgeInsetsFromLTRB(0, 0, 0, 34) }, ui.Offset{Dx: 328, Dy: 494}},
		{"snack bar", func(g *ScaffoldPrelayoutGeometry) { g.SnackBarSize = ui.Size{Width: 400, Height: 48} }, ui.Offset{Dx: 328, Dy: 600 - 48 - 56 - 16}},
		{"bottom sheet", func(g *ScaffoldPrelayoutGeometry) { g.BottomSheetSize = ui.Size{Width: 400, Height: 200} }, ui.Offset{Dx: 328, Dy: 600 - 200 - 28}},
		{"content above the bottom widgets", func(g *ScaffoldPrelayoutGeometry) { g.ContentBottom = 500 }, ui.Offset{Dx: 328, Dy: 500 - 56 - 16}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			geometry := g
			tc.edit(&geometry)
			if got := FloatingActionButtonLocationEndFloat.GetOffset(geometry); got != tc.want {
				t.Errorf("offset = %v, want %v", got, tc.want)
			}
		})
	}
}
