package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scroll_position_test.dart. 'whether
// we remember our scroll position' needs the Navigator (step 10.2), and
// 'jumpTo recommends deferred loading' builds its items lazily (step 9.3): the
// recommendation is tested here without a builder.

func TestScrollPositionJumpToNotifiesOnce(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(NewSizedBox(testList(items(50, 100)))))
	position := scrollableOf(ts).Position()
	count := 0
	position.AddListener(func() { count++ })
	position.JumpTo(100)
	if count != 1 {
		t.Errorf("jumpTo notified %d times, want 1", count)
	}
}

func TestScrollPositionEnsureVisibleHonorsTheAlignmentPolicy(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	children := make([]Widget, 11)
	for i := range children {
		children[i] = NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(i), Height: foundation.Some(110.0)})
	}
	ts.PumpWidget(ltr(NewSizedBox(testList(children, listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}}))))
	steps := []struct {
		item   int
		policy ScrollPositionAlignmentPolicy
		want   float64
	}{
		{0, ScrollPositionAlignmentPolicyKeepVisibleAtStart, 0},
		{1, ScrollPositionAlignmentPolicyKeepVisibleAtStart, 0},
		{1, ScrollPositionAlignmentPolicyKeepVisibleAtEnd, 0},
		{4, ScrollPositionAlignmentPolicyKeepVisibleAtStart, 0},
		{5, ScrollPositionAlignmentPolicyKeepVisibleAtStart, 0},
		{5, ScrollPositionAlignmentPolicyKeepVisibleAtEnd, 60},
		{0, ScrollPositionAlignmentPolicyKeepVisibleAtStart, 0},
	}
	for _, s := range steps {
		controller.Position().EnsureVisible(ts.RenderObject(gluttertest.ByKey(foundation.NewValueKey(s.item))), EnsureVisibleOpts{AlignmentPolicy: s.policy})
		if got := controller.Position().Pixels(); got != s.want {
			t.Fatalf("ensureVisible(item %d, %v): pixels = %v, want %v", s.item, s.policy, got, s.want)
		}
	}
}

func TestScrollPositionEnsureVisibleExplicitAndAnimated(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	children := make([]Widget, 11)
	for i := range children {
		children[i] = NewSizedBox(nil, SizedBoxOpts{Key: foundation.NewValueKey(i), Height: foundation.Some(110.0)})
	}
	ts.PumpWidget(ltr(testList(children, listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})))
	item := func(i int) rendering.RenderObject {
		return ts.RenderObject(gluttertest.ByKey(foundation.NewValueKey(i)))
	}

	// Explicit alignment with the leading edge, jumping.
	controller.Position().EnsureVisible(item(4), EnsureVisibleOpts{})
	if got := controller.Position().Pixels(); got != 4*110 {
		t.Errorf("leading alignment: pixels = %v, want %v", got, 4*110)
	}
	// The target is clamped to the extents: the last items cannot be at the top.
	controller.Position().EnsureVisible(item(8), EnsureVisibleOpts{})
	if got := controller.Position().Pixels(); got != controller.Position().MaxScrollExtent() {
		t.Errorf("clamped alignment: pixels = %v, want the maximum", got)
	}
	// Explicit alignment with the trailing edge.
	controller.Position().EnsureVisible(item(6), EnsureVisibleOpts{Alignment: 1})
	if got := controller.Position().Pixels(); got != 6*110+110-600 {
		t.Errorf("trailing alignment: pixels = %v, want %v", got, 6*110+110-600)
	}
	// An animated reveal completes at the target.
	future := controller.Position().EnsureVisible(item(10), EnsureVisibleOpts{Duration: 300 * time.Millisecond})
	ts.PumpAndSettle()
	if completed, ended := future.Completed(); !ended || !completed {
		t.Error("the future of the animated ensureVisible did not complete")
	}
	if got, want := controller.Position().Pixels(), controller.Position().MaxScrollExtent(); got != want {
		t.Errorf("animated reveal: pixels = %v, want the maximum %v", got, want)
	}
	// ScrollableEnsureVisible reaches the same position through the context.
	controller.JumpTo(0)
	ScrollableEnsureVisible(ts.Element(gluttertest.ByKey(foundation.NewValueKey(5))), EnsureVisibleOpts{})
	if got := controller.Position().Pixels(); got != 5*110 {
		t.Errorf("ScrollableEnsureVisible: pixels = %v, want %v", got, 5*110)
	}
}

func TestScrollPositionRecommendsDeferredLoadingWhileJumping(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(1, 500000))))
	position := scrollableOf(ts).Position()
	context := scrollableOf(ts).Context()
	if position.RecommendDeferredLoading(context) {
		t.Error("a position at rest recommended deferred loading")
	}
	position.JumpTo(500 * 100)
	// The jump is faster than the view's longest side per second until the frame
	// ends.
	if !position.RecommendDeferredLoading(context) {
		t.Error("a long jump did not recommend deferred loading")
	}
	ts.Pump()
	if position.RecommendDeferredLoading(context) {
		t.Error("deferred loading was still recommended after the frame")
	}
	// A smaller jump is not recommended.
	position.JumpTo(500*100 + 500)
	if position.RecommendDeferredLoading(context) {
		t.Error("a short jump recommended deferred loading")
	}
}

func TestScrollPositionStringAndMetrics(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewScrollController(ScrollControllerOpts{DebugLabel: "label"})
	defer controller.Dispose()
	ts.PumpWidget(ltr(testList(items(20, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}})))
	position := controller.Position()
	if !position.HaveDimensions() || !position.HasContentDimensions() || !position.HasViewportDimension() {
		t.Fatal("a laid out position must have its dimensions")
	}
	if position.MinScrollExtent() != 0 || position.MaxScrollExtent() != 1400 || position.ViewportDimension() != 600 {
		t.Errorf("extents %v..%v, viewport %v, want 0..1400 and 600", position.MinScrollExtent(), position.MaxScrollExtent(), position.ViewportDimension())
	}
	position.JumpTo(200)
	m := position.CopyWith()
	if m.ExtentBefore() != 200 || m.ExtentInside() != 600 || m.ExtentAfter() != 1200 || m.ExtentTotal() != 2000 {
		t.Errorf("extents before %v, inside %v, after %v, total %v", m.ExtentBefore(), m.ExtentInside(), m.ExtentAfter(), m.ExtentTotal())
	}
	if m.OutOfRange() || m.AtEdge() {
		t.Error("the middle of the content is in range and not at an edge")
	}
	if got := fmt.Sprint(m); got != "FixedScrollMetrics(200.0..[600.0]..1200.0)" {
		t.Errorf("metrics String = %q", got)
	}
	if got := position.String(); !stringsContain(got, "ScrollPositionWithSingleContext#", "label", "offset: 200.0", "range: 0.0..1400.0", "viewport: 600.0", "ScrollableState",
		"ClampingScrollPhysics -> RangeMaintainingScrollPhysics", "ScrollDirection.idle") {
		t.Errorf("position String = %q", got)
	}
}
