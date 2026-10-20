// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func tabsOf(n int) []widgets.Widget {
	tabs := make([]widgets.Widget, n)
	for i := range tabs {
		tabs[i] = NewTab(TabOpts{Text: fmt.Sprintf("TAB%d", i)})
	}
	return tabs
}

func pagesOf(n int) []widgets.Widget {
	pages := make([]widgets.Widget, n)
	for i := range pages {
		pages[i] = widgets.NewCenter(widgets.NewText(fmt.Sprintf("PAGE%d", i)))
	}
	return pages
}

// pumpTabs pumps an app with a default tab controller, a tab bar and a view.
func pumpTabs(t *testing.T, n int, bar func(tabs []widgets.Widget) *TabBar) (*gluttertest.Tester, func() *TabController) {
	t.Helper()
	var controller *TabController
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(n,
		widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
			controller = DefaultTabControllerOf(bc)
			return NewScaffold(ScaffoldOpts{
				AppBar: NewAppBar(AppBarOpts{Bottom: bar(tabsOf(n))}),
				Body:   NewTabBarView(pagesOf(n)),
			})
		}))}))
	return ts, func() *TabController { return controller }
}

func TestTabBarSelectsTheTappedTab(t *testing.T) {
	ts, controller := pumpTabs(t, 3, func(tabs []widgets.Widget) *TabBar { return NewTabBar(tabs) })
	if got := controller().Index(); got != 0 {
		t.Fatalf("initial index = %d, want 0", got)
	}
	if !ts.Exists(gluttertest.ByText("PAGE0")) {
		t.Error("the first page is not shown")
	}
	// Three tabs share 800 pixels: the second one is centered at 400.
	ts.Tap(gluttertest.ByText("TAB1"))
	if !controller().IndexIsChanging() {
		t.Error("the index is not changing right after the tap")
	}
	ts.PumpAndSettle()
	if controller().Index() != 1 || controller().PreviousIndex() != 0 || controller().IndexIsChanging() {
		t.Errorf("after the tap: index %d, previous %d, changing %v", controller().Index(), controller().PreviousIndex(), controller().IndexIsChanging())
	}
	if !ts.Exists(gluttertest.ByText("PAGE1")) {
		t.Error("the second page is not shown")
	}
	// A tab that is not adjacent.
	ts.Tap(gluttertest.ByText("TAB2"))
	ts.PumpAndSettle()
	if controller().Index() != 2 || !ts.Exists(gluttertest.ByText("PAGE2")) {
		t.Errorf("index = %d, page 2 shown = %v", controller().Index(), ts.Exists(gluttertest.ByText("PAGE2")))
	}
	ts.Tap(gluttertest.ByText("TAB0"))
	ts.PumpAndSettle()
	if controller().Index() != 0 || !ts.Exists(gluttertest.ByText("PAGE0")) {
		t.Errorf("index = %d, page 0 shown = %v", controller().Index(), ts.Exists(gluttertest.ByText("PAGE0")))
	}
}

func TestTabBarViewDragChangesTheTab(t *testing.T) {
	ts, controller := pumpTabs(t, 3, func(tabs []widgets.Widget) *TabBar { return NewTabBar(tabs) })
	ts.FlingFrom(ui.Offset{Dx: 600, Dy: 300}, ui.Offset{Dx: -500, Dy: 0}, 1500)
	ts.PumpAndSettle()
	if controller().Index() != 1 {
		t.Errorf("index after the drag = %d, want 1", controller().Index())
	}
}

func TestTabBarLayoutAndIndicator(t *testing.T) {
	tests := []struct {
		name       string
		scrollable bool
		wantTab    float64 // x of the second tab's center
	}{
		{"fixed", false, 400},
		{"scrollable", true, -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, _ := pumpTabs(t, 3, func(tabs []widgets.Widget) *TabBar {
				return NewTabBar(tabs, TabBarOpts{IsScrollable: tc.scrollable})
			})
			bar := gluttertest.ByType[*TabBar]()
			if got := ts.Size(bar).Height; got != 48 {
				t.Errorf("height = %v, want 48 (46 for the tab and 2 for the indicator)", got)
			}
			if tc.wantTab > 0 {
				if got := ts.Center(gluttertest.ByText("TAB1")).Dx; math.Abs(got-tc.wantTab) > 0.01 {
					t.Errorf("center of the second tab = %v, want %v", got, tc.wantTab)
				}
			}
			ts.Tap(gluttertest.ByText("TAB1"))
			ts.PumpAndSettle()
		})
	}
}

func TestTabBarEmpty(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(0, NewScaffold(ScaffoldOpts{
		AppBar: NewAppBar(AppBarOpts{Bottom: NewTabBar(nil)})}))}))
	if ts.Exists(gluttertest.ByType[*Tab]()) {
		t.Error("an empty tab bar shows tabs")
	}
}

func TestTabPreferredSize(t *testing.T) {
	icon := widgets.NewSizedBox(nil)
	tests := []struct {
		name string
		tab  *Tab
		want float64
	}{
		{"text", NewTab(TabOpts{Text: "x"}), 46},
		{"icon", NewTab(TabOpts{Icon: icon}), 46},
		{"text and icon", NewTab(TabOpts{Text: "x", Icon: icon}), 72},
		{"height", NewTab(TabOpts{Text: "x", Height: foundation.Some(30.0)}), 30},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tab.PreferredSize().Height; got != tc.want {
				t.Errorf("height = %v, want %v", got, tc.want)
			}
		})
	}
	if got := NewTabBar([]widgets.Widget{NewTab(TabOpts{Text: "x", Icon: icon})}).PreferredSize().Height; got != 74 {
		t.Errorf("bar height = %v, want 74", got)
	}
}

func TestTabController(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	var controller *TabController
	probe := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		controller = DefaultTabControllerOf(bc)
		return widgets.NewSizedBox(nil)
	})
	ts.PumpWidget(NewDefaultTabController(3, probe, DefaultTabControllerOpts{InitialIndex: 1}))
	if controller.Length() != 3 || controller.Index() != 1 || controller.PreviousIndex() != 1 {
		t.Fatalf("controller: length %d, index %d, previous %d", controller.Length(), controller.Index(), controller.PreviousIndex())
	}
	notified := 0
	controller.AddListener(func() { notified++ })
	controller.SetIndex(2)
	if controller.Index() != 2 || controller.PreviousIndex() != 1 || notified != 1 || controller.Animation().Value() != 2 {
		t.Errorf("SetIndex: index %d, previous %d, notified %d, animation %v", controller.Index(), controller.PreviousIndex(), notified, controller.Animation().Value())
	}
	controller.AnimateTo(0)
	if !controller.IndexIsChanging() {
		t.Error("AnimateTo does not change the index with an animation")
	}
	ts.PumpAndSettle()
	if controller.IndexIsChanging() || controller.Index() != 0 {
		t.Errorf("after AnimateTo: index %d, changing %v", controller.Index(), controller.IndexIsChanging())
	}
	controller.SetOffset(0.5)
	if controller.Offset() != 0.5 {
		t.Errorf("offset = %v, want 0.5", controller.Offset())
	}
	// The length shrinks below the selected index: the last tab is selected.
	controller.SetIndex(2)
	ts.PumpWidget(NewDefaultTabController(2, probe, DefaultTabControllerOpts{InitialIndex: 1}))
	if controller.Length() != 2 || controller.Index() != 1 {
		t.Errorf("after shrinking: length %d, index %d, want 2 and 1", controller.Length(), controller.Index())
	}
}

func TestUnderlineTabIndicator(t *testing.T) {
	a := NewUnderlineTabIndicator()
	if got := a.BorderSide().Width; got != 2 {
		t.Errorf("default weight = %v, want 2", got)
	}
	b := NewUnderlineTabIndicator(UnderlineTabIndicatorOpts{BorderSide: foundation.Some(a.BorderSide())})
	if !a.Equal(b) {
		t.Error("equal indicators are not equal")
	}
	clip := a.GetClipPath(ui.NewRectFromLTWH(0, 0, 100, 48), ui.TextDirectionLTR)
	if bounds := clip.GetBounds(); bounds != ui.NewRectFromLTWH(0, 46, 100, 2) {
		t.Errorf("clip bounds = %v, want 0,46 100x2", bounds)
	}
}

func TestTabErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func(t *testing.T)
		wantRoute string
	}{
		{"empty tab", func(*testing.T) { NewTab() }, "TabNoContent"},
		{"text and child", func(t *testing.T) { NewTab(TabOpts{Text: "x", Child: widgets.NewSizedBox(nil)}) }, "TabTextAndChild"},
		{"negative length", func(t *testing.T) { NewDefaultTabController(-1, widgets.NewSizedBox(nil)) }, "TabControllerLength"},
		{"initial index", func(t *testing.T) {
			NewDefaultTabController(2, widgets.NewSizedBox(nil), DefaultTabControllerOpts{InitialIndex: 2})
		}, "TabControllerInitialIndex"},
		{"indicator weight", func(t *testing.T) {
			NewTabBar(tabsOf(2), TabBarOpts{IndicatorWeight: foundation.Some(0.0)})
		}, "TabBarIndicatorWeight"},
		{"no default controller", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{AppBar: NewAppBar(AppBarOpts{Bottom: NewTabBar(tabsOf(2))})})}))
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "TabBarNoController"},
		{"no default controller for the view", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewTabBarView(pagesOf(2))})}))
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "TabBarViewNoController"},
		{"default controller missing", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				DefaultTabControllerOf(bc)
				return widgets.NewSizedBox(nil)
			}))
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "DefaultTabControllerMissing"},
		{"tab count", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(3, NewScaffold(ScaffoldOpts{
				AppBar: NewAppBar(AppBarOpts{Bottom: NewTabBar(tabsOf(2))})}))}))
			ts.Pump()
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "TabBarTabsCount"},
		{"view children count", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(3, NewScaffold(ScaffoldOpts{Body: NewTabBarView(pagesOf(2))}))}))
			ts.Pump()
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "TabBarViewChildrenCount"},
		{"scrollable fill", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(2, NewScaffold(ScaffoldOpts{
				AppBar: NewAppBar(AppBarOpts{Bottom: NewTabBar(tabsOf(2), TabBarOpts{IsScrollable: true, TabAlignment: TabAlignmentFill})})}))}))
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "TabBarAlignmentFillScrollable"},
		{"fixed start", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewDefaultTabController(2, NewScaffold(ScaffoldOpts{
				AppBar: NewAppBar(AppBarOpts{Bottom: NewTabBar(tabsOf(2), TabBarOpts{TabAlignment: TabAlignmentStart})})}))}))
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "TabBarAlignmentStartFixed"},
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
