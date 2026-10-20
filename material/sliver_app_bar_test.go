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

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// pumpSliverAppBar shows the app bar above a long list and returns the controller
// of the scroll view.
func pumpSliverAppBar(t *testing.T, bar *SliverAppBar, padding painting.EdgeInsets) (*gluttertest.Tester, *widgets.ScrollControllerBase) {
	t.Helper()
	controller := widgets.NewScrollController()
	t.Cleanup(controller.Dispose)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	mq := widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{Padding: padding, ViewPadding: padding})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(NewThemeData(), widgets.NewMediaQuery(mq, NewMaterial(
		widgets.NewCustomScrollView([]widgets.Widget{
			bar,
			widgets.NewSliverFixedExtentListList(50, manyBoxes(100)),
		}, widgets.CustomScrollViewOpts{Controller: controller}))))))
	return ts, controller
}

func manyBoxes(n int) []widgets.Widget {
	out := make([]widgets.Widget, n)
	for i := range out {
		out[i] = widgets.NewSizedBox(nil)
	}
	return out
}

func TestSliverAppBarExtents(t *testing.T) {
	title := func() widgets.Widget { return widgets.NewText("Title") }
	tests := []struct {
		name       string
		bar        func() *SliverAppBar
		padding    painting.EdgeInsets
		scroll     float64
		wantHeight float64 // of the AppBar widget
		wantTop    float64 // of the AppBar, in the view
	}{
		{"small, not scrolled", func() *SliverAppBar { return NewSliverAppBar(SliverAppBarOpts{Title: title()}) }, painting.EdgeInsets{}, 0, 56, 0},
		{"small is half scrolled away", func() *SliverAppBar { return NewSliverAppBar(SliverAppBarOpts{Title: title()}) }, painting.EdgeInsets{}, 28, 56, -28},
		{"small with status bar", func() *SliverAppBar { return NewSliverAppBar(SliverAppBarOpts{Title: title()}) },
			painting.EdgeInsets{Top: 24}, 0, 56 + 24, 0},
		{"pinned and expanded", func() *SliverAppBar {
			return NewSliverAppBar(SliverAppBarOpts{Title: title(), Pinned: foundation.Some(true), ExpandedHeight: foundation.Some(200.0)})
		}, painting.EdgeInsets{}, 0, 200, 0},
		{"pinned half collapsed", func() *SliverAppBar {
			return NewSliverAppBar(SliverAppBarOpts{Title: title(), Pinned: foundation.Some(true), ExpandedHeight: foundation.Some(200.0)})
		}, painting.EdgeInsets{}, 100, 100, 0},
		{"pinned collapsed", func() *SliverAppBar {
			return NewSliverAppBar(SliverAppBarOpts{Title: title(), Pinned: foundation.Some(true), ExpandedHeight: foundation.Some(200.0)})
		}, painting.EdgeInsets{}, 500, 56, 0},
		{"collapsed height", func() *SliverAppBar {
			return NewSliverAppBar(SliverAppBarOpts{Title: title(), Pinned: foundation.Some(true), ExpandedHeight: foundation.Some(200.0), CollapsedHeight: foundation.Some(80.0)})
		}, painting.EdgeInsets{}, 500, 80, 0},
		{"medium expanded", func() *SliverAppBar { return NewSliverAppBarMedium(SliverAppBarOpts{Title: title()}) }, painting.EdgeInsets{}, 0, 112, 0},
		{"medium collapsed", func() *SliverAppBar { return NewSliverAppBarMedium(SliverAppBarOpts{Title: title()}) }, painting.EdgeInsets{}, 500, 64, 0},
		{"large expanded", func() *SliverAppBar { return NewSliverAppBarLarge(SliverAppBarOpts{Title: title()}) }, painting.EdgeInsets{}, 0, 152, 0},
		{"large collapsed", func() *SliverAppBar { return NewSliverAppBarLarge(SliverAppBarOpts{Title: title()}) }, painting.EdgeInsets{}, 500, 64, 0},
		{"large with status bar collapsed", func() *SliverAppBar { return NewSliverAppBarLarge(SliverAppBarOpts{Title: title()}) },
			painting.EdgeInsets{Top: 24}, 500, 64 + 24, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, controller := pumpSliverAppBar(t, tc.bar(), tc.padding)
			controller.JumpTo(tc.scroll)
			ts.Pump()
			bar := gluttertest.ByType[*AppBar]()
			if got := ts.Size(bar).Height; math.Abs(got-tc.wantHeight) > 1e-9 {
				t.Errorf("app bar height = %v, want %v", got, tc.wantHeight)
			}
			if got := ts.TopLeft(bar).Dy; math.Abs(got-tc.wantTop) > 1e-9 {
				t.Errorf("app bar top = %v, want %v", got, tc.wantTop)
			}
		})
	}
}

func TestSliverAppBarFloating(t *testing.T) {
	tests := []struct {
		name     string
		floating bool
		wantTop  float64
	}{
		// Scrolling down 300 and back up a little: a floating bar shows up right away,
		// and the other one is still scrolled away, so it is not even built.
		{"not floating", false, math.NaN()},
		{"floating", true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, controller := pumpSliverAppBar(t, NewSliverAppBar(SliverAppBarOpts{Title: widgets.NewText("T"), Floating: tc.floating}), painting.EdgeInsets{})
			controller.JumpTo(300)
			ts.Pump()
			// The floating header follows the scroll offset of a scroll towards it.
			position := controller.Position().(*widgets.ScrollPositionWithSingleContext)
			position.AnimateTo(270, time.Second, animation.Curves.Linear)
			position.UpdateUserScrollDirection(rendering.ScrollDirectionForward)
			ts.PumpAndSettle()
			if math.IsNaN(tc.wantTop) {
				if ts.Exists(gluttertest.ByType[*AppBar]()) {
					t.Error("the app bar is shown")
				}
				return
			}
			if !ts.Exists(gluttertest.ByType[*AppBar]()) {
				t.Fatal("the floating app bar did not show up")
			}
			if got := ts.TopLeft(gluttertest.ByType[*AppBar]()).Dy; got <= -56 || got > tc.wantTop {
				t.Errorf("app bar top = %v, want it partly visible", got)
			}
		})
	}
}

func TestSliverAppBarElevationWhenScrolledUnder(t *testing.T) {
	ts, controller := pumpSliverAppBar(t, NewSliverAppBar(SliverAppBarOpts{Title: widgets.NewText("T"), Pinned: foundation.Some(true)}), painting.EdgeInsets{})
	material := gluttertest.Descendant(gluttertest.ByType[*AppBar](), gluttertest.ByType[*Material]())
	elevation := func() float64 { return gluttertest.WidgetAs[*Material](ts, material).Elevation() }
	if got := elevation(); got != 0 {
		t.Errorf("elevation at rest = %v, want 0", got)
	}
	controller.JumpTo(10)
	ts.Pump()
	// The app bar is pinned with the list scrolling under it: Material 3 raises it
	// to the scrolled under elevation of 3.
	ts.PumpAndSettle()
	if got := elevation(); got != 3 {
		t.Errorf("elevation scrolled under = %v, want 3", got)
	}
}

func TestSliverAppBarErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name string
		run  func()
		want string
	}{
		{"snap without floating", func() { NewSliverAppBar(SliverAppBarOpts{Snap: true}) }, "SliverAppBarSnapNotFloating"},
		{"stretch trigger offset", func() { NewSliverAppBar(SliverAppBarOpts{StretchTriggerOffset: foundation.Some(0.0)}) }, "SliverAppBarStretchTriggerOffset"},
		{"collapsed height", func() { NewSliverAppBar(SliverAppBarOpts{CollapsedHeight: foundation.Some(10.0)}) }, "SliverAppBarCollapsedHeight"},
		{"medium collapsed height", func() { NewSliverAppBarMedium(SliverAppBarOpts{CollapsedHeight: foundation.Some(10.0)}) }, "SliverAppBarCollapsedHeight"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appErr, ok := errors.FromError(recoverError(tc.run))
			if !ok || !appErr.HasRoute(tc.want) {
				t.Errorf("error = %v, want the route %s", appErr, tc.want)
			}
		})
	}
}
