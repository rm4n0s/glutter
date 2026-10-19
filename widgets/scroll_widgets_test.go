package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Tests of the small widgets of the scrolling step: PrimaryScrollController,
// TrackingScrollController and Viewport. The 'TrackingScrollController saves
// offset' case of tracking_scroll_controller_test.dart builds its pages with a
// PageView and waits for step 9.3.

func TestPrimaryScrollControllerShouldInherit(t *testing.T) {
	controller := NewScrollController()
	defer controller.Dispose()
	tests := []struct {
		name      string
		platform  foundation.TargetPlatform
		direction painting.Axis
		opts      PrimaryScrollControllerOpts
		none      bool
		want      bool
	}{
		{"vertical on Android", foundation.TargetPlatformAndroid, painting.AxisVertical, PrimaryScrollControllerOpts{}, false, true},
		{"vertical on iOS", foundation.TargetPlatformIOS, painting.AxisVertical, PrimaryScrollControllerOpts{}, false, true},
		{"vertical on Fuchsia", foundation.TargetPlatformFuchsia, painting.AxisVertical, PrimaryScrollControllerOpts{}, false, true},
		{"vertical on Linux", foundation.TargetPlatformLinux, painting.AxisVertical, PrimaryScrollControllerOpts{}, false, false},
		{"vertical on Windows", foundation.TargetPlatformWindows, painting.AxisVertical, PrimaryScrollControllerOpts{}, false, false},
		{"vertical on macOS", foundation.TargetPlatformMacOS, painting.AxisVertical, PrimaryScrollControllerOpts{}, false, false},
		{"horizontal on Android", foundation.TargetPlatformAndroid, painting.AxisHorizontal, PrimaryScrollControllerOpts{}, false, false},
		{"horizontal if asked", foundation.TargetPlatformAndroid, painting.AxisHorizontal,
			PrimaryScrollControllerOpts{ScrollDirection: foundation.Some(painting.AxisHorizontal)}, false, true},
		{"desktop if asked", foundation.TargetPlatformLinux, painting.AxisVertical,
			PrimaryScrollControllerOpts{AutomaticallyInheritForPlatforms: []foundation.TargetPlatform{foundation.TargetPlatformLinux}}, false, true},
		{"none", foundation.TargetPlatformAndroid, painting.AxisVertical, PrimaryScrollControllerOpts{}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usePlatform(t, tt.platform)
			ts := gluttertest.New(t)
			var got bool
			var found ScrollController
			child := NewBuilder(func(bc BuildContext) Widget {
				got = PrimaryScrollControllerShouldInherit(bc, tt.direction)
				found = PrimaryScrollControllerMaybeOf(bc)
				return NewSizedBox(nil)
			})
			var root Widget = NewPrimaryScrollController(controller, child, tt.opts)
			if tt.none {
				root = NewPrimaryScrollControllerNone(child)
			}
			ts.PumpWidget(ltr(root))
			if got != tt.want {
				t.Errorf("shouldInherit = %v, want %v", got, tt.want)
			}
			if tt.none && found != nil || !tt.none && found != ScrollController(controller) {
				t.Errorf("controller = %v", found)
			}
		})
	}
}

func TestPrimaryScrollControllerNoneBlocksTheOneAbove(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	var found ScrollController
	var inherit bool
	ts.PumpWidget(ltr(NewPrimaryScrollController(controller, NewPrimaryScrollControllerNone(NewBuilder(func(bc BuildContext) Widget {
		found = PrimaryScrollControllerMaybeOf(bc)
		inherit = PrimaryScrollControllerShouldInherit(bc, painting.AxisVertical)
		return NewSizedBox(nil)
	})))))
	if found != nil || inherit {
		t.Errorf("controller %v, shouldInherit %v below a none", found, inherit)
	}
}

func TestPrimaryScrollControllerNotifiesDependentsWhenTheControllerChanges(t *testing.T) {
	ts := gluttertest.New(t)
	a, b := NewScrollController(), NewScrollController()
	defer a.Dispose()
	defer b.Dispose()
	builds := 0
	var found ScrollController
	child := NewBuilder(func(bc BuildContext) Widget { // one widget, so that only a notification rebuilds it
		builds++
		found = PrimaryScrollControllerOf(bc)
		return NewSizedBox(nil)
	})
	frame := func(c ScrollController) Widget { return ltr(NewPrimaryScrollController(c, child)) }
	ts.PumpWidget(frame(a))
	ts.PumpWidget(frame(a))
	if builds != 1 || found != ScrollController(a) {
		t.Fatalf("builds = %d, controller = %v after the same controller", builds, found)
	}
	ts.PumpWidget(frame(b))
	if builds != 2 || found != ScrollController(b) {
		t.Errorf("builds = %d, controller = %v after another controller", builds, found)
	}
}

func TestTrackingScrollControllerCallsOnAttachAndOnDetach(t *testing.T) {
	ts := gluttertest.New(t)
	attach, detach := 0, 0
	controller := NewTrackingScrollController(ScrollControllerOpts{
		OnAttach: func(ScrollPosition) { attach++ }, OnDetach: func(ScrollPosition) { detach++ }})
	defer controller.Dispose()
	ts.PumpWidget(ltr(testList(items(10, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})))
	ts.PumpAndSettle()
	if attach != 1 || detach != 0 {
		t.Fatalf("attach = %d, detach = %d", attach, detach)
	}
	ts.PumpWidget(NewContainer(nil))
	ts.PumpAndSettle()
	if attach != 1 || detach != 1 {
		t.Errorf("attach = %d, detach = %d", attach, detach)
	}
}

func TestTrackingScrollControllerStartsNewPositionsAtTheLastUpdatedOffset(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewTrackingScrollController()
	defer controller.Dispose()
	list := func(key string) Widget {
		return testList(items(10, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: controller, Key: foundation.NewValueKey(key)}})
	}
	ts.PumpWidget(ltr(list("a")))
	if controller.InitialScrollOffset() != 0 {
		t.Fatalf("initial offset = %v", controller.InitialScrollOffset())
	}
	controller.JumpTo(110)
	ts.PumpAndSettle()
	if controller.MostRecentlyUpdatedPosition() == nil || controller.InitialScrollOffset() != 110 {
		t.Fatalf("most recent position %v, initial offset %v", controller.MostRecentlyUpdatedPosition(), controller.InitialScrollOffset())
	}
	ts.PumpWidget(ltr(list("b"))) // a new scrollable starts where the last one was
	if got := controller.Position().Pixels(); got != 110 {
		t.Errorf("the new position starts at %v, want 110", got)
	}
	ts.PumpWidget(NewContainer(nil))
	if controller.InitialScrollOffset() != 0 {
		t.Errorf("initial offset = %v after the positions are gone, want 0", controller.InitialScrollOffset())
	}
}

func TestViewportBuildsItsSliversInOrderAndKeepsTheCenter(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	offset := rendering.NewViewportOffsetFixed(0)
	key := func(i int) foundation.Key { return foundation.NewValueKey(i) }
	sliver := func(i int, h float64) Widget {
		return newSliverBox(NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(100 + i), Height: foundation.Some(h)}), WidgetOpts{Key: key(i)})
	}
	slivers := []Widget{sliver(0, 100), sliver(1, 200), sliver(2, 50)}
	ts.PumpWidget(ltr(NewViewport(offset, slivers, ViewportOpts{Center: key(1)})))
	// The center sliver is at the offset zero, the one before it above.
	top := func(i int) float64 { return ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey(100 + i))).Dy }
	if top(1) != 0 || top(2) != 200 || top(0) != -100 {
		t.Errorf("tops = %v, %v and %v, want -100, 0 and 200", top(0), top(1), top(2))
	}
	// Without a center the first sliver is it.
	ts.PumpWidget(ltr(NewViewport(offset, slivers)))
	if top(0) != 0 || top(1) != 100 || top(2) != 300 {
		t.Errorf("tops = %v, %v and %v, want 0, 100 and 300", top(0), top(1), top(2))
	}
	// An anchor places the center in the viewport.
	ts.PumpWidget(ltr(NewViewport(offset, slivers, ViewportOpts{Center: key(1), Anchor: 0.5})))
	if top(1) != 300 {
		t.Errorf("the center is at %v with anchor 0.5, want 300", top(1))
	}
	// A viewport without slivers builds nothing and has no center.
	ts.PumpWidget(ltr(NewViewport(offset, nil)))
	if ts.Count(gluttertest.ByType[*Viewport]()) != 1 || ts.Size(gluttertest.ByType[*Viewport]()) != (ui.Size{Width: 800, Height: 600}) {
		t.Error("the empty viewport is not there")
	}
}

func TestViewportMovingTheCenterKeepsTheSlivers(t *testing.T) {
	ts := gluttertest.New(t)
	offset := rendering.NewViewportOffsetFixed(0)
	sliver := func(i int) Widget {
		return newSliverBox(NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(100 + i), Height: foundation.Some(100.0)}), WidgetOpts{Key: foundation.NewValueKey(i)})
	}
	ts.PumpWidget(ltr(NewViewport(offset, []Widget{sliver(0), sliver(1), sliver(2)}, ViewportOpts{Center: foundation.NewValueKey(0)})))
	first := ts.Element(gluttertest.ByKey(foundation.NewValueKey(101)))
	ts.PumpWidget(ltr(NewViewport(offset, []Widget{sliver(0), sliver(1), sliver(2)}, ViewportOpts{Center: foundation.NewValueKey(2)})))
	if ts.Element(gluttertest.ByKey(foundation.NewValueKey(101))) != first {
		t.Error("the element of a sliver was rebuilt when the center moved")
	}
	if got := ts.TopLeft(gluttertest.ByKey(foundation.NewValueKey(102))).Dy; got != 0 {
		t.Errorf("the new center is at %v, want 0", got)
	}
}
