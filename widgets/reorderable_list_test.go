package widgets_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/reorderable_list_test.dart.
// Not ported: the semantics tests, the proxy decorator and drop animation tests,
// the auto scroll tests, and the tests that need DefaultTextStyle and IconTheme
// for the drag handle.

// The test font has no advance width for the text, so the items are filled by an
// expanded box: that is where the text is hit.

// reorderableApp is TestWidgetsApp of widgets_app_tester.dart: it gives the list
// the Directionality and the Overlay that the dragged item needs.
func reorderableApp(child Widget) Widget {
	return ltr(NewLocalizations(ui.NewLocale("en", "US"), []LocalizationsDelegate{DefaultWidgetsLocalizationsDelegate}, NewOverlayWrap(child)))
}

func reorderableMove(items *[]int) ReorderCallback {
	return func(from, to int) {
		v := (*items)[from]
		*items = slices.Delete(*items, from, from+1)
		*items = slices.Insert(*items, to, v)
	}
}

func reorderableTestList(items *[]int, o SliverReorderableListOpts) Widget {
	return NewStatefulBuilder(func(_ BuildContext, setState StateSetter) Widget {
		opts := o
		opts.OnReorderItem = func(from, to int) {
			setState(func() { reorderableMove(items)(from, to) })
		}
		return NewCustomScrollView([]Widget{NewSliverReorderableList(func(_ BuildContext, i int) Widget {
			return NewContainer(NewReorderableDragStartListener(i, NewSizedBoxExpand(NewText(fmt.Sprintf("item %d", (*items)[i])))),
				ContainerOpts{Key: foundation.NewValueKey((*items)[i]), Height: foundation.Some(100.0)})
		}, len(*items), opts)})
	})
}

func orderedEquals(t *testing.T, got, want []int) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("items = %v, want %v", got, want)
	}
}

func TestSliverReorderableListDragAndDrop(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	items := []int{0, 1, 2, 3, 4, 5, 6, 7}
	pressDragRelease := func(start, delta ui.Offset) {
		drag := ts.StartGesture(start)
		ts.Pump(gestures.PressTimeout)
		drag.MoveBy(delta)
		ts.Pump(gestures.PressTimeout)
		drag.Up()
		ts.PumpAndSettle()
	}
	check := func(visible, hidden []int) {
		t.Helper()
		for _, i := range visible {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("item %d", i))) != 1 {
				t.Errorf("item %d is not visible", i)
			}
		}
		for _, i := range hidden {
			if ts.Count(gluttertest.ByText(fmt.Sprintf("item %d", i))) != 0 {
				t.Errorf("item %d is visible", i)
			}
		}
	}
	at := func(i int) ui.Offset { return ts.TopLeft(gluttertest.ByText(fmt.Sprintf("item %d", i))) }
	center := func(i int) ui.Offset { return ts.Center(gluttertest.ByText(fmt.Sprintf("item %d", i))) }
	// The list is 800x600, 8 items of 800x100 with a text at the origin of the item.
	ts.PumpWidget(reorderableApp(reorderableTestList(&items, SliverReorderableListOpts{})))
	visible, hidden := []int{0, 1, 2, 3, 4, 5}, []int{6, 7}
	check(visible, hidden)

	// Drag item 0 down by less than half: it snaps back.
	pressDragRelease(ui.Offset{Dx: 12, Dy: 50}, ui.Offset{Dy: 40}) // Dart drags by 60 but misses the text
	check(visible, hidden)
	if at(0) != (ui.Offset{}) || at(1) != (ui.Offset{Dy: 100}) {
		t.Errorf("item 0 at %v, item 1 at %v", at(0), at(1))
	}
	orderedEquals(t, items, []int{0, 1, 2, 3, 4, 5, 6, 7})

	// Drag item 0 down by more than half: it displaces item 1.
	pressDragRelease(center(0), ui.Offset{Dy: 51})
	check(visible, hidden)
	if at(1) != (ui.Offset{}) || at(0) != (ui.Offset{Dy: 100}) {
		t.Errorf("item 1 at %v, item 0 at %v", at(1), at(0))
	}
	orderedEquals(t, items, []int{1, 0, 2, 3, 4, 5, 6, 7})

	// Drag item 0 back.
	pressDragRelease(center(0), ui.Offset{Dy: -51})
	check(visible, hidden)
	if at(0) != (ui.Offset{}) || at(1) != (ui.Offset{Dy: 100}) {
		t.Errorf("item 0 at %v, item 1 at %v", at(0), at(1))
	}
	orderedEquals(t, items, []int{0, 1, 2, 3, 4, 5, 6, 7})

	// Drag item 1 to item 3.
	pressDragRelease(center(1), ui.Offset{Dy: 151})
	check(visible, hidden)
	if at(0) != (ui.Offset{}) || at(1) != (ui.Offset{Dy: 300}) || at(3) != (ui.Offset{Dy: 200}) {
		t.Errorf("item 0 at %v, item 1 at %v, item 3 at %v", at(0), at(1), at(3))
	}
	orderedEquals(t, items, []int{0, 2, 3, 1, 4, 5, 6, 7})

	// Drag item 1 back.
	pressDragRelease(center(1), ui.Offset{Dy: -200})
	check(visible, hidden)
	if at(0) != (ui.Offset{}) || at(1) != (ui.Offset{Dy: 100}) || at(3) != (ui.Offset{Dy: 300}) {
		t.Errorf("item 0 at %v, item 1 at %v, item 3 at %v", at(0), at(1), at(3))
	}
	orderedEquals(t, items, []int{0, 1, 2, 3, 4, 5, 6, 7})
}

func TestSliverReorderableListWorksWithGestureSettings(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/103404
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	items := []int{0, 1, 2, 3, 4}
	calls := 0
	move := reorderableMove(&items)
	ts.PumpWidget(reorderableApp(NewMediaQuery(NewMediaQueryData(MediaQueryDataOpts{GestureSettings: foundation.Some(gestures.NewDeviceGestureSettings(foundation.Some(8.0)))}),
		NewCustomScrollView([]Widget{NewSliverReorderableList(func(_ BuildContext, i int) Widget {
			return NewSizedBox(NewReorderableDragStartListener(i, NewText(fmt.Sprintf("item %d", items[i]))),
				SizedBoxOpts{Key: foundation.NewValueKey(items[i]), Height: foundation.Some(100.0)})
		}, len(items), SliverReorderableListOpts{OnReorderItem: func(from, to int) { calls++; move(from, to) }})}))))
	drag := ts.StartGesture(ts.Center(gluttertest.ByText("item 0")))
	ts.Pump(gestures.PressTimeout)
	// Drag a little so that the multi drag recognizer competes with the vertical drag one.
	drag.MoveBy(ui.Offset{Dy: 10})
	ts.Pump()
	// Drag enough to move the first item down.
	drag.MoveBy(ui.Offset{Dy: 40})
	ts.Pump()
	drag.Up()
	ts.PumpAndSettle()
	if calls != 1 {
		t.Errorf("onReorderItem called %d times, want 1", calls)
	}
	orderedEquals(t, items, []int{1, 0, 2, 3, 4})
}

func TestSliverReorderableListCallsOnReorderStartAndEnd(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	items := []int{0, 1, 2, 3, 4, 5, 6, 7}
	start, end := foundation.None[int](), foundation.None[int]()
	item0 := gluttertest.ByTextContaining("item 0")
	ts.PumpWidget(reorderableApp(reorderableTestList(&items, SliverReorderableListOpts{
		OnReorderStart: func(i int) { start = foundation.Some(i) },
		OnReorderEnd:   func(i int) { end = foundation.Some(i) },
	})))
	drag := ts.StartGesture(ts.Center(item0))
	ts.Pump(gestures.PressTimeout)
	// Drag enough for the move to start.
	drag.MoveBy(ui.Offset{Dy: 20})
	if start != foundation.Some(0) || end.IsSome() {
		t.Errorf("start %v end %v", start, end)
	}
	// Move item 0 from index 0 to index 3.
	drag.MoveBy(ui.Offset{Dy: 300})
	ts.PumpAndSettle()
	drag.Up()
	ts.PumpAndSettle()
	if end != foundation.Some(3) {
		t.Errorf("end = %v, want 3", end)
	}
	start, end = foundation.None[int](), foundation.None[int]()
	drag = ts.StartGesture(ts.Center(item0))
	ts.Pump(gestures.PressTimeout)
	drag.MoveBy(ui.Offset{Dy: 20})
	if start != foundation.Some(2) || end.IsSome() {
		t.Errorf("start %v end %v", start, end)
	}
	// Move item 0 from index 2 to index 0.
	drag.MoveBy(ui.Offset{Dy: -200})
	ts.PumpAndSettle()
	drag.Up()
	ts.PumpAndSettle()
	if end != foundation.Some(0) {
		t.Errorf("end = %v, want 0", end)
	}
}

func TestReorderableListAssertions(t *testing.T) {
	skipUnlessDebug(t)
	builder := func(BuildContext, int) Widget { return NewSizedBox(nil) }
	noop := func(int, int) {}
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"negative itemCount", "ReorderableItemCountNegative", func() {
			NewSliverReorderableList(builder, -1, SliverReorderableListOpts{OnReorderItem: noop})
		}},
		{"neither onReorder nor onReorderItem (sliver)", "ReorderableCallbackInvalid", func() { NewSliverReorderableList(builder, 0) }},
		{"both onReorder and onReorderItem (sliver)", "ReorderableCallbackInvalid", func() {
			NewSliverReorderableList(builder, 0, SliverReorderableListOpts{OnReorder: noop, OnReorderItem: noop})
		}},
		{"neither onReorder nor onReorderItem", "ReorderableCallbackInvalid", func() { NewReorderableList(builder, 0) }},
		{"both onReorder and onReorderItem", "ReorderableCallbackInvalid", func() {
			NewReorderableList(builder, 0, ReorderableListOpts{OnReorder: noop, OnReorderItem: noop})
		}},
		{"both itemExtent and prototypeItem", "ReorderableExtentConflict", func() {
			NewReorderableList(builder, 3, ReorderableListOpts{ItemExtent: foundation.Some(30.0), PrototypeItem: NewSizedBox(nil), OnReorderItem: noop})
		}},
		{"both itemExtent and prototypeItem (sliver)", "ReorderableExtentConflict", func() {
			NewSliverReorderableList(builder, 3, SliverReorderableListOpts{ItemExtent: foundation.Some(30.0), PrototypeItem: NewSizedBox(nil), OnReorderItem: noop})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { requireTag(t, tc.tag, tc.call) })
	}
}

func TestReorderableListItemExtents(t *testing.T) {
	tests := []struct {
		name string
		opts func() ReorderableListOpts
	}{
		{"itemExtent", func() ReorderableListOpts { return ReorderableListOpts{ItemExtent: foundation.Some(30.0)} }},
		{"prototypeItem", func() ReorderableListOpts {
			return ReorderableListOpts{PrototypeItem: NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(30.0)})}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useAndroid(t)
			ts := gluttertest.New(t, gluttertest.Opts{})
			numbers := []int{0, 1, 2}
			opts := tc.opts()
			opts.OnReorderItem = func(from, to int) {}
			ts.PumpWidget(reorderableApp(NewReorderableList(func(_ BuildContext, i int) Widget {
				// Children with different heights.
				return NewSizedBox(NewReorderableDragStartListener(i, NewText(fmt.Sprint(numbers[i]))),
					SizedBoxOpts{Key: foundation.NewValueKey(numbers[i]), Height: foundation.Some(float64(20 + numbers[i]*10))})
			}, len(numbers), opts)))
			for _, n := range numbers {
				if h := ts.Size(gluttertest.ByKey(foundation.NewValueKey(n))).Height; h != 30 {
					t.Errorf("item %d height = %v, want 30", n, h)
				}
			}
		})
	}
}

func TestReorderableDragStartListenerEnabled(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		delayed       bool
		wantReordered bool
	}{
		{"drag start enabled", true, false, true},
		{"drag start disabled", false, false, false},
		{"delayed drag start enabled", true, true, true},
		{"delayed drag start disabled", false, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useAndroid(t)
			ts := gluttertest.New(t, gluttertest.Opts{})
			items := []int{0, 1, 2, 3, 4}
			calls := 0
			move := reorderableMove(&items)
			ts.PumpWidget(reorderableApp(NewReorderableList(func(_ BuildContext, i int) Widget {
				text := NewText(fmt.Sprintf("item %d", items[i]))
				var child Widget
				if tc.delayed {
					child = NewReorderableDelayedDragStartListener(i, text, ReorderableDragStartListenerOpts{Enabled: foundation.Some(tc.enabled)})
				} else {
					child = NewReorderableDragStartListener(i, text, ReorderableDragStartListenerOpts{Enabled: foundation.Some(tc.enabled)})
				}
				return NewSizedBox(child, SizedBoxOpts{Key: foundation.NewValueKey(items[i]), Height: foundation.Some(100.0)})
			}, len(items), ReorderableListOpts{OnReorderItem: func(from, to int) { calls++; move(from, to) }})))
			drag := ts.StartGesture(ts.Center(gluttertest.ByText("item 0")))
			if tc.delayed {
				ts.Pump(gestures.LongPressTimeout)
			} else {
				ts.Pump(gestures.PressTimeout)
			}
			drag.MoveBy(ui.Offset{Dy: 50})
			ts.Pump()
			drag.Up()
			ts.PumpAndSettle()
			if (calls == 1) != tc.wantReordered {
				t.Errorf("onReorderItem called %d times", calls)
			}
			if tc.wantReordered {
				orderedEquals(t, items, []int{1, 0, 2, 3, 4})
			} else {
				orderedEquals(t, items, []int{0, 1, 2, 3, 4})
			}
		})
	}
}

func TestReorderableListDoesNotCrashAtZeroArea(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(reorderableApp(NewCenter(NewSizedBoxShrink(NewReorderableList(func(_ BuildContext, i int) Widget {
		return NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(i)})
	}, 1, ReorderableListOpts{OnReorderItem: func(int, int) {}})))))
	if ts.Count(gluttertest.ByType[*ReorderableList]()) != 1 {
		t.Error("no list")
	}
}
