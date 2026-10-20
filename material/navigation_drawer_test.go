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
	"github.com/rm4n0s/glutter/widgets"
)

func TestNavigationDrawer(t *testing.T) {
	var selected []int
	icon := func() widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(24.0), Height: foundation.Some(24.0)})
	}
	ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
		return NewScaffold(ScaffoldOpts{Body: probe, Drawer: NewNavigationDrawer([]widgets.Widget{
			widgets.NewText("Header"),
			NewNavigationDrawerDestination(icon(), widgets.NewText("Inbox")),
			NewNavigationDrawerDestination(icon(), widgets.NewText("Outbox")),
			NewNavigationDrawerDestination(icon(), widgets.NewText("Trash"), NavigationDrawerDestinationOpts{NoEnabled: true}),
		}, NavigationDrawerOpts{OnDestinationSelected: func(i int) { selected = append(selected, i) }})})
	})
	state().OpenDrawer()
	ts.PumpAndSettle()
	if !state().IsDrawerOpen() {
		t.Fatal("the drawer did not open")
	}
	// Destinations are 56 high, below the header text.
	if got := ts.Size(gluttertest.ByType[*Drawer]()).Width; got != 304 {
		t.Errorf("drawer width = %v, want 304", got)
	}
	ts.Tap(gluttertest.ByText("Outbox"))
	ts.Tap(gluttertest.ByText("Trash"))
	if len(selected) != 1 || selected[0] != 1 {
		t.Errorf("selected = %v, want [1] (the disabled destination is not selectable)", selected)
	}
}

func TestNavigationDrawerSelectedIndex(t *testing.T) {
	tests := []struct {
		name string
		opts NavigationDrawerOpts
		want foundation.Opt[int]
	}{
		{"default", NavigationDrawerOpts{}, foundation.Some(0)},
		{"given", NavigationDrawerOpts{SelectedIndex: foundation.Some(2)}, foundation.Some(2)},
		{"none", NavigationDrawerOpts{NoSelectedIndex: true}, foundation.None[int]()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewNavigationDrawer(nil, tc.opts).SelectedIndex(); got != tc.want {
				t.Errorf("SelectedIndex = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNavigationDrawerDestinationWithoutDrawer(t *testing.T) {
	skipUnlessDebug(t)
	err := recoverError(func() {
		ts, _ := scaffoldProbe(t, func(widgets.Widget) *Scaffold {
			return NewScaffold(ScaffoldOpts{Body: NewNavigationDrawerDestination(widgets.NewSizedBox(nil), widgets.NewText("x"))})
		})
		if err := ts.TakeError(); err != nil {
			panic(err)
		}
	})
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute("NavigationDrawerDestinationInfoMissing") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}
