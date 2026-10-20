// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/material/refresh_indicator_test.dart.

type refreshApp struct {
	refreshes int
	pending   chan struct{}
	statuses  []RefreshIndicatorStatus
	opts      RefreshIndicatorOpts
	noSpinner bool
	itemCount int
}

func (a *refreshApp) onRefresh() <-chan struct{} {
	a.refreshes++
	return a.pending
}

func (a *refreshApp) widget() widgets.Widget {
	count := a.itemCount
	if count == 0 {
		count = 20
	}
	items := make([]widgets.Widget, count)
	for i := range items {
		items[i] = widgets.NewSizedBox(widgets.NewText(itoaTest(i)), widgets.SizedBoxOpts{Height: foundation.Some(200.0)})
	}
	list := widgets.NewListView(items, widgets.ListViewOpts{Physics: widgets.NewAlwaysScrollableScrollPhysics(nil)})
	if a.noSpinner {
		return NewRefreshIndicatorNoSpinner(a.onRefresh, list, RefreshIndicatorNoSpinnerOpts{OnStatusChange: func(s RefreshIndicatorStatus) { a.statuses = append(a.statuses, s) }})
	}
	return NewRefreshIndicator(a.onRefresh, list, a.opts)
}

func itoaTest(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoaTest(i/10) + string(rune('0'+i%10))
}

func pumpRefresh(t *testing.T, a *refreshApp) *gluttertest.Tester {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Theme: standardTheme(true), Home: a.widget()}))
	return ts
}

func TestRefreshIndicatorRefreshes(t *testing.T) {
	a := &refreshApp{pending: make(chan struct{})}
	ts := pumpRefresh(t, a)
	ts.Drag(gluttertest.ByText("0"), ui.Offset{Dx: 0, Dy: 300})
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	if a.refreshes != 1 {
		t.Fatalf("onRefresh was called %d times, want 1", a.refreshes)
	}
	if !ts.Exists(gluttertest.ByType[*RefreshProgressIndicator]()) {
		t.Error("the indicator is not shown while the refresh runs")
	}
	// The indicator goes away when the refresh is done.
	close(a.pending)
	ts.Pump()
	for i := 0; i < 20 && ts.Exists(gluttertest.ByType[*RefreshProgressIndicator]()); i++ {
		time.Sleep(5 * time.Millisecond)
		ts.Pump(100 * time.Millisecond)
	}
	if ts.Exists(gluttertest.ByType[*RefreshProgressIndicator]()) {
		t.Error("the indicator is still shown after the refresh")
	}
}

func TestRefreshIndicatorShortDragDoesNotRefresh(t *testing.T) {
	a := &refreshApp{pending: make(chan struct{})}
	ts := pumpRefresh(t, a)
	ts.Drag(gluttertest.ByText("0"), ui.Offset{Dx: 0, Dy: 50})
	ts.PumpAndSettle()
	if a.refreshes != 0 {
		t.Errorf("onRefresh was called %d times after a short drag", a.refreshes)
	}
	if ts.Exists(gluttertest.ByType[*RefreshProgressIndicator]()) {
		t.Error("the indicator is still shown after the drag was released")
	}
}

func TestRefreshIndicatorOnlyAtTheEdge(t *testing.T) {
	a := &refreshApp{pending: make(chan struct{})}
	ts := pumpRefresh(t, a)
	// Scroll away from the edge, then drag down from the middle: no refresh in
	// the onEdge mode.
	ts.Drag(gluttertest.ByText("1"), ui.Offset{Dx: 0, Dy: -300})
	ts.PumpAndSettle()
	ts.Drag(gluttertest.ByText("2"), ui.Offset{Dx: 0, Dy: 500})
	ts.PumpAndSettle()
	if a.refreshes != 0 {
		t.Errorf("onRefresh was called %d times for a drag that did not start at the edge", a.refreshes)
	}
}

func TestRefreshIndicatorTriggerModeAnywhere(t *testing.T) {
	a := &refreshApp{pending: make(chan struct{}), opts: RefreshIndicatorOpts{TriggerMode: RefreshIndicatorTriggerModeAnywhere}}
	ts := pumpRefresh(t, a)
	ts.Drag(gluttertest.ByText("1"), ui.Offset{Dx: 0, Dy: -300})
	ts.PumpAndSettle()
	ts.Drag(gluttertest.ByText("2"), ui.Offset{Dx: 0, Dy: 800})
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	if a.refreshes != 1 {
		t.Errorf("onRefresh was called %d times in the anywhere mode, want 1", a.refreshes)
	}
}

func TestRefreshIndicatorPosition(t *testing.T) {
	a := &refreshApp{pending: make(chan struct{}), opts: RefreshIndicatorOpts{Displacement: foundation.Some(100.0), EdgeOffset: foundation.Some(20.0)}}
	ts := pumpRefresh(t, a)
	ts.Drag(gluttertest.ByText("0"), ui.Offset{Dx: 0, Dy: 300})
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	top := ts.TopLeft(gluttertest.ByType[*RefreshProgressIndicator]()).Dy
	// The indicator is 20 below the top edge (the edge offset) and 100 further
	// down (the displacement), less what the size transition has not grown yet.
	if top > 20+100+4 || top < 20 {
		t.Errorf("the indicator is at %v", top)
	}
	if got := ts.Center(gluttertest.ByType[*RefreshProgressIndicator]()).Dx; got != 400 {
		t.Errorf("the indicator is centered at %v, want 400", got)
	}
}

func TestRefreshIndicatorStatusChanges(t *testing.T) {
	a := &refreshApp{pending: make(chan struct{}), noSpinner: true}
	ts := pumpRefresh(t, a)
	ts.Drag(gluttertest.ByText("0"), ui.Offset{Dx: 0, Dy: 300})
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	if ts.Exists(gluttertest.ByType[*RefreshProgressIndicator]()) {
		t.Error("a noSpinner indicator shows a spinner")
	}
	want := []RefreshIndicatorStatus{RefreshIndicatorStatusDrag, RefreshIndicatorStatusArmed, RefreshIndicatorStatusSnap}
	if len(a.statuses) < len(want) {
		t.Fatalf("statuses %v", a.statuses)
	}
	for i, s := range want {
		if a.statuses[i] != s {
			t.Errorf("status %d is %v, want %v", i, a.statuses[i], s)
		}
	}
}

func TestRefreshIndicatorShowFromCode(t *testing.T) {
	a := &refreshApp{pending: make(chan struct{})}
	ts := pumpRefresh(t, a)
	state := gluttertest.StateOf[*RefreshIndicatorState](ts, gluttertest.ByType[*RefreshIndicator]())
	done := state.Show()
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	if a.refreshes != 1 {
		t.Fatalf("onRefresh was called %d times, want 1", a.refreshes)
	}
	select {
	case <-done:
		t.Error("the channel of Show is closed before the refresh is done")
	default:
	}
	close(a.pending)
	for i := 0; i < 20; i++ {
		time.Sleep(5 * time.Millisecond)
		ts.Pump(100 * time.Millisecond)
		select {
		case <-done:
			return
		default:
		}
	}
	t.Error("the channel of Show is not closed after the refresh")
}

func TestRefreshIndicatorSemanticsLabel(t *testing.T) {
	skipUnlessDebug(t)
	a := &refreshApp{pending: make(chan struct{})}
	ts := pumpRefresh(t, a)
	handle := ts.EnsureSemantics()
	defer handle.Dispose()
	state := gluttertest.StateOf[*RefreshIndicatorState](ts, gluttertest.ByType[*RefreshIndicator]())
	state.Show()
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	if !ts.Exists(gluttertest.BySemanticsLabel("Refresh")) {
		t.Error("the indicator has no label")
	}
}

func TestRefreshIndicatorErrors(t *testing.T) {
	skipUnlessDebug(t)
	wantRoute(t, recoverError(func() {
		NewRefreshIndicator(func() <-chan struct{} { return nil }, widgets.NewSizedBox(nil), RefreshIndicatorOpts{Elevation: foundation.Some(-1.0)})
	}), "RefreshIndicatorElevationNegative")
}
