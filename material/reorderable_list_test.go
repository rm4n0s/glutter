// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material_test

import (
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/material/reorderable_list_test.dart.
// Not ported: the semantics tests, the golden tests of the dragged item, the tests of
// the scroll controller and of the auto scroll speed (SliverReorderableList is tested
// in widgets with the same engine), and the tests that build a list without an Overlay
// or MaterialLocalizations, which only check the debug assertions of those widgets.

// reorderItem is a list item of the test: an expanded box with a key.
func reorderItem(id int, extent float64, horizontal bool) widgets.Widget {
	o := widgets.ContainerOpts{Key: foundation.NewValueKey(id), Color: foundation.Some(ui.NewColorFromARGB(0xFF, 0xCC, 0xCC, 0xCC))}
	if horizontal {
		o.Width = foundation.Some(extent)
	} else {
		o.Height = foundation.Some(extent)
	}
	return widgets.NewContainer(nil, o)
}

func reorderItemKey(id int) foundation.Key { return foundation.NewValueKey(id) }

// reorderHandle is the drag handle of the item at the position: the handles are in the
// order of the items, as the list is not scrolled.
func reorderHandle(position int) gluttertest.Finder {
	return gluttertest.At(gluttertest.ByIcon(Icons.DragHandle()), position)
}

// reorderTestList is the list of the items, which moves them in the slice when they
// are dropped.
type reorderTestList struct {
	items    []int
	moves    [][2]int
	starts   []int
	ends     []int
	opts     ReorderableListViewOpts
	platform foundation.TargetPlatform
}

func (l *reorderTestList) widget() widgets.Widget {
	return widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
		o := l.opts
		horizontal := o.ScrollDirection == painting.AxisHorizontal
		o.OnReorderItem = func(from, to int) {
			l.moves = append(l.moves, [2]int{from, to})
			setState(func() {
				v := l.items[from]
				l.items = slices.Delete(l.items, from, from+1)
				l.items = slices.Insert(l.items, to, v)
			})
		}
		o.OnReorderStart = func(i int) { l.starts = append(l.starts, i) }
		o.OnReorderEnd = func(i int) { l.ends = append(l.ends, i) }
		return NewReorderableListViewBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
			return reorderItem(l.items[i], 100, horizontal)
		}, len(l.items), o)
	})
}

func (l *reorderTestList) pump(t *testing.T) *gluttertest.Tester {
	t.Helper()
	platform := l.platform
	if platform == foundation.TargetPlatformUnspecified {
		platform = foundation.TargetPlatformLinux
	}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{
		Theme: NewThemeData(ThemeDataOpts{Platform: foundation.Some(platform)}),
		Home:  NewScaffold(ScaffoldOpts{Body: l.widget()}),
	}))
	return ts
}

func TestReorderableListViewDragsByTheHandleOnTheDesktop(t *testing.T) {
	l := &reorderTestList{items: []int{0, 1, 2, 3}}
	ts := l.pump(t)
	for i := range 4 {
		if n := ts.Count(reorderHandle(i)); n != 1 {
			t.Fatalf("item %d has %d drag handles, want 1", i, n)
		}
	}
	// The handle is at the end of the item, 8 away from the edge.
	handle := ts.Center(reorderHandle(0))
	if handle.Dx < 700 || handle.Dy != 50 {
		t.Errorf("handle of item 0 is at %v, want the end of the first item", handle)
	}

	g := ts.StartGesture(handle)
	g.MoveBy(ui.Offset{Dy: 60})
	ts.Pump()
	g.MoveBy(ui.Offset{Dy: 60})
	ts.Pump()
	g.Up()
	ts.PumpAndSettle()

	if want := [][2]int{{0, 1}}; !slices.Equal(l.moves, want) {
		t.Errorf("moves = %v, want %v", l.moves, want)
	}
	if want := []int{1, 0, 2, 3}; !slices.Equal(l.items, want) {
		t.Errorf("items = %v, want %v", l.items, want)
	}
	// The drag ends at the place where the item is inserted, before the item is
	// removed from the old index: after the second item.
	if !slices.Equal(l.starts, []int{0}) || !slices.Equal(l.ends, []int{2}) {
		t.Errorf("onReorderStart %v, onReorderEnd %v, want [0] and [2]", l.starts, l.ends)
	}
	if y := ts.TopLeft(gluttertest.ByKey(reorderItemKey(0))).Dy; y != 100 {
		t.Errorf("item 0 is at y=%v, want 100", y)
	}
}

func TestReorderableListViewDragsHorizontally(t *testing.T) {
	l := &reorderTestList{items: []int{0, 1, 2, 3}, opts: ReorderableListViewOpts{ScrollDirection: painting.AxisHorizontal}}
	ts := l.pump(t)
	// The handle is at the bottom of the item, 8 away from the edge.
	handle := ts.Center(reorderHandle(0))
	if handle.Dx != 50 || handle.Dy < 500 {
		t.Fatalf("handle of item 0 is at %v, want the bottom center of the first item", handle)
	}
	g := ts.StartGesture(handle)
	g.MoveBy(ui.Offset{Dx: 60})
	ts.Pump()
	g.MoveBy(ui.Offset{Dx: 60})
	ts.Pump()
	g.Up()
	ts.PumpAndSettle()
	if want := []int{1, 0, 2, 3}; !slices.Equal(l.items, want) {
		t.Errorf("items = %v, want %v", l.items, want)
	}
}

func TestReorderableListViewDragsByLongPressOnMobile(t *testing.T) {
	for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia} {
		t.Run(platform.String(), func(t *testing.T) {
			l := &reorderTestList{items: []int{0, 1, 2, 3}, platform: platform}
			ts := l.pump(t)
			if n := ts.Count(gluttertest.ByIcon(Icons.DragHandle())); n != 0 {
				t.Fatalf("%d drag handles on a mobile platform, want none", n)
			}
			// A drag without the long press scrolls the list instead.
			ts.DragFrom(ui.Offset{Dx: 400, Dy: 50}, ui.Offset{Dy: 120})
			ts.PumpAndSettle()
			if len(l.moves) != 0 {
				t.Fatalf("a drag without a long press moved %v", l.moves)
			}
			ts.PumpAndSettle()

			g := ts.StartGesture(ui.Offset{Dx: 400, Dy: 50})
			ts.Pump(gestures.LongPressTimeout + gestures.PressTimeout)
			g.MoveBy(ui.Offset{Dy: 60})
			ts.Pump()
			g.MoveBy(ui.Offset{Dy: 60})
			ts.Pump()
			g.Up()
			ts.PumpAndSettle()
			if want := []int{1, 0, 2, 3}; !slices.Equal(l.items, want) {
				t.Errorf("items = %v, want %v", l.items, want)
			}
		})
	}
}

func TestReorderableListViewWithoutDefaultDragHandles(t *testing.T) {
	l := &reorderTestList{items: []int{0, 1, 2}, opts: ReorderableListViewOpts{BuildDefaultDragHandles: foundation.Some(false)}}
	ts := l.pump(t)
	if n := ts.Count(gluttertest.ByIcon(Icons.DragHandle())); n != 0 {
		t.Errorf("%d drag handles, want none", n)
	}
	ts.DragFrom(ts.Center(gluttertest.ByKey(reorderItemKey(0))), ui.Offset{Dy: 150})
	ts.PumpAndSettle()
	if len(l.moves) != 0 {
		t.Errorf("a drag moved %v without a drag handle", l.moves)
	}
}

func TestReorderableListViewMouseCursor(t *testing.T) {
	tests := []struct {
		name   string
		cursor widgets.WidgetStateMouseCursor
		hover  string
		drag   string
	}{
		{"default", nil, "grab", "grabbing"},
		{"given", widgets.WidgetStateMouseCursorResolveWith(func(s widgets.WidgetStates) services.MouseCursor {
			if s.Has(widgets.WidgetStateDragged) {
				return services.SystemMouseCursors.Click
			}
			return services.SystemMouseCursors.Text
		}), "text", "click"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &reorderTestList{items: []int{0, 1, 2}}
			if tt.cursor != nil {
				l.opts.MouseCursor = tt.cursor
			}
			ts := l.pump(t)
			ts.Hover(ts.Center(reorderHandle(1)))
			ts.Pump()
			last := func() string {
				c := ts.ActivatedCursors()
				if len(c) == 0 {
					return ""
				}
				return c[len(c)-1].Kind
			}
			if got := last(); got != tt.hover {
				t.Errorf("cursor over the handle = %q, want %q", got, tt.hover)
			}
			g := ts.StartGesture(ts.Center(reorderHandle(1)), gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
			g.MoveBy(ui.Offset{Dy: 30})
			ts.Pump()
			if got := last(); got != tt.drag {
				t.Errorf("cursor while dragging = %q, want %q", got, tt.drag)
			}
			g.Up()
			ts.PumpAndSettle()
		})
	}
}

func TestReorderableListViewDefaultProxyDecoratorRaisesTheItem(t *testing.T) {
	l := &reorderTestList{items: []int{0, 1, 2}}
	ts := l.pump(t)
	elevations := func() []float64 {
		var out []float64
		for _, w := range ts.Widgets(gluttertest.ByType[*Material]()) {
			out = append(out, w.(*Material).Elevation())
		}
		return out
	}
	before := elevations()
	g := ts.StartGesture(ts.Center(reorderHandle(0)))
	g.MoveBy(ui.Offset{Dy: 30})
	ts.Pump()
	ts.Pump(300 * time.Millisecond)
	raised := 0.0
	for _, e := range elevations() {
		raised = max(raised, e)
	}
	if raised != 6 {
		t.Errorf("the dragged item has the elevation %v (before: %v), want 6", raised, before)
	}
	g.Up()
	ts.PumpAndSettle()
}

func TestReorderableListViewGivenProxyDecorator(t *testing.T) {
	var got []int
	l := &reorderTestList{items: []int{0, 1, 2}, opts: ReorderableListViewOpts{
		ProxyDecorator: func(child widgets.Widget, index int, _ animation.Animation[float64]) widgets.Widget {
			got = append(got, index)
			return child
		},
	}}
	ts := l.pump(t)
	g := ts.StartGesture(ts.Center(reorderHandle(1)))
	g.MoveBy(ui.Offset{Dy: 30})
	ts.Pump()
	if len(got) == 0 || got[0] != 1 {
		t.Errorf("the proxy decorator was called for %v, want the index 1", got)
	}
	g.Up()
	ts.PumpAndSettle()
}

// reorderProbe counts how many states are created: an item that moves keeps its state.
type reorderProbe struct {
	widgets.StatefulWidget
	inits *int
}

func (w *reorderProbe) CreateState() widgets.State { return &reorderProbeState{} }

type reorderProbeState struct {
	widgets.StateOf[*reorderProbe]
}

func (s *reorderProbeState) InitState() { *s.Widget().inits++ }
func (s *reorderProbeState) Build(widgets.BuildContext) widgets.Widget {
	return widgets.NewSizedBox(nil)
}

func TestReorderableListViewKeepsTheStateOfTheItems(t *testing.T) {
	inits := 0
	items := []int{0, 1, 2}
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	var moves [][2]int
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: widgets.NewStatefulBuilder(
		func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			return NewReorderableListViewBuilder(func(_ widgets.BuildContext, i int) widgets.Widget {
				return widgets.NewContainer(&reorderProbe{StatefulWidget: widgets.NewStatefulWidgetBase(), inits: &inits},
					widgets.ContainerOpts{Key: reorderItemKey(items[i]), Height: foundation.Some(100.0)})
			}, len(items), ReorderableListViewOpts{OnReorderItem: func(from, to int) {
				moves = append(moves, [2]int{from, to})
				setState(func() {
					v := items[from]
					items = slices.Delete(items, from, from+1)
					items = slices.Insert(items, to, v)
				})
			}})
		})})}))
	if inits != 3 {
		t.Fatalf("%d states after the first build, want 3", inits)
	}
	g := ts.StartGesture(ts.Center(reorderHandle(0)))
	g.MoveBy(ui.Offset{Dy: 60})
	ts.Pump()
	g.MoveBy(ui.Offset{Dy: 60})
	ts.Pump()
	g.Up()
	ts.PumpAndSettle()
	if len(moves) != 1 {
		t.Fatalf("moves = %v, want one", moves)
	}
	if inits != 3 {
		t.Errorf("%d states after the reorder, want 3: an item that moved was rebuilt from scratch", inits)
	}
}

func TestReorderableListViewHeaderFooterAndPadding(t *testing.T) {
	header := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("header"), Height: foundation.Some(50.0)})
	footer := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("footer"), Height: foundation.Some(30.0)})
	tests := []struct {
		name   string
		opts   ReorderableListViewOpts
		header ui.Offset
		item0  ui.Offset
		footer ui.Offset
		hw     float64
	}{
		// The items are 3 of 100. The padding is 10 on every side, and the header and the
		// footer share it with the list: the header has no bottom padding, nor the footer a
		// top padding.
		{"vertical", ReorderableListViewOpts{Header: header, Footer: footer, Padding: painting.NewEdgeInsetsAll(10)},
			ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dx: 10, Dy: 60}, ui.Offset{Dx: 10, Dy: 360}, 780},
		{"no header or footer", ReorderableListViewOpts{Padding: painting.NewEdgeInsetsAll(10)},
			ui.Offset{}, ui.Offset{Dx: 10, Dy: 10}, ui.Offset{}, 0},
		{"header only", ReorderableListViewOpts{Header: header, Padding: painting.NewEdgeInsetsAll(10)},
			ui.Offset{Dx: 10, Dy: 10}, ui.Offset{Dx: 10, Dy: 60}, ui.Offset{}, 780},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &reorderTestList{items: []int{0, 1, 2}, opts: tt.opts}
			ts := l.pump(t)
			if tt.opts.Header != nil {
				if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("header"))); got != tt.header {
					t.Errorf("header at %v, want %v", got, tt.header)
				}
				if got := ts.Size(gluttertest.ByKey(foundation.NewValueKey("header"))).Width; got != tt.hw {
					t.Errorf("header width %v, want %v", got, tt.hw)
				}
			}
			if got := ts.TopLeft(gluttertest.ByKey(reorderItemKey(0))); got != tt.item0 {
				t.Errorf("item 0 at %v, want %v", got, tt.item0)
			}
			if tt.opts.Footer != nil {
				if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("footer"))); got != tt.footer {
					t.Errorf("footer at %v, want %v", got, tt.footer)
				}
			}
		})
	}
}

func TestReorderableListViewReverseHeader(t *testing.T) {
	header := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey("header"), Height: foundation.Some(50.0)})
	l := &reorderTestList{items: []int{0, 1, 2}, opts: ReorderableListViewOpts{Header: header, Reverse: true, Padding: painting.NewEdgeInsetsAll(10)}}
	ts := l.pump(t)
	// The header is at the start of the scroll, which is the bottom, with the padding
	// below it and none above it; the items follow upward.
	if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey("header"))); got != (ui.Offset{Dx: 10, Dy: 540}) {
		t.Errorf("header at %v, want (10, 540)", got)
	}
	if got := ts.TopLeft(gluttertest.ByKey(reorderItemKey(0))); got != (ui.Offset{Dx: 10, Dy: 440}) {
		t.Errorf("item 0 at %v, want (10, 440)", got)
	}
}

func TestReorderableListViewAssertions(t *testing.T) {
	skipUnlessDebug(t)
	builder := func(widgets.BuildContext, int) widgets.Widget { return widgets.NewSizedBox(nil) }
	noop := func(int, int) {}
	keyed := widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Key: foundation.NewValueKey(1)})
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"a child without a key", "ReorderableListViewChildKeyMissing", func() {
			NewReorderableListView([]widgets.Widget{keyed, widgets.NewSizedBox(nil)}, ReorderableListViewOpts{OnReorderItem: noop})
		}},
		{"negative itemCount", "ReorderableItemCountNegative", func() {
			NewReorderableListViewBuilder(builder, -1, ReorderableListViewOpts{OnReorderItem: noop})
		}},
		{"neither onReorder nor onReorderItem", "ReorderableCallbackInvalid", func() { NewReorderableListViewBuilder(builder, 1) }},
		{"both onReorder and onReorderItem", "ReorderableCallbackInvalid", func() {
			NewReorderableListViewBuilder(builder, 1, ReorderableListViewOpts{OnReorder: noop, OnReorderItem: noop})
		}},
		{"neither onReorder nor onReorderItem (children)", "ReorderableCallbackInvalid", func() {
			NewReorderableListView([]widgets.Widget{keyed})
		}},
		{"itemExtent and prototypeItem", "ReorderableExtentConflict", func() {
			NewReorderableListViewBuilder(builder, 1, ReorderableListViewOpts{OnReorderItem: noop, ItemExtent: foundation.Some(10.0), PrototypeItem: keyed})
		}},
		{"itemExtentBuilder and itemExtent", "ReorderableExtentConflict", func() {
			NewReorderableListViewBuilder(builder, 1, ReorderableListViewOpts{
				OnReorderItem: noop, ItemExtent: foundation.Some(10.0), ItemExtentBuilder: rendering.ItemExtentBuilder(func(int, rendering.SliverLayoutDimensions) foundation.Opt[float64] { return foundation.Some(1.0) })})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := func() (v any) {
				defer func() { v = recover() }()
				tt.call()
				return nil
			}()
			err, ok := v.(error)
			if !ok {
				t.Fatalf("expected a panic with an error, got %v", v)
			}
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.tag) {
				t.Errorf("unexpected failure path: %v, want the tag %s", err, tt.tag)
			}
		})
	}

	t.Run("an item without a key", func(t *testing.T) {
		ts := gluttertest.New(t)
		ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{
			Body: NewReorderableListViewBuilder(builder, 1, ReorderableListViewOpts{OnReorderItem: noop})})}))
		appErr, ok := errors.FromError(ts.TakeError())
		if !ok || !appErr.HasRoute("ReorderableListViewItemKeyMissing") {
			t.Errorf("expected the tag ReorderableListViewItemKeyMissing, got %v", appErr)
		}
	})
}

func TestReorderableListViewChildren(t *testing.T) {
	var moves [][2]int
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(NewMaterialApp(MaterialAppOpts{Home: NewScaffold(ScaffoldOpts{Body: NewReorderableListView(
		[]widgets.Widget{reorderItem(0, 100, false), reorderItem(1, 100, false), reorderItem(2, 100, false)},
		ReorderableListViewOpts{OnReorderItem: func(from, to int) { moves = append(moves, [2]int{from, to}) }})})}))
	if n := ts.Count(gluttertest.ByType[*widgets.SliverReorderableList]()); n != 1 {
		t.Fatalf("%d reorderable slivers, want 1", n)
	}
	g := ts.StartGesture(ts.Center(reorderHandle(1)))
	g.MoveBy(ui.Offset{Dy: 60})
	ts.Pump()
	g.MoveBy(ui.Offset{Dy: 60})
	ts.Pump()
	g.Up()
	ts.PumpAndSettle()
	if want := [][2]int{{1, 2}}; !slices.Equal(moves, want) {
		t.Errorf("moves = %v, want %v", moves, want)
	}
}
