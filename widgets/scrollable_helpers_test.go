package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scrollable_helpers_test.dart. The
// keyboard cases and 'Can scroll using intents only' need Shortcuts, Actions and
// ScrollAction (step 10.6), and the deprecated clipBehavior has no counterpart;
// the autoscroller cases build their list without ListView.builder (step 9.3).

func TestScrollableDetailsCopyWithAndEqual(t *testing.T) {
	controller := NewScrollController()
	defer controller.Dispose()
	details := NewScrollableDetails(painting.AxisDirectionDown, ScrollableDetailsOpts{
		Controller: controller, Physics: NewAlwaysScrollableScrollPhysics(nil), DecorationClipBehavior: foundation.Some(ui.ClipHardEdge)})
	if copied := details.CopyWith(ScrollableDetailsCopy{}); !details.Equal(copied) {
		t.Fatalf("a copy with no changes differs: %v vs %v", details, copied)
	}
	clamping := NewClampingScrollPhysics(nil)
	copied := details.CopyWith(ScrollableDetailsCopy{Direction: painting.AxisDirectionLeft, Physics: clamping,
		DecorationClipBehavior: foundation.Some(ui.ClipNone)})
	want := NewScrollableDetails(painting.AxisDirectionLeft, ScrollableDetailsOpts{
		Controller: controller, Physics: clamping, DecorationClipBehavior: foundation.Some(ui.ClipNone)})
	if !copied.Equal(want) {
		t.Errorf("copy = %v, want %v", copied, want)
	}
	if copied.Equal(details) {
		t.Error("the changed copy equals the original")
	}
}

func TestScrollableDetailsString(t *testing.T) {
	controller := NewScrollController()
	defer controller.Dispose()
	bare := NewScrollableDetails(painting.AxisDirectionRight)
	if got, want := bare.String(), "ScrollableDetails(axisDirection: AxisDirection.right)"; got != want {
		t.Errorf("bare = %q, want %q", got, want)
	}
	full := NewScrollableDetails(painting.AxisDirectionDown, ScrollableDetailsOpts{
		Controller: controller, Physics: NewAlwaysScrollableScrollPhysics(nil), DecorationClipBehavior: foundation.Some(ui.ClipHardEdge)})
	for _, part := range []string{"ScrollableDetails(axisDirection: AxisDirection.down, scroll controller: ScrollController", "(no clients), ",
		"scroll physics: AlwaysScrollableScrollPhysics, decorationClipBehavior: Clip.hardEdge)"} {
		if !stringsContain(full.String(), part) {
			t.Errorf("%q does not contain %q", full.String(), part)
		}
	}
}

func TestScrollableDetailsConstructors(t *testing.T) {
	tests := []struct {
		name    string
		details ScrollableDetails
		want    painting.AxisDirection
	}{
		{"vertical", NewScrollableDetailsVertical(), painting.AxisDirectionDown},
		{"vertical reversed", NewScrollableDetailsVertical(ScrollableDetailsOpts{Reverse: true}), painting.AxisDirectionUp},
		{"horizontal", NewScrollableDetailsHorizontal(), painting.AxisDirectionRight},
		{"horizontal reversed", NewScrollableDetailsHorizontal(ScrollableDetailsOpts{Reverse: true}), painting.AxisDirectionLeft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.details.Direction != tt.want {
				t.Errorf("direction = %v, want %v", tt.details.Direction, tt.want)
			}
		})
	}
}

func TestScrollActionDirectionalIncrement(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(50, 100))))
	state := scrollableOf(ts)
	tests := []struct {
		name   string
		intent ScrollIntent
		want   float64
	}{
		{"line down", ScrollIntent{Direction: painting.AxisDirectionDown}, 50},
		{"line up", ScrollIntent{Direction: painting.AxisDirectionUp}, -50},
		{"page down", ScrollIntent{Direction: painting.AxisDirectionDown, Type: ScrollIncrementTypePage}, 0.8 * 600},
		{"page up", ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}, -0.8 * 600},
		{"the other axis", ScrollIntent{Direction: painting.AxisDirectionRight}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScrollActionGetDirectionalIncrement(state, tt.intent); got != tt.want {
				t.Errorf("increment = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScrollActionUsesTheIncrementCalculator(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var seen []ScrollIncrementType
	ts.PumpWidget(ltr(testList(items(50, 100), listOpts{ScrollableOpts: ScrollableOpts{
		IncrementCalculator: func(d ScrollIncrementDetails) float64 {
			seen = append(seen, d.Type)
			return 7
		}}})))
	state := scrollableOf(ts)
	if got := ScrollActionGetDirectionalIncrement(state, ScrollIntent{Direction: painting.AxisDirectionUp, Type: ScrollIncrementTypePage}); got != -7 {
		t.Errorf("increment = %v, want -7", got)
	}
	if len(seen) != 1 || seen[0] != ScrollIncrementTypePage {
		t.Errorf("the calculator saw %v", seen)
	}
}

func TestEdgeDraggingAutoScrollerScrollsNearTheEdges(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(ltr(testList(items(50, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})))
	scrolled := 0
	scroller := NewEdgeDraggingAutoScroller(scrollableOf(ts), 1, func() { scrolled++ })

	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 0, 100, 100)) // at the start: nothing to scroll back to
	ts.Pump()
	if scroller.Scrolling() || controller.Position().Pixels() != 0 {
		t.Fatalf("scrolling = %v, pixels = %v at the start", scroller.Scrolling(), controller.Position().Pixels())
	}

	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 520, 100, 100)) // past the bottom edge by 20
	if !scroller.Scrolling() {
		t.Fatal("not scrolling below the bottom edge")
	}
	ts.Pump()            // the animation starts
	ts.Pump(time.Second) // and ends
	if got := controller.Position().Pixels(); got != 20 {
		t.Errorf("pixels = %v, want 20 (the overdrag of one animation)", got)
	}
	ts.Pump(10 * time.Millisecond) // the animation is done after its duration
	if scrolled == 0 {
		t.Error("onScrollViewScrolled was not called")
	}
	scroller.StopAutoScroll()
	ts.PumpAndSettle() // the current animation ends
	before := controller.Position().Pixels()
	ts.Pump(time.Second)
	if got := controller.Position().Pixels(); got != before {
		t.Errorf("pixels moved from %v to %v after the stop", before, got)
	}

	// Past the top edge it scrolls back.
	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, -10, 100, 100))
	ts.Pump()
	ts.Pump(time.Second)
	if got := controller.Position().Pixels(); got >= before {
		t.Errorf("pixels = %v, want less than %v", got, before)
	}
	scroller.StopAutoScroll()
	ts.PumpAndSettle()
}

func TestEdgeDraggingAutoScrollerHandlesDragTargetSizeWithTransformScale(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewCenter(NewTransformScale(
		NewSizedBoxSquare(foundation.Some(400.0), testList(items(20, 100))),
		TransformScaleOpts{Scale: foundation.Some(0.5)}))))
	ts.PumpAndSettle()
	state := scrollableOf(ts)
	scroller := NewEdgeDraggingAutoScroller(state, 1, nil)
	box := state.Context().FindRenderObject().(rendering.Box)
	scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 0, box.Size().Width, box.Size().Height))
	ts.Pump()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	scroller.StopAutoScroll()
	ts.PumpAndSettle()
}

func TestEdgeDraggingAutoScrollerDragTargetTooLarge(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(50, 100))))
	scroller := NewEdgeDraggingAutoScroller(scrollableOf(ts), 1, nil)
	requireTag(t, "AutoScrollDragTargetTooLarge", func() {
		scroller.StartAutoScrollIfNecessary(ui.NewRectFromLTWH(0, 0, 100, 5000))
	})
}
