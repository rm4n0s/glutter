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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func TestDrawerAndSheetThemeData(t *testing.T) {
	red, blue := ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)
	t.Run("drawer", func(t *testing.T) {
		a := NewDrawerThemeData(DrawerThemeDataOpts{BackgroundColor: foundation.Some(red), Elevation: foundation.Some(0.0), Width: foundation.Some(100.0)})
		if !a.Equal(a.CopyWith(DrawerThemeDataCopy{})) {
			t.Error("a copy with nothing changed is not equal")
		}
		c := a.CopyWith(DrawerThemeDataCopy{ScrimColor: foundation.Some(blue)})
		if a.Equal(c) || c.ScrimColor.Value() != blue || c.Width.Value() != 100 {
			t.Errorf("copy = %v", c)
		}
		if got := DrawerThemeDataLerp(a, NewDrawerThemeData(DrawerThemeDataOpts{Width: foundation.Some(200.0)}), 0.5).Width.Value(); got != 150 {
			t.Errorf("lerped width = %v, want 150", got)
		}
	})
	t.Run("bottom sheet", func(t *testing.T) {
		a := NewBottomSheetThemeData(BottomSheetThemeDataOpts{BackgroundColor: foundation.Some(red), ModalElevation: foundation.Some(4.0)})
		if !a.Equal(a.CopyWith(BottomSheetThemeDataCopy{})) {
			t.Error("a copy with nothing changed is not equal")
		}
		c := a.CopyWith(BottomSheetThemeDataCopy{ShowDragHandle: foundation.Some(true)})
		if a.Equal(c) || !c.ShowDragHandle.Value() {
			t.Errorf("copy = %v", c)
		}
		if got := BottomSheetThemeDataLerp(a, NewBottomSheetThemeData(BottomSheetThemeDataOpts{ModalElevation: foundation.Some(8.0)}), 0.5).ModalElevation.Value(); got != 6 {
			t.Errorf("lerped elevation = %v, want 6", got)
		}
	})
	t.Run("navigation bar", func(t *testing.T) {
		a := NewNavigationBarThemeData(NavigationBarThemeDataOpts{Height: foundation.Some(60.0), LabelBehavior: NavigationDestinationLabelBehaviorAlwaysHide})
		if !a.Equal(a.CopyWith(NavigationBarThemeDataCopy{})) {
			t.Error("a copy with nothing changed is not equal")
		}
		b := NewNavigationBarThemeData(NavigationBarThemeDataOpts{Height: foundation.Some(100.0), LabelBehavior: NavigationDestinationLabelBehaviorAlwaysShow})
		got := NavigationBarThemeDataLerp(a, b, 0.5)
		if got.Height.Value() != 80 || got.LabelBehavior != NavigationDestinationLabelBehaviorAlwaysShow {
			t.Errorf("lerp = %v", got)
		}
	})
	t.Run("navigation rail", func(t *testing.T) {
		a := NewNavigationRailThemeData(NavigationRailThemeDataOpts{MinWidth: foundation.Some(50.0), LabelType: NavigationRailLabelTypeAll, UseIndicator: foundation.Some(true)})
		if !a.Equal(a.CopyWith(NavigationRailThemeDataCopy{})) {
			t.Error("a copy with nothing changed is not equal")
		}
		got := NavigationRailThemeDataLerp(a, NewNavigationRailThemeData(NavigationRailThemeDataOpts{MinWidth: foundation.Some(100.0)}), 0.5)
		if got.MinWidth.Value() != 75 {
			t.Errorf("lerp = %v", got)
		}
	})
	t.Run("navigation drawer", func(t *testing.T) {
		a := NewNavigationDrawerThemeData(NavigationDrawerThemeDataOpts{TileHeight: foundation.Some(40.0), IndicatorSize: foundation.Some(ui.Size{Width: 100, Height: 40})})
		if !a.Equal(a.CopyWith(NavigationDrawerThemeDataCopy{})) {
			t.Error("a copy with nothing changed is not equal")
		}
		got := NavigationDrawerThemeDataLerp(a, NewNavigationDrawerThemeData(NavigationDrawerThemeDataOpts{TileHeight: foundation.Some(60.0)}), 0.5)
		if got.TileHeight.Value() != 50 {
			t.Errorf("lerp = %v", got)
		}
	})
	t.Run("tab bar", func(t *testing.T) {
		a := NewTabBarThemeData(TabBarThemeDataOpts{DividerHeight: foundation.Some(2.0), TabAlignment: TabAlignmentCenter, IndicatorSize: TabBarIndicatorSizeLabel})
		if !a.Equal(a.CopyWith(TabBarThemeDataCopy{})) {
			t.Error("a copy with nothing changed is not equal")
		}
		got := TabBarThemeDataLerp(a, NewTabBarThemeData(TabBarThemeDataOpts{DividerHeight: foundation.Some(4.0)}), 0.5)
		if got.DividerHeight.Value() != 3 {
			t.Errorf("lerp = %v", got)
		}
	})
}

// TestNavigationThemesAreUsed checks that the themes above the widgets, and the
// theme of the app, give them their values.
func TestNavigationThemesAreUsed(t *testing.T) {
	t.Run("navigation bar height", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
			Theme: NewThemeData(ThemeDataOpts{NavigationBarTheme: NewNavigationBarThemeData(NavigationBarThemeDataOpts{Height: foundation.Some(100.0)})}),
			Home:  NewScaffold(ScaffoldOpts{BottomNavigationBar: NewNavigationBar(navDestinations())})}))
		if got := ts.Size(gluttertest.ByType[*NavigationBar]()).Height; got != 100 {
			t.Errorf("height = %v, want 100", got)
		}
	})
	t.Run("navigation rail width from the inherited theme", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewRow([]widgets.Widget{
			NewNavigationRailTheme(NewNavigationRailThemeData(NavigationRailThemeDataOpts{MinWidth: foundation.Some(120.0)}),
				NewNavigationRail(railDestinations(), foundation.Some(0))),
			widgets.NewExpanded(bodyBox())})})}))
		if got := ts.Size(gluttertest.ByType[*NavigationRail]()).Width; got != 120 {
			t.Errorf("width = %v, want 120", got)
		}
	})
	t.Run("tab bar divider", func(t *testing.T) {
		ts, _ := pumpTabs(t, 2, func(tabs []widgets.Widget) *TabBar { return NewTabBar(tabs) })
		if got := TabBarThemeOf(ts.Element(gluttertest.ByType[*TabBar]())).DividerHeight; got.IsSome() {
			t.Errorf("the default theme has a divider height: %v", got)
		}
	})
}
