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

func TestDrawerHeaderHeight(t *testing.T) {
	tests := []struct {
		name      string
		statusBar float64
		margin    painting.EdgeInsetsGeometry
		want      float64
	}{
		{"default margin", 0, nil, 161 + 8},
		{"status bar", 24, nil, 161 + 24 + 8},
		{"no margin", 0, painting.EdgeInsetsZero, 161},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mq := widgets.NewMediaQueryData(widgets.MediaQueryDataOpts{Padding: painting.NewEdgeInsetsFromLTRB(0, tc.statusBar, 0, 0)})
			ts := pumpScaffold(t, ui.TextDirectionLTR, mq, NewScaffold(ScaffoldOpts{
				Body: widgets.NewColumn([]widgets.Widget{NewDrawerHeader(DrawerHeaderOpts{Margin: tc.margin, Child: widgets.NewText("HEADER")})})}))
			if got := ts.Size(gluttertest.ByType[*DrawerHeader]()).Height; got != tc.want {
				t.Errorf("height = %v, want %v", got, tc.want)
			}
			// The child is placed below the status bar, in the padding of the header.
			if got := ts.TopLeft(gluttertest.ByText("HEADER")); got.Dx != 16 || got.Dy != 16+tc.statusBar {
				t.Errorf("the text is at %v, want (16, %v)", got, 16+tc.statusBar)
			}
		})
	}
}

func TestAppBarImpliedActionButtons(t *testing.T) {
	t.Run("drawer buttons open the drawers", func(t *testing.T) {
		ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
			return NewScaffold(ScaffoldOpts{AppBar: NewAppBar(), Body: probe, Drawer: drawerContent("drawer"), EndDrawer: drawerContent("end")})
		})
		ts.Tap(gluttertest.ByType[*DrawerButton]())
		ts.PumpAndSettle()
		if !state().IsDrawerOpen() {
			t.Error("the drawer button did not open the drawer")
		}
		ts.TapAt(ui.Offset{Dx: 380, Dy: 300}) // the scrim
		ts.PumpAndSettle()
		if state().IsDrawerOpen() {
			t.Fatal("the scrim did not close the drawer")
		}
		ts.Tap(gluttertest.ByType[*EndDrawerButton]())
		ts.PumpAndSettle()
		if !state().IsEndDrawerOpen() {
			t.Error("the end drawer button did not open the end drawer")
		}
	})
	t.Run("back button pops", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{AppBar: NewAppBar(), Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			return widgets.NewGestureDetector(widgets.NewText("PUSH"), widgets.GestureDetectorOpts{OnTap: func() {
				widgets.NavigatorPush(bc, NewMaterialPageRoute(func(widgets.BuildContext) widgets.Widget {
					return NewScaffold(ScaffoldOpts{AppBar: NewAppBar(), Body: widgets.NewText("SECOND")})
				}))
			}})
		})})}))
		if ts.Exists(gluttertest.ByType[*BackButton]()) {
			t.Fatal("the first route has a back button")
		}
		ts.Tap(gluttertest.ByText("PUSH"))
		ts.PumpAndSettle()
		if !ts.Exists(gluttertest.ByType[*BackButton]()) {
			t.Fatal("the second route has no back button")
		}
		ts.Tap(gluttertest.ByType[*BackButton]())
		ts.PumpAndSettle()
		if ts.Exists(gluttertest.ByText("SECOND")) {
			t.Error("the back button did not pop the route")
		}
	})
	t.Run("automatic implying off", func(t *testing.T) {
		ts, _ := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
			return NewScaffold(ScaffoldOpts{AppBar: NewAppBar(AppBarOpts{NoAutomaticallyImplyLeading: true}), Body: probe, Drawer: drawerContent("drawer")})
		})
		if ts.Exists(gluttertest.ByType[*DrawerButton]()) {
			t.Error("a drawer button is implied")
		}
	})
	_ = foundation.None[int]
}
