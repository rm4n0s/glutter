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
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// scaffoldProbe pumps an app whose home is the scaffold that build returns, and
// returns the state of that scaffold.
func scaffoldProbe(t *testing.T, build func(probe widgets.Widget) *Scaffold, theme ...*ThemeData) (*gluttertest.Tester, func() *ScaffoldState) {
	t.Helper()
	var state *ScaffoldState
	probe := widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		state = ScaffoldOf(bc)
		return bodyBox()
	})
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: build(probe), Theme: firstOpts(theme)}))
	return ts, func() *ScaffoldState { return state }
}

func drawerContent(key string) widgets.Widget {
	return NewDrawer(DrawerOpts{Child: widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey(key)})})
}

func TestScaffoldDrawers(t *testing.T) {
	nan := math.NaN()
	var drawerEvents, endEvents []bool
	ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
		return NewScaffold(ScaffoldOpts{
			Body: probe, Drawer: drawerContent("drawer"), EndDrawer: drawerContent("end"),
			OnDrawerChanged:    func(open bool) { drawerEvents = append(drawerEvents, open) },
			OnEndDrawerChanged: func(open bool) { endEvents = append(endEvents, open) },
		})
	})
	drawer := gluttertest.ByKey(foundation.NewValueKey("drawer"))
	if !state().HasDrawer() || !state().HasEndDrawer() || state().IsDrawerOpen() || state().IsEndDrawerOpen() {
		t.Fatal("the drawers are not described by the state")
	}
	if ts.Exists(drawer) {
		t.Error("the closed drawer is built")
	}

	steps := []struct {
		name         string
		do           func()
		wantOpen     bool
		wantEndOpen  bool
		wantDrawerAt float64 // NaN when the drawer is not built
	}{
		{"open", func() { state().OpenDrawer() }, true, false, 0},
		{"close", func() { state().CloseDrawer() }, false, false, nan},
		{"open the end drawer", func() { state().OpenEndDrawer() }, false, true, nan},
		{"open the drawer closes the end drawer", func() { state().OpenDrawer() }, true, false, 0},
		{"tap the scrim", func() { ts.TapAt(ui.Offset{Dx: 390, Dy: 300}) }, false, false, nan},
		{"escape", func() {
			state().OpenDrawer()
			ts.PumpAndSettle()
			ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
		}, false, false, nan},
	}
	for _, st := range steps {
		st.do()
		ts.PumpAndSettle()
		if state().IsDrawerOpen() != st.wantOpen || state().IsEndDrawerOpen() != st.wantEndOpen {
			t.Errorf("%s: open = %v/%v, want %v/%v", st.name, state().IsDrawerOpen(), state().IsEndDrawerOpen(), st.wantOpen, st.wantEndOpen)
		}
		if math.IsNaN(st.wantDrawerAt) {
			if ts.Exists(drawer) {
				t.Errorf("%s: the closed drawer is built", st.name)
			}
		} else if ts.Exists(drawer) && ts.TopLeft(drawer).Dx != st.wantDrawerAt {
			t.Errorf("%s: drawer at x = %v, want %v", st.name, ts.TopLeft(drawer).Dx, st.wantDrawerAt)
		}
	}
	if want := []bool{true, false, true, false, true, false}; !equalBools(drawerEvents, want) {
		t.Errorf("drawer changes = %v, want %v", drawerEvents, want)
	}
	if want := []bool{true, false}; !equalBools(endEvents, want) {
		t.Errorf("end drawer changes = %v, want %v", endEvents, want)
	}
}

func equalBools(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestScaffoldDrawerEdgeDrag(t *testing.T) {
	tests := []struct {
		name     string
		start    ui.Offset
		wantOpen bool
	}{
		{"from the edge", ui.Offset{Dx: 5, Dy: 300}, true},
		{"from the middle", ui.Offset{Dx: 200, Dy: 300}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
				return NewScaffold(ScaffoldOpts{Body: probe, Drawer: drawerContent("drawer")})
			}, NewThemeData(ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformAndroid)}))
			ts.FlingFrom(tc.start, ui.Offset{Dx: 250, Dy: 0}, 1000)
			ts.PumpAndSettle()
			if state().IsDrawerOpen() != tc.wantOpen {
				t.Errorf("open = %v, want %v", state().IsDrawerOpen(), tc.wantOpen)
			}
		})
	}
}

func TestScaffoldPersistentBottomSheet(t *testing.T) {
	sheet := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(100.0), Key: foundation.NewValueKey("sheet")})
	ts, _ := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
		return NewScaffold(ScaffoldOpts{Body: probe, BottomSheet: sheet})
	})
	if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("sheet"))).Dy; got != 500 {
		t.Errorf("sheet top = %v, want 500", got)
	}
	// The sheet is removed with the property.
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: bodyBox()})}))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByKey(foundation.NewValueKey("sheet"))) {
		t.Error("the persistent sheet is still there after Scaffold.BottomSheet became nil")
	}
}

func TestScaffoldShowBottomSheet(t *testing.T) {
	ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
		return NewScaffold(ScaffoldOpts{Body: probe})
	})
	sheet := gluttertest.ByKey(foundation.NewValueKey("sheet"))
	controller := state().ShowBottomSheet(func(widgets.BuildContext) widgets.Widget {
		return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(100.0), Key: foundation.NewValueKey("sheet")})
	})
	ts.PumpAndSettle()
	if got := ts.TopLeft(sheet).Dy; got != 500 {
		t.Errorf("sheet top = %v, want 500", got)
	}
	select {
	case <-controller.Closed():
		t.Fatal("the sheet is closed already")
	default:
	}
	controller.Close()
	ts.PumpAndSettle()
	select {
	case <-controller.Closed():
	default:
		t.Error("Closed did not receive after Close")
	}
	if ts.Exists(sheet) {
		t.Error("the sheet is still in the tree after its animation")
	}
}

func TestScaffoldShowBottomSheetDragAndShow(t *testing.T) {
	ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
		return NewScaffold(ScaffoldOpts{Body: probe})
	})
	build := func(key string) widgets.WidgetBuilder {
		return func(widgets.BuildContext) widgets.Widget {
			return widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(400.0), Height: foundation.Some(100.0), Key: foundation.NewValueKey(key)})
		}
	}
	first := state().ShowBottomSheet(build("a"))
	ts.PumpAndSettle()
	ts.FlingFrom(ui.Offset{Dx: 200, Dy: 550}, ui.Offset{Dx: 0, Dy: 200}, 2000)
	ts.PumpAndSettle()
	select {
	case <-first.Closed():
	default:
		t.Error("dragging the sheet down did not close it")
	}
	// Showing another sheet replaces the current one.
	state().ShowBottomSheet(build("b"))
	second := state().ShowBottomSheet(build("c"))
	ts.PumpAndSettle()
	if ts.Exists(gluttertest.ByKey(foundation.NewValueKey("b"))) || !ts.Exists(gluttertest.ByKey(foundation.NewValueKey("c"))) {
		t.Error("the second sheet did not replace the first")
	}
	second.Close()
	ts.PumpAndSettle()
}

func TestScaffoldBodyScrim(t *testing.T) {
	ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
		return NewScaffold(ScaffoldOpts{Body: probe})
	})
	barriers := gluttertest.Descendant(gluttertest.ByType[*Scaffold](), gluttertest.ByType[*widgets.ModalBarrier]())
	before := ts.Count(barriers)
	state().ShowBodyScrim(true, 0.5)
	ts.Pump()
	if got := ts.Count(barriers); got != before+1 {
		t.Errorf("barriers = %d, want %d", got, before+1)
	}
	state().ShowBodyScrim(false, 0)
	ts.Pump()
	if got := ts.Count(barriers); got != before {
		t.Errorf("barriers = %d, want %d", got, before)
	}
}

func TestScaffoldBottomSheetErrors(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name      string
		run       func(t *testing.T)
		wantRoute string
	}{
		{"show with a persistent sheet", func(t *testing.T) {
			_, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
				return NewScaffold(ScaffoldOpts{Body: probe, BottomSheet: widgets.NewSizedBox(nil)})
			})
			state().ShowBottomSheet(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) })
		}, "ScaffoldBottomSheetShowConflict"},
		{"persistent sheet while another is shown", func(t *testing.T) {
			ts, state := scaffoldProbe(t, func(probe widgets.Widget) *Scaffold {
				return NewScaffold(ScaffoldOpts{Body: probe})
			})
			state().ShowBottomSheet(func(widgets.BuildContext) widgets.Widget { return widgets.NewSizedBox(nil) })
			ts.PumpAndSettle()
			ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: bodyBox(), BottomSheet: widgets.NewSizedBox(nil)})}))
			if err := ts.TakeError(); err != nil {
				panic(err)
			}
		}, "ScaffoldBottomSheetUpdateConflict"},
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

func firstOpts[T any](v []T) T {
	var zero T
	if len(v) == 0 {
		return zero
	}
	return v[0]
}
