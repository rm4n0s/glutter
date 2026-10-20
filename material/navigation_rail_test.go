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

func railDestinations() []NavigationRailDestination {
	icon := func() widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)})
	}
	return []NavigationRailDestination{
		NewNavigationRailDestination(icon(), widgets.NewText("First")),
		NewNavigationRailDestination(icon(), widgets.NewText("Second")),
		NewNavigationRailDestination(icon(), widgets.NewText("Third")),
	}
}

func pumpRail(t *testing.T, rail func() *NavigationRail) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
		Body: widgets.NewRow([]widgets.Widget{rail(), widgets.NewExpanded(bodyBox())})})}))
	return ts
}

func TestNavigationRailSize(t *testing.T) {
	tests := []struct {
		name      string
		opts      NavigationRailOpts
		wantWidth float64
	}{
		{"default", NavigationRailOpts{}, 80},
		{"min width", NavigationRailOpts{MinWidth: foundation.Some(100.0)}, 100},
		{"labels", NavigationRailOpts{LabelType: NavigationRailLabelTypeAll}, 80},
		{"extended", NavigationRailOpts{Extended: true}, 256},
		{"extended with a min width", NavigationRailOpts{Extended: true, MinExtendedWidth: foundation.Some(300.0)}, 300},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := pumpRail(t, func() *NavigationRail {
				return NewNavigationRail(railDestinations(), foundation.Some(0), tc.opts)
			})
			if got := ts.Size(gluttertest.ByType[*NavigationRail]()).Width; got != tc.wantWidth {
				t.Errorf("width = %v, want %v", got, tc.wantWidth)
			}
		})
	}
}

func TestNavigationRailSelectionAndExtension(t *testing.T) {
	selected := foundation.Some(0)
	extended := false
	var tapped []int
	build := func() *NavigationRail {
		return NewNavigationRail(railDestinations(), selected, NavigationRailOpts{
			Extended: extended, OnDestinationSelected: func(i int) { tapped = append(tapped, i) }})
	}
	ts := pumpRail(t, build)
	rail := gluttertest.ByType[*NavigationRail]()
	if got := ts.Size(rail).Width; got != 80 {
		t.Errorf("narrow width = %v, want 80", got)
	}
	// A destination is 44 high (6 + 32 + 6 around the indicator), below the 8 of
	// spacing at the top.
	ts.TapAt(ui.Offset{Dx: 40, Dy: 8 + 44 + 22})
	if len(tapped) != 1 || tapped[0] != 1 {
		t.Errorf("tapped = %v, want [1]", tapped)
	}
	extended = true
	selected = foundation.Some(1)
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
		Body: widgets.NewRow([]widgets.Widget{build(), widgets.NewExpanded(bodyBox())})})}))
	ts.PumpAndSettle()
	if got := ts.Size(rail).Width; got != 256 {
		t.Errorf("extended width = %v, want 256", got)
	}
}

func TestNavigationRailErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func()
		wantRoute string
	}{
		{"selected index", func() { NewNavigationRail(railDestinations(), foundation.Some(3)) }, "NavigationRailSelectedIndex"},
		{"elevation", func() {
			NewNavigationRail(railDestinations(), foundation.Some(0), NavigationRailOpts{Elevation: foundation.Some(0.0)})
		}, "NavigationRailElevation"},
		{"min width", func() {
			NewNavigationRail(railDestinations(), foundation.Some(0), NavigationRailOpts{MinWidth: foundation.Some(0.0)})
		}, "NavigationRailMinWidth"},
		{"min extended width", func() {
			NewNavigationRail(railDestinations(), foundation.Some(0), NavigationRailOpts{MinExtendedWidth: foundation.Some(-1.0)})
		}, "NavigationRailMinExtendedWidth"},
		{"widths", func() {
			NewNavigationRail(railDestinations(), foundation.Some(0), NavigationRailOpts{MinWidth: foundation.Some(100.0), MinExtendedWidth: foundation.Some(50.0)})
		}, "NavigationRailMinWidths"},
		{"extended labels", func() {
			NewNavigationRail(railDestinations(), foundation.Some(0), NavigationRailOpts{Extended: true, LabelType: NavigationRailLabelTypeAll})
		}, "NavigationRailExtendedLabels"},
		{"extended animation", func() {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			ts.PumpWidget(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				NavigationRailExtendedAnimation(bc)
				return widgets.NewSizedBox(nil)
			}))
			if err := ts.TakeError(); err != nil {
				panic(err)
			}
		}, "NavigationRailExtendedAnimationMissing"},
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
