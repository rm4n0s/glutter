package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scroll_controller_test.dart. The
// GridView case waits for step 9.3, and the memory event test needs the leak
// tracker.

func listWith(controller ScrollController, key foundation.Key, physics ScrollPhysics, n int, height float64) Widget {
	return ltr(testList(items(n, height), listOpts{ScrollableOpts: ScrollableOpts{Key: key, Controller: controller, Physics: physics}}))
}

func TestScrollControllerControl(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()

	ts.PumpWidget(listWith(controller, nil, nil, 50, 200))
	check := func(step string, want float64) {
		t.Helper()
		if controller.Offset() != want || realOffset(ts) != controller.Offset() {
			t.Fatalf("%s: controller offset %v, real offset %v, want %v", step, controller.Offset(), realOffset(ts), want)
		}
	}
	check("initial", 0)
	controller.JumpTo(653)
	check("after jumpTo", 653)
	ts.Pump()
	check("after pump", 653)

	controller.AnimateTo(326, 300*time.Millisecond, animation.Curves.Ease)
	ts.PumpAndSettle()
	check("after animateTo", 326)

	// A widget with another key is a new Scrollable, which starts again at the
	// initial offset of the controller.
	ts.PumpWidget(listWith(controller, foundation.NewKey("second"), nil, 50, 200))
	check("new scrollable", 0)
	controller.JumpTo(653)
	check("new scrollable, after jumpTo", 653)

	controller2 := NewScrollController()
	defer controller2.Dispose()
	ts.PumpWidget(listWith(controller2, foundation.NewKey("second"), nil, 50, 200))
	requirePanic(t, func() { controller.Offset() })
	if controller2.Offset() != 653 || realOffset(ts) != controller2.Offset() {
		t.Fatalf("second controller offset %v, real offset %v, want 653", controller2.Offset(), realOffset(ts))
	}
	requirePanic(t, func() { controller.JumpTo(120) })
	requirePanic(t, func() { controller.AnimateTo(132, 300*time.Millisecond, animation.Curves.Ease) })

	ts.PumpWidget(listWith(controller2, foundation.NewKey("second"), NewBouncingScrollPhysics(), 50, 200))
	if controller2.Offset() != 653 || realOffset(ts) != controller2.Offset() {
		t.Fatalf("after new physics: offset %v, real offset %v, want 653", controller2.Offset(), realOffset(ts))
	}
	controller2.JumpTo(432)
	if controller2.Offset() != 432 || realOffset(ts) != 432 {
		t.Fatalf("after jumpTo: offset %v, real offset %v, want 432", controller2.Offset(), realOffset(ts))
	}
	ts.Pump()
	if controller2.Offset() != 432 || realOffset(ts) != 432 {
		t.Fatalf("after pump: offset %v, real offset %v, want 432", controller2.Offset(), realOffset(ts))
	}
}

func TestScrollControllerDrivenActivityEndingAfterDispose(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(ltr(testList([]Widget{NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(200000.0)})},
		listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})))
	controller.AnimateTo(1000, time.Second, animation.Curves.Linear)
	ts.Pump() // Start the animation.
	// We will now change the tree on the same frame as the animation ends.
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewSizedBox(nil)), gluttertest.PumpWidgetOpts{Duration: 2 * time.Second})
}

func TestScrollControllerWithoutPositions(t *testing.T) {
	skipUnlessDebug(t)
	controller := NewScrollController()
	defer controller.Dispose()
	t.Run("read operations fail", func(t *testing.T) {
		requireTag(t, "ScrollControllerNotAttached", func() { controller.Offset() })
		requireTag(t, "ScrollControllerNotAttached", func() { controller.Position() })
	})
	t.Run("write operations fail", func(t *testing.T) {
		requireTag(t, "ScrollControllerNotAttached", func() { controller.AnimateTo(1, time.Second, animation.Curves.Linear) })
		requireTag(t, "ScrollControllerNotAttached", func() { controller.JumpTo(1) })
	})
}

func twoListsOnOneController(controller ScrollController) Widget {
	inner := func() Widget {
		return NewContainer(testList(items(50, 200), listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}}),
			ContainerOpts{Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxHeight: foundation.Some(500.0)}))})
	}
	return ltr(testList([]Widget{inner(), inner()}))
}

func TestScrollControllerWithTwoPositions(t *testing.T) {
	useAndroid(t) // the desktop scrollbar needs one position
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(twoListsOnOneController(controller))
	if len(controller.Positions()) != 2 {
		t.Fatalf("positions = %d, want 2", len(controller.Positions()))
	}
	t.Run("read operations fail", func(t *testing.T) {
		skipUnlessDebug(t)
		requireTag(t, "ScrollControllerMultipleClients", func() { controller.Offset() })
		requireTag(t, "ScrollControllerMultipleClients", func() { controller.Position() })
	})
	t.Run("write operations do not fail", func(t *testing.T) {
		ts.Binding.UIThread().Adopt() // a subtest runs on its own goroutine
		controller.JumpTo(1)
		controller.AnimateTo(1, time.Second, animation.Curves.Linear)
		ts.PumpAndSettle()
	})
}

func TestScrollControllersNotifyWhenThePositionChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	var log []float64
	controller.AddListener(func() { log = append(log, controller.Offset()) })
	ts.PumpWidget(listWith(controller, nil, nil, 50, 200))
	if len(log) != 0 {
		t.Fatalf("log = %v, want empty", log)
	}
	ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: -250})
	if len(log) != 2 || log[0] != 20 || log[1] != 250 {
		t.Fatalf("log = %v, want [20 250]", log)
	}
	log = nil
	controller.Dispose()
	ts.Drag(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: -130})
	if len(log) != 0 {
		t.Fatalf("log after dispose = %v, want empty", log)
	}
}

func TestScrollControllerKeepScrollOffset(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	bucket := NewPageStorageBucket()
	buildFrame := func(controller ScrollController) Widget {
		return ltr(NewPageStorage(bucket,
			NewKeyedSubtree(
				testList(items(50, 100), listOpts{ScrollableOpts: ScrollableOpts{Key: foundation.NewUniqueKey(), Controller: controller}}),
				WidgetOpts{Key: NewPageStorageKey("ListView")})))
	}
	topLeftOfItem := func(i int) ui.Offset { return ts.TopLeft(itemFinder(i)) }

	// keepScrollOffset: true (the default). The scroll offset is restored when
	// the list is recreated with a new ScrollController.

	// The initialScrollOffset is used in this case, because there's no saved
	// scroll offset.
	controller := NewScrollController(ScrollControllerOpts{InitialScrollOffset: 200})
	defer controller.Dispose()
	ts.PumpWidget(buildFrame(controller))
	if got := topLeftOfItem(2); got != (ui.Offset{}) {
		t.Fatalf("item 2 at %v, want the origin", got)
	}
	controller.JumpTo(2000)
	ts.Pump()
	if got := topLeftOfItem(20); got != (ui.Offset{}) {
		t.Fatalf("item 20 at %v, want the origin", got)
	}

	// The initialScrollOffset isn't used in this case, because the scroll offset
	// can be restored.
	controller = NewScrollController(ScrollControllerOpts{InitialScrollOffset: 25})
	defer controller.Dispose()
	ts.PumpWidget(buildFrame(controller))
	if controller.Offset() != 2000 {
		t.Fatalf("restored offset = %v, want 2000", controller.Offset())
	}
	if got := topLeftOfItem(20); got != (ui.Offset{}) {
		t.Fatalf("item 20 at %v, want the origin", got)
	}

	// keepScrollOffset: false. The scroll offset is -not- restored when the list
	// is recreated with a new ScrollController and the initialScrollOffset is used.
	controller = NewScrollController(ScrollControllerOpts{KeepScrollOffset: foundation.Some(false), InitialScrollOffset: 100})
	defer controller.Dispose()
	ts.PumpWidget(buildFrame(controller))
	if controller.Offset() != 100 {
		t.Fatalf("offset = %v, want 100", controller.Offset())
	}
	if got := topLeftOfItem(1); got != (ui.Offset{}) {
		t.Fatalf("item 1 at %v, want the origin", got)
	}
}

func TestIsScrollingNotifierWorksWithPointerScroll(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	isScrolling := false
	controller.AddListener(func() { isScrolling = controller.Position().IsScrollingNotifier().Value() })
	ts.PumpWidget(listWith(controller, nil, nil, 50, 100))
	ts.Scroll(gluttertest.ByType[*Scrollable](), ui.Offset{Dy: 300})
	// When the listener was notified, the value of the isScrollingNotifier should
	// have been true.
	if !isScrolling {
		t.Error("the scroll position was not scrolling when it notified")
	}
}

func TestTrackingScrollController(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewTrackingScrollController(ScrollControllerOpts{InitialScrollOffset: 50})
	defer controller.Dispose()
	ts.PumpWidget(listWith(controller, nil, nil, 50, 100))
	if controller.InitialScrollOffset() != 50 || controller.MostRecentlyUpdatedPosition() != nil {
		t.Fatalf("initial offset %v, most recent %v", controller.InitialScrollOffset(), controller.MostRecentlyUpdatedPosition())
	}
	controller.JumpTo(300)
	if controller.MostRecentlyUpdatedPosition() != controller.Position() || controller.InitialScrollOffset() != 300 {
		t.Errorf("most recent %v, initial offset %v, want the position and 300", controller.MostRecentlyUpdatedPosition(), controller.InitialScrollOffset())
	}
	ts.PumpWidget(ltr(NewSizedBox(nil)))
	if controller.MostRecentlyUpdatedPosition() != nil || controller.InitialScrollOffset() != 50 {
		t.Errorf("after the scrollable is gone: most recent %v, initial offset %v", controller.MostRecentlyUpdatedPosition(), controller.InitialScrollOffset())
	}
}

func TestScrollControllerString(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController(ScrollControllerOpts{DebugLabel: "list", InitialScrollOffset: 10})
	defer controller.Dispose()
	if got := controller.String(); !stringsContain(got, "list", "initialScrollOffset: 10.0, ", "no clients") {
		t.Errorf("String = %q", got)
	}
	ts.PumpWidget(listWith(controller, nil, nil, 50, 100))
	if got := controller.String(); !stringsContain(got, "one client, offset 10.0") {
		t.Errorf("String = %q", got)
	}
}
