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

// pumpModalSheet pumps an app that opens a modal bottom sheet from its body.
func pumpModalSheet(t *testing.T, o ShowModalBottomSheetOpts) (*gluttertest.Tester, *<-chan any) {
	t.Helper()
	result := new(<-chan any)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewBuilder(func(bc widgets.BuildContext) widgets.Widget {
		return widgets.NewCenter(widgets.NewGestureDetector(widgets.NewText("OPEN"), widgets.GestureDetectorOpts{OnTap: func() {
			*result = ShowModalBottomSheet(bc, func(bc widgets.BuildContext) widgets.Widget {
				return widgets.NewGestureDetector(
					widgets.NewSizedBox(widgets.NewText("SHEET"), widgets.SizedBoxOpts{Width: foundation.Some(800.0), Height: foundation.Some(200.0)}),
					widgets.GestureDetectorOpts{OnTap: func() { widgets.NavigatorPop(bc, "done") }})
			}, o)
		}}))
	})})}))
	return ts, result
}

func TestShowModalBottomSheet(t *testing.T) {
	ts, result := pumpModalSheet(t, ShowModalBottomSheetOpts{})
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	sheet := gluttertest.ByText("SHEET")
	if !ts.Exists(sheet) {
		t.Fatal("the sheet is not shown")
	}
	// The sheet is 200 high, at the bottom, and at most 640 wide, centered (Material 3).
	box := gluttertest.Descendant(gluttertest.ByType[*BottomSheet](), gluttertest.ByType[*Material]())
	if got := ts.Size(box); got != (ui.Size{Width: 640, Height: 200}) {
		t.Errorf("sheet size = %v, want 640x200", got)
	}
	if got := ts.TopLeft(box); got != (ui.Offset{Dx: 80, Dy: 400}) {
		t.Errorf("sheet at %v, want (80, 400)", got)
	}
	// Tapping the content pops the route with a value.
	ts.Tap(sheet)
	ts.PumpAndSettle()
	select {
	case v := <-*result:
		if v != "done" {
			t.Errorf("popped with %v, want done", v)
		}
	default:
		t.Error("the route did not complete")
	}
	if ts.Exists(sheet) {
		t.Error("the sheet is still shown")
	}
}

func TestModalBottomSheetDismissal(t *testing.T) {
	tests := []struct {
		name      string
		opts      ShowModalBottomSheetOpts
		act       func(ts *gluttertest.Tester)
		wantShown bool
	}{
		{"tap the barrier", ShowModalBottomSheetOpts{}, func(ts *gluttertest.Tester) { ts.TapAt(ui.Offset{Dx: 400, Dy: 100}) }, false},
		{"barrier not dismissible", ShowModalBottomSheetOpts{NoDismissible: true}, func(ts *gluttertest.Tester) { ts.TapAt(ui.Offset{Dx: 400, Dy: 100}) }, true},
		{"drag down", ShowModalBottomSheetOpts{}, func(ts *gluttertest.Tester) {
			ts.FlingFrom(ui.Offset{Dx: 400, Dy: 450}, ui.Offset{Dx: 0, Dy: 300}, 2000)
		}, false},
		{"drag disabled", ShowModalBottomSheetOpts{NoEnableDrag: true}, func(ts *gluttertest.Tester) {
			ts.FlingFrom(ui.Offset{Dx: 400, Dy: 450}, ui.Offset{Dx: 0, Dy: 300}, 2000)
		}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, _ := pumpModalSheet(t, tc.opts)
			ts.Tap(gluttertest.ByText("OPEN"))
			ts.PumpAndSettle()
			tc.act(ts)
			ts.PumpAndSettle()
			if got := ts.Exists(gluttertest.ByText("SHEET")); got != tc.wantShown {
				t.Errorf("sheet shown = %v, want %v", got, tc.wantShown)
			}
		})
	}
}

func TestModalBottomSheetSizing(t *testing.T) {
	tests := []struct {
		name       string
		opts       ShowModalBottomSheetOpts
		wantHeight float64
	}{
		{"content", ShowModalBottomSheetOpts{}, 200},
		{"scroll controlled", ShowModalBottomSheetOpts{IsScrollControlled: true}, 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, _ := pumpModalSheet(t, tc.opts)
			ts.Tap(gluttertest.ByText("OPEN"))
			ts.PumpAndSettle()
			if got := ts.Size(gluttertest.Descendant(gluttertest.ByType[*BottomSheet](), gluttertest.ByType[*Material]())).Height; got != tc.wantHeight {
				t.Errorf("height = %v, want %v", got, tc.wantHeight)
			}
		})
	}
}

func TestBottomSheetDragHandle(t *testing.T) {
	ts, _ := pumpModalSheet(t, ShowModalBottomSheetOpts{ShowDragHandle: foundation.Some(true)})
	ts.Tap(gluttertest.ByText("OPEN"))
	ts.PumpAndSettle()
	// The handle takes the 48 of a minimum interactive dimension above the content.
	box := gluttertest.Descendant(gluttertest.ByType[*BottomSheet](), gluttertest.ByType[*Material]())
	if got := ts.Size(box).Height; got != 200+48 {
		t.Errorf("height with a handle = %v, want 248", got)
	}
}
