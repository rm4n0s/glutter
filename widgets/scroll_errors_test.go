package widgets_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// The debug-only checks of the scrolling classes (step 9.2), each with its tag.

// capture pumps a Builder and returns its context.
func capture(ts *gluttertest.Tester) BuildContext {
	var bc BuildContext
	ts.PumpWidget(ltr(NewBuilder(func(c BuildContext) Widget {
		bc = c
		return NewSizedBox(nil)
	})))
	return bc
}

func TestScrollLookupsWithoutAncestors(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	bc := capture(ts)
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"Scrollable", "ScrollableNotFound", func() { ScrollableOf(bc) }},
		{"PageStorage", "PageStorageMissing", func() { PageStorageOf(bc) }},
		{"PrimaryScrollController", "PrimaryScrollControllerMissing", func() { PrimaryScrollControllerOf(bc) }},
		{"ScrollNotificationObserver", "ScrollNotificationObserverMissing", func() { ScrollNotificationObserverOf(bc) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, tt.tag, tt.call) })
	}
	// The Maybe variants return nothing instead.
	if ScrollableMaybeOf(bc) != nil || MaybePageStorageOf(bc) != nil || PrimaryScrollControllerMaybeOf(bc) != nil || ScrollNotificationObserverMaybeOf(bc) != nil {
		t.Error("a Maybe lookup found an ancestor that is not there")
	}
}

func TestScrollableSemanticChildCountMustNotBeNegative(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "ScrollableSemanticChildCountInvalid", func() {
		NewScrollable(func(_ BuildContext, p rendering.ViewportOffset) Widget { return NewViewport(p, nil, ViewportOpts{}) },
			ScrollableOpts{SemanticChildCount: foundation.Some(-1)})
	})
}

func TestViewportCenterMustBeOneOfTheSlivers(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(3, 100))))
	position := scrollableOf(ts).Position()
	slivers := []Widget{newSliverBox(NewSizedBox(nil), WidgetOpts{Key: foundation.NewValueKey("a")})}
	requireTag(t, "ViewportCenterInvalid", func() {
		NewViewport(position, slivers, ViewportOpts{Center: foundation.NewValueKey("missing")})
	})
	slivers = append(slivers, newSliverBox(NewSizedBox(nil), WidgetOpts{Key: foundation.NewValueKey("a")}))
	requireTag(t, "ViewportCenterInvalid", func() {
		NewViewport(position, slivers, ViewportOpts{Center: foundation.NewValueKey("a")})
	})
}

func TestOverscrollNotificationNeedsAnOverscroll(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(3, 100))))
	state := scrollableOf(ts)
	for _, overscroll := range []float64{0, math.Inf(1), math.NaN()} {
		requireTag(t, "OverscrollNotificationInvalid", func() {
			NewOverscrollNotification(state.Position(), state.Context(), overscroll)
		})
	}
}

func TestScrollControllerAttachAndDetach(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	other := NewScrollController()
	defer other.Dispose()
	ts.PumpWidget(ltr(NewRow([]Widget{
		NewSizedBox(testList(items(3, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: controller}}), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
		NewSizedBox(testList(items(3, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: other}}), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
	})))
	requireTag(t, "ScrollControllerPositionAttached", func() { controller.Attach(controller.Position()) })
	requireTag(t, "ScrollControllerPositionNotAttached", func() { controller.Detach(other.Position()) })

	tracking := NewTrackingScrollController()
	defer tracking.Dispose()
	// Attached to the controller only, not tracked: the second check fails.
	tracking.ScrollControllerBase.Attach(other.Position())
	requireTag(t, "ScrollControllerPositionNotAttached", func() { tracking.Detach(other.Position()) })
	tracking.Attach(other.Position())
	tracking.ScrollControllerBase.Detach(other.Position())
	requireTag(t, "ScrollControllerPositionAttached", func() { tracking.Attach(other.Position()) })
}

func TestScrollPositionChecks(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	a, b := NewScrollController(), NewScrollController()
	defer a.Dispose()
	defer b.Dispose()
	ts.PumpWidget(ltr(NewRow([]Widget{
		NewSizedBox(testList(items(3, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: a}}), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
		NewSizedBox(testList(items(3, 100), listOpts{ScrollableOpts: ScrollableOpts{Controller: b}}), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
	})))
	p, other := a.Position(), b.Position()
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"absorb a position of another scrollable", "ScrollPositionAbsorbInvalid", func() { p.Absorb(other) }},
		{"extents in the wrong order", "ScrollPositionExtentsInvalid", func() { p.ApplyContentDimensions(10, 5) }},
		{"set pixels while idle", "ScrollPositionNotScrolling", func() { p.SetPixels(5) }},
		{"overscroll while idle", "ScrollOverscrollWhileIdle", func() { p.DidOverscrollBy(5) }},
		{"metrics notification that was not scheduled", "ScrollMetricsNotificationUnscheduled", func() { p.DidUpdateScrollMetrics() }},
		{"ensure visible of a detached object", "ScrollEnsureVisibleDetached", func() { p.EnsureVisible(rendering.NewRenderLimitedBox(nil)) }},
		{"begin an activity of another delegate", "ScrollActivityDelegateInvalid", func() {
			p.(*ScrollPositionWithSingleContext).BeginActivity(NewIdleScrollActivity(other.(*ScrollPositionWithSingleContext)))
		}},
		{"update the delegate of an activity to itself", "ScrollActivityDelegateSame", func() { p.Activity().UpdateDelegate(p.(*ScrollPositionWithSingleContext)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, tt.tag, tt.call) })
	}
}

func TestScrollPositionMustNotChangeDuringLayout(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	var call func()
	pump := func() {
		ts.PumpWidget(ltr(NewRow([]Widget{
			NewSizedBox(testList(items(3, 100)), SizedBoxOpts{Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
			NewBuilder(func(BuildContext) Widget {
				if call != nil {
					call()
				}
				return NewSizedBox(nil)
			}),
		})))
	}
	pump()
	position := scrollableOf(ts).Position()
	tests := []struct {
		name string
		call func()
	}{
		{"set pixels", func() { position.Base().SetPixels(1) }},
		{"update scroll metrics", func() { position.DidUpdateScrollMetrics() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call = tt.call
			pump()
			_ = requireErrorTag(t, takeFirst(t, ts), "ScrollPositionChangedDuringLayout")
			call = nil
			pump()
		})
	}
}

// hugeOverscrollPhysics reports more overscroll than the change.
type hugeOverscrollPhysics struct{ ScrollPhysicsBase }

func newHugeOverscrollPhysics() *hugeOverscrollPhysics {
	p := &hugeOverscrollPhysics{}
	InitScrollPhysicsBase(&p.ScrollPhysicsBase, p, nil)
	return p
}

func (p *hugeOverscrollPhysics) ApplyTo(ScrollPhysics) ScrollPhysics { return p }

func (p *hugeOverscrollPhysics) ApplyBoundaryConditions(ScrollMetrics, float64) float64 { return 1e9 }

func TestScrollPhysicsOverscrollMustNotExceedTheDelta(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(30, 100), listOpts{ScrollableOpts: ScrollableOpts{Physics: newHugeOverscrollPhysics()}})))
	position := scrollableOf(ts).Position()
	requireTag(t, "ScrollPhysicsOverscrollInvalid", func() { position.Base().SetPixels(50) })
}

func TestScrollDragControllerChecks(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(30, 100))))
	state := scrollableOf(ts)
	position := state.Position().(*ScrollPositionWithSingleContext)
	newController := func() *ScrollDragController {
		return NewScrollDragController(position, gestures.NewDragStartDetails())
	}
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"an update without a primary delta", "ScrollDragPrimaryDeltaMissing", func() { newController().Update(gestures.DragUpdateDetails{}) }},
		{"an end without a primary velocity", "ScrollDragPrimaryVelocityMissing", func() { newController().End(gestures.DragEndDetails{}) }},
		{"update the delegate to itself", "ScrollActivityDelegateSame", func() { newController().UpdateDelegate(position) }},
		{"a scroll start with update details", "ScrollDragDetailsUnexpected", func() {
			c := newController()
			activity := NewDragScrollActivity(position, c)
			position.BeginActivity(activity)
			c.Update(gestures.DragUpdateDetails{PrimaryDelta: foundation.Some(1.0)})
			activity.DispatchScrollStartNotification(position, state.Context())
		}},
		{"a scroll update with start details", "ScrollDragDetailsUnexpected", func() {
			NewDragScrollActivity(position, newController()).DispatchScrollUpdateNotification(position, state.Context(), 1)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTag(t, tt.tag, tt.call) })
	}
}

// quirkyPosition is a position whose boundary conditions and new dimensions the
// test controls.
type quirkyPosition struct {
	ScrollPositionWithSingleContext
	boundary func(value float64) float64
	newDims  func()
}

func newQuirkyPosition(ts *gluttertest.Tester) *quirkyPosition {
	state := scrollableOf(ts)
	q := &quirkyPosition{}
	InitScrollPositionBase(&q.ScrollPositionBase, q, NewClampingScrollPhysics(nil), state, ScrollPositionOpts{KeepScrollOffset: foundation.Some(false)})
	q.CorrectPixels(0)
	q.GoIdle()
	return q
}

func (q *quirkyPosition) ApplyBoundaryConditions(value float64) float64 {
	if q.boundary != nil {
		return q.boundary(value)
	}
	return q.ScrollPositionWithSingleContext.ApplyBoundaryConditions(value)
}

func (q *quirkyPosition) ApplyNewDimensions() {
	q.ScrollPositionWithSingleContext.ApplyNewDimensions()
	if q.newDims != nil {
		q.newDims()
	}
}

func TestScrollPositionBoundaryConditionsMustNotExceedTheDelta(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(30, 100))))
	q := newQuirkyPosition(ts)
	q.boundary = func(float64) float64 { return 1e9 }
	requireTag(t, "ScrollOverscrollInvalid", func() { q.Base().SetPixels(5) })
}

func TestScrollPositionDimensionsMustBeCorrectedWithCorrectForNewDimensions(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(30, 100))))
	q := newQuirkyPosition(ts)
	q.newDims = func() { q.CorrectBy(1) }
	requireTag(t, "ScrollPositionCorrectionMissing", func() { q.ApplyContentDimensions(0, 100) })
}

func TestScrollableDragStateChecks(t *testing.T) {
	skipUnlessDebug(t)
	useAndroid(t)
	ts := gluttertest.New(t)
	ts.PumpWidget(ltr(testList(items(30, 100))))
	vertical := scrollableRecognizer(t, ts)
	t.Run("a down while there is a hold", func(t *testing.T) {
		vertical.OnDown(gestures.DragDownDetails{})
		requireTag(t, "ScrollableDragStateInvalid", func() { vertical.OnDown(gestures.DragDownDetails{}) })
		vertical.OnCancel()
	})
	t.Run("a start while there is a drag", func(t *testing.T) {
		vertical.OnDown(gestures.DragDownDetails{})
		vertical.OnStart(gestures.NewDragStartDetails())
		requireTag(t, "ScrollableDragStateInvalid", func() { vertical.OnStart(gestures.NewDragStartDetails()) })
		vertical.OnCancel()
	})
}

func scrollableRecognizer(t *testing.T, ts *gluttertest.Tester) *gestures.VerticalDragGestureRecognizer {
	t.Helper()
	detector := gluttertest.StateOf[*RawGestureDetectorState](ts, gluttertest.Descendant(gluttertest.ByType[*Scrollable](), gluttertest.ByType[*RawGestureDetector]()))
	for _, r := range detector.Recognizers() {
		if v, ok := r.(*gestures.VerticalDragGestureRecognizer); ok {
			return v
		}
	}
	t.Fatal("the scrollable has no vertical drag recognizer")
	return nil
}
