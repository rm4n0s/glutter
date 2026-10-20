// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scroll_notification_test.dart. They
// scroll a Scrollable of one sliver instead of a SingleChildScrollView (step
// 9.3).

func tall(height float64) Widget {
	return testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(height)})})
}

func TestScrollMetricsNotification(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var events []any
	frame := func(height float64) Widget {
		return ltr(NewNotificationListener[Notification](tall(height), func(n Notification) bool {
			events = append(events, n)
			return false
		}))
	}
	check := func(height, before, inside, after, total float64) {
		t.Helper()
		if len(events) != 1 {
			t.Fatalf("events = %d, want 1: %v", len(events), events)
		}
		n, ok := events[0].(*ScrollMetricsNotification)
		if !ok {
			t.Fatalf("event is %T, want a ScrollMetricsNotification", events[0])
		}
		m := n.Metrics()
		if m.ExtentBefore() != before || m.ExtentInside() != inside || m.ExtentAfter() != after || m.ExtentTotal() != total {
			t.Errorf("height %v: metrics before %v inside %v after %v total %v", height, m.ExtentBefore(), m.ExtentInside(), m.ExtentAfter(), m.ExtentTotal())
		}
	}
	ts.PumpWidget(frame(1200))
	check(1200, 0, 600, 600, 1200)

	events = nil
	ts.PumpWidget(frame(1000)) // changing the content dimensions triggers a new event
	check(1000, 0, 600, 400, 1000)

	events = nil
	g := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	ts.Pump(time.Second)
	g.MoveBy(ui.Offset{Dx: -10, Dy: -10})
	ts.Pump(time.Second)
	g.Up()
	ts.Pump(time.Second)
	// Deviation from the Dart test: a SingleChildScrollView only repaints when it
	// scrolls, but a Viewport lays out again, so the user scroll changes the
	// metrics the position saw last and triggers a ScrollMetricsNotification.
	want := []reflect.Type{reflect.TypeFor[*ScrollStartNotification](), reflect.TypeFor[*UserScrollNotification](),
		reflect.TypeFor[*ScrollUpdateNotification](), reflect.TypeFor[*ScrollMetricsNotification](),
		reflect.TypeFor[*ScrollEndNotification](), reflect.TypeFor[*UserScrollNotification]()}
	if got := typesOf(events); !reflect.DeepEqual(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}

	events = nil
	ts.PumpWidget(frame(500)) // changing the content dimensions again
	check(500, 0, 600, 0, 600)

	events = nil
	ts.PumpWidget(frame(500)) // the content dimensions do not change
	if len(events) != 0 {
		t.Errorf("events = %v, want none", events)
	}
}

func TestScrollMetricsNotificationIsNotAScrollNotification(t *testing.T) {
	var n any = NewScrollMetricsNotification(nil, nil)
	if _, ok := n.(ScrollNotification); ok {
		t.Error("a ScrollMetricsNotification is a ScrollNotification")
	}
	if _, ok := any(NewScrollUpdateNotification(nil, nil)).(ScrollNotification); !ok {
		t.Error("a ScrollUpdateNotification is not a ScrollNotification")
	}
}

func typesOf(events []any) []reflect.Type {
	var out []reflect.Type
	for _, e := range events {
		out = append(out, reflect.TypeOf(e))
	}
	return out
}

func TestScrollNotificationBasics(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var last ScrollNotification
	ts.PumpWidget(ltr(NewNotificationListener[ScrollNotification](tall(1200), func(n ScrollNotification) bool {
		switch n.(type) {
		case *ScrollStartNotification, *ScrollUpdateNotification, *ScrollEndNotification:
			last = n
		}
		return false
	})))
	g := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	ts.Pump(time.Second)
	start, ok := last.(*ScrollStartNotification)
	if !ok || start.Depth() != 0 || !start.DragDetails.IsSome() || start.DragDetails.Value().GlobalPosition != (ui.Offset{Dx: 100, Dy: 100}) {
		t.Fatalf("start = %v", last)
	}
	g.MoveBy(ui.Offset{Dx: -10, Dy: -10})
	ts.Pump(time.Second)
	update, ok := last.(*ScrollUpdateNotification)
	if !ok || update.Depth() != 0 || !update.DragDetails.IsSome() {
		t.Fatalf("update = %v", last)
	}
	if d := update.DragDetails.Value(); d.GlobalPosition != (ui.Offset{Dx: 90, Dy: 90}) || d.Delta != (ui.Offset{Dy: -10}) {
		t.Errorf("update drag details = %+v", d)
	}
	g.Up()
	ts.Pump(time.Second)
	end, ok := last.(*ScrollEndNotification)
	if !ok || end.Depth() != 0 || !end.DragDetails.IsSome() || end.DragDetails.Value().Velocity != gestures.VelocityZero {
		t.Fatalf("end = %v", last)
	}
}

func nestedScrollables(dragStart gestures.DragStartBehavior, outer, inner func(ScrollNotification) bool) Widget {
	innerScrollable := NewScrollable(func(_ BuildContext, p rendering.ViewportOffset) Widget {
		return NewViewport(p, []Widget{newSliverBox(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(1200.0)}))}, ViewportOpts{})
	}, ScrollableOpts{DragStartBehavior: dragStart})
	var innerWidget Widget = NewContainer(innerScrollable, ContainerOpts{Padding: painting.NewEdgeInsetsAll(50)})
	if inner != nil {
		innerWidget = NewNotificationListener[ScrollNotification](innerWidget, inner)
	}
	outerScrollable := NewScrollable(func(_ BuildContext, p rendering.ViewportOffset) Widget {
		return NewViewport(p, []Widget{newSliverBox(NewSizedBox(innerWidget, SizedBoxOpts{Height: foundation.Some(1200.0)}))}, ViewportOpts{})
	}, ScrollableOpts{DragStartBehavior: dragStart})
	return ltr(NewNotificationListener[ScrollNotification](outerScrollable, outer))
}

func TestScrollNotificationDepth(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var depth0Types, depth1Types []reflect.Type
	var depth0Values, depth1Values []int
	ts.PumpWidget(nestedScrollables(gestures.DragStartBehaviorDown,
		func(n ScrollNotification) bool {
			depth1Types, depth1Values = append(depth1Types, reflect.TypeOf(n)), append(depth1Values, n.Depth())
			return false
		},
		func(n ScrollNotification) bool {
			depth0Types, depth0Values = append(depth0Types, reflect.TypeOf(n)), append(depth0Values, n.Depth())
			return false
		}))
	g := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	ts.Pump(time.Second)
	g.MoveBy(ui.Offset{Dx: -10, Dy: -40})
	ts.Pump(time.Second)
	g.Up()
	ts.Pump(time.Second)

	want := []reflect.Type{reflect.TypeFor[*ScrollStartNotification](), reflect.TypeFor[*UserScrollNotification](),
		reflect.TypeFor[*ScrollUpdateNotification](), reflect.TypeFor[*ScrollEndNotification](), reflect.TypeFor[*UserScrollNotification]()}
	if !reflect.DeepEqual(depth0Types, want) || !reflect.DeepEqual(depth1Types, want) {
		t.Errorf("depth 0 types %v, depth 1 types %v, want %v", depth0Types, depth1Types, want)
	}
	if !reflect.DeepEqual(depth0Values, []int{0, 0, 0, 0, 0}) || !reflect.DeepEqual(depth1Values, []int{1, 1, 1, 1, 1}) {
		t.Errorf("depths %v and %v", depth0Values, depth1Values)
	}
}

func TestScrollNotificationObserverWidget(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var observer *ScrollNotificationObserverState
	var last ScrollNotification
	handle := func(n ScrollNotification) {
		switch n := n.(type) {
		case *ScrollUpdateNotification:
			// The observer also reports a ScrollMetricsNotification as an update
			// without drag details; a Viewport sends one after a scroll, a
			// SingleChildScrollView does not.
			if n.DragDetails.IsSome() {
				last = n
			}
		case *ScrollStartNotification, *ScrollEndNotification:
			last = n
		}
	}
	ts.PumpWidget(ltr(NewScrollNotificationObserver(NewBuilder(func(bc BuildContext) Widget {
		observer = ScrollNotificationObserverOf(bc)
		return tall(1200)
	}))))
	h := observer.AddListener(handle)

	g := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	ts.PumpAndSettle()
	start, ok := last.(*ScrollStartNotification)
	if !ok || start.Depth() != 0 || start.DragDetails.Value().GlobalPosition != (ui.Offset{Dx: 100, Dy: 100}) {
		t.Fatalf("start = %v", last)
	}
	g.MoveBy(ui.Offset{Dx: -10, Dy: -10})
	ts.PumpAndSettle()
	update, ok := last.(*ScrollUpdateNotification)
	if !ok || update.DragDetails.Value().GlobalPosition != (ui.Offset{Dx: 90, Dy: 90}) || update.DragDetails.Value().Delta != (ui.Offset{Dy: -10}) {
		t.Fatalf("update = %v", last)
	}
	g.Up()
	ts.PumpAndSettle()
	if _, ok := last.(*ScrollEndNotification); !ok {
		t.Fatalf("end = %v", last)
	}

	observer.RemoveListener(h)
	last = nil
	g = ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	ts.PumpAndSettle()
	g.MoveBy(ui.Offset{Dx: -10, Dy: -10})
	ts.PumpAndSettle()
	g.Up()
	ts.PumpAndSettle()
	if last != nil {
		t.Errorf("a removed listener was called with %v", last)
	}
}

func TestScrollNotificationObserverIsDisposedWithItsWidget(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	var observer *ScrollNotificationObserverState
	ts.PumpWidget(ltr(NewScrollNotificationObserver(NewBuilder(func(bc BuildContext) Widget {
		observer = ScrollNotificationObserverOf(bc)
		return NewSizedBox(nil)
	}))))
	ts.PumpWidget(ltr(NewSizedBox(nil)))
	requireTag(t, "ScrollNotificationObserverDisposed", func() { observer.AddListener(func(ScrollNotification) {}) })
}

func TestScrollbarThumbDragTriggersScrollStartUpdateEndNotifications(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	var last ScrollNotification
	scrollable := testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(1200.0)})}, listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})
	ts.PumpWidget(ltr(NewMediaQuery(MediaQueryData{}, NewSizedBox(NewRawScrollbar(NewNotificationListener[ScrollNotification](scrollable, func(n ScrollNotification) bool {
		switch n.(type) {
		case *ScrollStartNotification, *ScrollUpdateNotification, *ScrollEndNotification:
			last = n
		}
		return true
	}), RawScrollbarOpts{ThumbVisibility: foundation.Some(true), Controller: controller}), SizedBoxOpts{Width: foundation.Some(800.0), Height: foundation.Some(600.0)}))))
	ts.PumpAndSettle()
	if controller.Offset() != 0 || last != nil {
		t.Fatalf("offset %v, notification %v", controller.Offset(), last)
	}
	g := ts.StartGesture(ui.Offset{Dx: 797, Dy: 45})
	ts.PumpAndSettle()
	if _, ok := last.(*ScrollStartNotification); !ok {
		t.Fatalf("notification = %v, want a start", last)
	}
	for _, want := range []float64{20, 40} {
		g.MoveBy(ui.Offset{Dy: 10})
		ts.PumpAndSettle()
		if _, ok := last.(*ScrollUpdateNotification); !ok || controller.Offset() != want {
			t.Fatalf("notification = %v offset = %v, want an update at %v", last, controller.Offset(), want)
		}
	}
	g.Up()
	ts.PumpAndSettle()
	if _, ok := last.(*ScrollEndNotification); !ok {
		t.Errorf("notification = %v, want an end", last)
	}
}
