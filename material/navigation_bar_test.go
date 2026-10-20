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
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func navDestinations() []widgets.Widget {
	return []widgets.Widget{
		NewNavigationDestination(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)}), "One"),
		NewNavigationDestination(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)}), "Two"),
		NewNavigationDestination(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)}), "Three"),
	}
}

func pumpNavigationBar(t *testing.T, bar func() *NavigationBar) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{BottomNavigationBar: bar()})}))
	return ts
}

func TestNavigationBarLayout(t *testing.T) {
	tests := []struct {
		name         string
		behavior     NavigationDestinationLabelBehavior
		wantLabel    bool
		wantLabelOpa float64
	}{
		{"always show", NavigationDestinationLabelBehaviorAlwaysShow, true, 1},
		{"always hide", NavigationDestinationLabelBehaviorAlwaysHide, true, 0},
		{"only show selected", NavigationDestinationLabelBehaviorOnlyShowSelected, true, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpNavigationBar(t, func() *NavigationBar {
				return NewNavigationBar(navDestinations(), NavigationBarOpts{LabelBehavior: tc.behavior})
			})
			bar := gluttertest.ByType[*NavigationBar]()
			if got := ts.Size(bar); got != (ui.Size{Width: 400, Height: 80}) {
				t.Errorf("bar size = %v, want 400x80", got)
			}
			if got := ts.TopLeft(bar).Dy; got != 520 {
				t.Errorf("bar top = %v, want 520", got)
			}
			if !ts.Exists(gluttertest.ByText("One")) {
				t.Error("the label of the first destination is missing")
			}
		})
	}
}

func TestNavigationBarSelection(t *testing.T) {
	selected := -1
	index := 0
	var ts *gluttertest.Tester
	build := func() *NavigationBar {
		return NewNavigationBar(navDestinations(), NavigationBarOpts{
			SelectedIndex: index, OnDestinationSelected: func(i int) { selected = i }})
	}
	ts = pumpNavigationBar(t, build)
	ts.Tap(gluttertest.ByText("Two"))
	if selected != 1 {
		t.Errorf("selected = %d, want 1", selected)
	}
	ts.Tap(gluttertest.ByText("Three"))
	if selected != 2 {
		t.Errorf("selected = %d, want 2", selected)
	}
	// The indicator animates in on the selected destination.
	index = 1
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{BottomNavigationBar: build()})}))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByType[*NavigationIndicator]()) {
		t.Error("no indicator")
	}
}

func TestNavigationBarErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"one destination", func() { NewNavigationBar(navDestinations()[:1]) }, "NavigationBarDestinations"},
		{"destination without a bar", func() {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: navDestinations()[0]})}))
			if err := ts.TakeError(); err != nil {
				panic(err)
			}
		}, "NavigationDestinationInfoMissing"},
		{"selected index", func() { NewNavigationBar(navDestinations(), NavigationBarOpts{SelectedIndex: 3}) }, "NavigationBarSelectedIndex"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := recoverError(tc.run)
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
