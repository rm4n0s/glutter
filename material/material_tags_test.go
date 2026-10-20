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

// takeAndPanic panics with the first error the framework reported, so that a
// test can follow its route.
func takeAndPanic(ts *gluttertest.Tester) {
	if errs := ts.TakeErrors(); len(errs) > 0 {
		panic(errs[0])
	}
}

// TestDrawerSheetAndTabErrors checks the route of the failures of the drawer, the
// bottom sheet and the tabs.
func TestDrawerSheetAndTabErrors(t *testing.T) {
	skipUnlessDebug(t)
	pump := func(t *testing.T, w widgets.Widget) *gluttertest.Tester {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.PumpWidget(w)
		return ts
	}
	tests := []struct {
		name      string
		run       func(t *testing.T)
		wantRoute string
	}{
		{"drawer elevation", func(*testing.T) { NewDrawer(DrawerOpts{Elevation: foundation.Some(-1.0)}) }, "DrawerElevationNegative"},
		{"no drawer controller", func(t *testing.T) {
			takeAndPanic(pump(t, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				DrawerControllerOf(bc)
				return widgets.NewSizedBox(nil)
			})))
		}, "DrawerControllerMissing"},
		{"bottom sheet elevation", func(*testing.T) {
			NewBottomSheet(func() {}, func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) },
				BottomSheetOpts{Elevation: foundation.Some(-1.0)})
		}, "BottomSheetElevationNegative"},
		{"bottom sheet without a controller", func(t *testing.T) {
			ts := pump(t, NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewAlign(
				NewBottomSheet(func() {}, func(widgets.BuildContext) widgets.Widget {
					return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(400.0), Height: foundation.Some(200.0)})
				}), widgets.AlignOpts{Alignment: painting.AlignmentBottomCenter})})}))
			ts.FlingFrom(ui.Offset{Dx: 400, Dy: 500}, ui.Offset{Dx: 0, Dy: 100}, 1000)
			takeAndPanic(ts)
		}, "BottomSheetNoAnimationController"},
		{"route controller created twice", func(t *testing.T) {
			ts := pump(t, NewMaterialApp(MaterialAppOpts{Home: NewScaffold()}))
			var route *ModalBottomSheetRoute
			navigator := widgets.NavigatorOf(ts.Element(gluttertest.ByType[*Scaffold]()))
			route = NewModalBottomSheetRoute(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) })
			navigator.Push(route)
			ts.PumpAndSettle()
			route.CreateAnimationController()
		}, "ModalBottomSheetRouteControllerExists"},
		{"tab controller index", func(t *testing.T) {
			ts, controller := pumpTabs(t, 3, func(tabs []widgets.Widget) *TabBar { return NewTabBar(tabs) })
			_ = ts
			controller().SetIndex(5)
		}, "TabControllerIndexRange"},
		{"tab controller offset", func(t *testing.T) {
			_, controller := pumpTabs(t, 3, func(tabs []widgets.Widget) *TabBar { return NewTabBar(tabs) })
			controller().SetOffset(2)
		}, "TabControllerOffset"},
		{"tab bar scroll controller detached", func(*testing.T) {
			NewTabBarScrollController().CreateScrollPosition(nil, nil, nil)
		}, "TabBarScrollControllerDetached"},
		{"indicator padding", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(2, NewScaffold(ScaffoldOpts{
				AppBar: NewAppBar(AppBarOpts{Bottom: NewTabBar(tabsOf(2), TabBarOpts{IndicatorPadding: painting.NewEdgeInsetsAll(500)})})}))}))
			takeAndPanic(ts)
		}, "TabBarIndicatorPaddingTooBig"},
		{"underline indicator without a size", func(*testing.T) {
			NewUnderlineTabIndicator().CreateBoxPainter(nil).Paint(nil, ui.Offset{}, painting.ImageConfiguration{})
		}, "UnderlineTabIndicatorSize"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := recoverError(func() { tc.run(t) })
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tc.wantRoute) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
