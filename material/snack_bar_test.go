// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// snackApp pumps an app whose home is a Scaffold with a floating action button
// and returns the tester and the state of the ScaffoldMessenger of the app.
func snackApp(t *testing.T, scaffold func() *Scaffold) (*gluttertest.Tester, *widgets.GlobalKey[*ScaffoldMessengerState]) {
	t.Helper()
	key := widgets.NewGlobalKey[*ScaffoldMessengerState]()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{ScaffoldMessengerKey: key, Home: scaffold()}))
	return ts, key
}

func plainScaffold() *Scaffold {
	return NewScaffold(ScaffoldOpts{Body: bodyBox()})
}

func fabScaffold() *Scaffold {
	return NewScaffold(ScaffoldOpts{Body: bodyBox(), FloatingActionButton: NewFloatingActionButton(func() {}, widgets.NewText("F"))})
}

func TestSnackBarLifecycle(t *testing.T) {
	ts, key := snackApp(t, plainScaffold)
	messenger, ok := key.CurrentState()
	if !ok {
		t.Fatal("the key of the app does not reach the ScaffoldMessenger")
	}
	controller := messenger.ShowSnackBar(NewSnackBar(widgets.NewText("snack"), SnackBarOpts{Duration: foundation.Some(2 * time.Second)}))
	ts.Pump()
	text := gluttertest.ByText("snack")
	if !ts.Exists(text) {
		t.Fatal("the snack bar is not shown")
	}
	// It slides in: the animation takes 250 ms.
	ts.Pump(125 * time.Millisecond)
	mid := ts.TopLeft(text).Dy
	ts.Pump(125 * time.Millisecond)
	end := ts.TopLeft(text).Dy
	if !(mid > end) {
		t.Errorf("the snack bar does not slide in: y %v then %v", mid, end)
	}
	// A fixed snack bar sits at the bottom of the scaffold, above nothing else.
	if bottom := ts.BottomRight(gluttertest.ByType[*SnackBar]()).Dy; bottom != 600 {
		t.Errorf("the bottom of the snack bar = %v, want 600", bottom)
	}
	// The animation is done in the first frame after its duration, and the
	// messenger starts the timer when it builds after that.
	ts.Pump(time.Millisecond)
	ts.Pump(1999 * time.Millisecond)
	if !ts.Exists(text) {
		t.Error("the snack bar is hidden before its duration is over")
	}
	ts.Pump(time.Millisecond)
	ts.PumpAndSettle()
	if ts.Exists(text) {
		t.Error("the snack bar is still shown after its duration")
	}
	select {
	case reason := <-controller.Closed():
		if reason != SnackBarClosedReasonTimeout {
			t.Errorf("reason = %v, want timeout", reason)
		}
	default:
		t.Error("the snack bar did not complete its Closed channel")
	}
}

func TestSnackBarClosedReasons(t *testing.T) {
	tests := []struct {
		name string
		do   func(ts *gluttertest.Tester, m *ScaffoldMessengerState, c *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason])
		want SnackBarClosedReason
	}{
		{"hide", func(_ *gluttertest.Tester, m *ScaffoldMessengerState, _ *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]) {
			m.HideCurrentSnackBar()
		}, SnackBarClosedReasonHide},
		{"remove", func(_ *gluttertest.Tester, m *ScaffoldMessengerState, _ *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]) {
			m.RemoveCurrentSnackBar()
		}, SnackBarClosedReasonRemove},
		{"close", func(_ *gluttertest.Tester, _ *ScaffoldMessengerState, c *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]) {
			c.Close()
		}, SnackBarClosedReasonHide},
		{"action", func(ts *gluttertest.Tester, _ *ScaffoldMessengerState, _ *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]) {
			ts.Tap(gluttertest.ByText("UNDO"))
		}, SnackBarClosedReasonAction},
		// As in Flutter, clearing hides the current snack bar and drops the others.
		{"clear", func(_ *gluttertest.Tester, m *ScaffoldMessengerState, _ *ScaffoldFeatureController[*SnackBar, SnackBarClosedReason]) {
			m.ClearSnackBars()
		}, SnackBarClosedReasonHide},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, key := snackApp(t, plainScaffold)
			messenger, _ := key.CurrentState()
			pressed := 0
			controller := messenger.ShowSnackBar(NewSnackBar(widgets.NewText("snack"), SnackBarOpts{
				Action: NewSnackBarAction("UNDO", func() { pressed++ }), Persist: foundation.Some(true)}))
			ts.PumpAndSettle()
			tc.do(ts, messenger, controller)
			ts.PumpAndSettle()
			select {
			case got := <-controller.Closed():
				if got != tc.want {
					t.Errorf("reason = %v, want %v", got, tc.want)
				}
			default:
				t.Fatal("Closed did not complete")
			}
			if (tc.name == "action") != (pressed == 1) {
				t.Errorf("action pressed %d times", pressed)
			}
			if ts.Exists(gluttertest.ByText("snack")) {
				t.Error("the snack bar is still shown")
			}
		})
	}
}

func TestSnackBarQueue(t *testing.T) {
	ts, key := snackApp(t, plainScaffold)
	messenger, _ := key.CurrentState()
	messenger.ShowSnackBar(NewSnackBar(widgets.NewText("first"), SnackBarOpts{Duration: foundation.Some(time.Second)}))
	messenger.ShowSnackBar(NewSnackBar(widgets.NewText("second"), SnackBarOpts{Duration: foundation.Some(time.Second)}))
	ts.PumpAndSettle()
	if !ts.Exists(gluttertest.ByText("first")) || ts.Exists(gluttertest.ByText("second")) {
		t.Fatal("only the first snack bar is shown at a time")
	}
	ts.Pump(1500 * time.Millisecond)
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByText("first")) || !ts.Exists(gluttertest.ByText("second")) {
		t.Fatal("the second snack bar replaces the first when its time is over")
	}
}

func TestSnackBarPositionWithFloatingActionButton(t *testing.T) {
	tests := []struct {
		name         string
		behavior     SnackBarBehavior
		wantSnackTop float64 // the bottom of the snack bar
		wantFabTop   float64
	}{
		// The fixed snack bar is at the bottom; the button is above it with a 16 margin.
		{"fixed", SnackBarBehaviorFixed, 600, 0},
		// The floating snack bar is above the button's top, which stays.
		{"floating", SnackBarBehaviorFloating, 600 - 16 - 56 - 0, 600 - 16 - 56},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, key := snackApp(t, fabScaffold)
			messenger, _ := key.CurrentState()
			fab := gluttertest.ByType[*FloatingActionButton]()
			before := ts.TopLeft(fab).Dy
			messenger.ShowSnackBar(NewSnackBar(widgets.NewText("snack"), SnackBarOpts{Behavior: tc.behavior, Persist: foundation.Some(true)}))
			ts.PumpAndSettle()
			snack := gluttertest.ByType[*SnackBar]()
			snackSize := ts.Size(snack)
			snackTop := ts.TopLeft(snack).Dy
			fabY := ts.TopLeft(fab).Dy
			switch tc.behavior {
			case SnackBarBehaviorFixed:
				if bottom := snackTop + snackSize.Height; bottom != 600 {
					t.Errorf("bottom of the snack bar = %v, want 600", bottom)
				}
				if want := snackTop - 56 - 16; fabY != want {
					t.Errorf("button y = %v, want %v (above the snack bar)", fabY, want)
				}
			default:
				if want := before - 0; fabY != want {
					t.Errorf("a floating snack bar moved the button to %v from %v", fabY, before)
				}
				if bottom := snackTop + snackSize.Height; bottom > fabY {
					t.Errorf("the floating snack bar (bottom %v) overlaps the button (top %v)", bottom, fabY)
				}
			}
		})
	}
}

func TestSnackBarErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func(t *testing.T)
		wantRoute string
	}{
		{"negative elevation", func(*testing.T) {
			NewSnackBar(widgets.NewText("x"), SnackBarOpts{Elevation: foundation.Some(-1.0)})
		}, "SnackBarElevationNegative"},
		{"width and margin", func(*testing.T) {
			NewSnackBar(widgets.NewText("x"), SnackBarOpts{Width: foundation.Some(100.0), Margin: painting.NewEdgeInsetsAll(8)})
		}, "SnackBarWidthAndMargin"},
		{"overflow threshold", func(*testing.T) {
			NewSnackBar(widgets.NewText("x"), SnackBarOpts{ActionOverflowThreshold: foundation.Some(2.0)})
		}, "SnackBarOverflowThreshold"},
		{"margin without floating", func(t *testing.T) {
			ts, key := snackApp(t, plainScaffold)
			state, _ := key.CurrentState()
			state.ShowSnackBar(NewSnackBar(widgets.NewText("x"), SnackBarOpts{Margin: painting.NewEdgeInsetsAll(8)}))
			ts.Pump()
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "SnackBarFloatingOnly"},
		{"floating snack bar off screen", func(t *testing.T) {
			key := widgets.NewGlobalKey[*ScaffoldMessengerState]()
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			// The bottom view padding makes a snack bar as high as the screen not fit.
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{ScaffoldMessengerKey: key, Home: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				data := widgets.MediaQueryOf(bc).CopyWith(widgets.MediaQueryDataCopyOpts{ViewPadding: foundation.Some(painting.NewEdgeInsetsFromLTRB(0, 0, 0, 50))})
				return widgets.NewMediaQuery(data, plainScaffold())
			})}))
			state, _ := key.CurrentState()
			state.ShowSnackBar(NewSnackBar(widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(700.0)}),
				SnackBarOpts{Behavior: SnackBarBehaviorFloating}))
			ts.Pump()
			ts.Pump(time.Second)
			if errs := ts.TakeErrors(); len(errs) > 0 {
				panic(errs[0])
			}
		}, "FloatingSnackBarOffScreen"},
		{"no messenger", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			var err error
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				err = recoverError(func() { ScaffoldMessengerOf(bc) })
				return widgets.NewSizedBox(nil)
			})))
			panic(err)
		}, "ScaffoldMessengerMissing"},
		{"no scaffold", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
			var state *ScaffoldMessengerState
			ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, NewScaffoldMessenger(widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
				state = ScaffoldMessengerOf(bc)
				return widgets.NewSizedBox(nil)
			}))))
			state.ShowSnackBar(NewSnackBar(widgets.NewText("x")))
		}, "ShowSnackBarNoScaffold"},
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

func TestSnackBarActionAndCloseIcon(t *testing.T) {
	ts, key := snackApp(t, plainScaffold)
	messenger, _ := key.CurrentState()
	pressed := 0
	messenger.ShowSnackBar(NewSnackBar(widgets.NewText("snack"), SnackBarOpts{
		Action:        NewSnackBarAction("Undo", func() { pressed++ }),
		ShowCloseIcon: foundation.Some(true),
		Duration:      foundation.Some(10 * time.Second),
	}))
	ts.PumpAndSettle()

	// A single line of text has 14 pixels above and below it: the snack bar is 48
	// high, and its content has horizontal margins only around the action.
	if got := ts.Size(gluttertest.ByType[*SnackBar]()).Height; got != 48 {
		t.Errorf("height = %v, want 48", got)
	}
	action := gluttertest.Descendant(gluttertest.ByType[*SnackBarAction](), gluttertest.ByType[*TextButton]())
	if !ts.Exists(action) {
		t.Fatal("the action is not a TextButton")
	}
	closeButton := gluttertest.ByType[*IconButton]()
	if !ts.Exists(closeButton) {
		t.Fatal("the close icon is not an IconButton")
	}
	if got := gluttertest.WidgetAs[*IconButton](ts, closeButton).Tooltip(); got != "Close" {
		t.Errorf("tooltip of the close icon = %q, want Close", got)
	}
	// The action and the close icon are next to each other, 12 pixels
	// horizontally: the right margin of the action and the left one of the icon.
	actionRight := ts.TopLeft(action).Dx + ts.Size(action).Width
	if gap := ts.TopLeft(closeButton).Dx - actionRight; gap <= 0 || gap > 40 {
		t.Errorf("gap between the action and the close icon = %v, want a small positive one", gap)
	}
	ts.Tap(action)
	ts.PumpAndSettle()
	if pressed != 1 {
		t.Errorf("the action was pressed %d times, want 1", pressed)
	}
}
