package widgets_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/scrollable_of_test.dart (with a
// Scrollable of one sliver for the SingleChildScrollView of step 9.3).

// scrollPositionListener is ScrollPositionListener of the Dart test.
type scrollPositionListener struct {
	StatefulWidget
	child Widget
	log   func(string)
}

func (w *scrollPositionListener) CreateState() State { return &scrollPositionListenerState{} }

type scrollPositionListenerState struct {
	StateOf[*scrollPositionListener]
	position ScrollPosition
	handle   foundation.ListenerHandle
}

func (s *scrollPositionListenerState) detach() {
	if s.position != nil {
		s.position.RemoveListener(s.handle)
	}
}

func (s *scrollPositionListenerState) DidChangeDependencies() {
	s.detach()
	s.position = nil
	if scrollable := ScrollableMaybeOf(s.Context()); scrollable != nil {
		s.position = scrollable.Position()
		s.handle = s.position.AddListener(func() { s.Widget().log(fmt.Sprintf("listener %.1f", s.position.Pixels())) })
	}
	pixels := "<nil>"
	if s.position != nil {
		pixels = fmt.Sprintf("%.1f", s.position.Pixels())
	}
	s.Widget().log("didChangeDependencies " + pixels)
}

func (s *scrollPositionListenerState) Dispose() { s.detach() }

func (s *scrollPositionListenerState) Build(BuildContext) Widget { return s.Widget().child }

func TestScrollableOfDependentRebuildsWhenThePositionChanges(t *testing.T) {
	ts := gluttertest.New(t)
	controller := NewScrollController()
	defer controller.Dispose()
	var logValue string
	// Changing the physics causes the position of the controller to be rebuilt.
	frame := func(physics ScrollPhysics) Widget {
		return ltr(testList([]Widget{&scrollPositionListener{NewStatefulWidgetBase(), NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(400.0)}),
			func(s string) { logValue = s }}}, listOpts{ScrollableOpts: ScrollableOpts{Controller: controller, Physics: physics}}))
	}
	steps := []struct {
		name string
		do   func()
		want string
	}{
		{"first frame", func() { ts.PumpWidget(frame(nil)) }, "didChangeDependencies 0.0"},
		{"jump", func() { controller.JumpTo(100) }, "listener 100.0"},
		{"clamping physics", func() { ts.PumpWidget(frame(NewClampingScrollPhysics(nil))) }, "didChangeDependencies 100.0"},
		{"jump again", func() { controller.JumpTo(200) }, "listener 200.0"},
		{"and again", func() { controller.JumpTo(300) }, "listener 300.0"},
		{"bouncing physics", func() { ts.PumpWidget(frame(NewBouncingScrollPhysics())) }, "didChangeDependencies 300.0"},
		{"last jump", func() { controller.JumpTo(400) }, "listener 400.0"},
	}
	for _, s := range steps {
		s.do()
		if logValue != s.want {
			t.Fatalf("%s: log = %q, want %q", s.name, logValue, s.want)
		}
	}
}

func TestScrollableOfIsPossibleUsingScrollNotificationContext(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t)
	var notification ScrollNotification
	ts.PumpWidget(ltr(NewNotificationListener[ScrollNotification](tall(1200), func(n ScrollNotification) bool {
		notification = n
		return false
	})))
	g := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	ts.Pump(time.Second)
	if got := ScrollableOf(notification.Context()); got != scrollableOf(ts) {
		t.Errorf("ScrollableOf(notification.Context()) = %v, want the scrollable", got)
	}
	g.Up() // finish the gesture to release resources
	ts.PumpAndSettle()
}

// deferringScrollController is TestScrollController of the Dart test.
type deferringScrollController struct {
	ScrollControllerBase
	deferLoading bool
}

func newDeferringScrollController(deferLoading bool) *deferringScrollController {
	c := &deferringScrollController{deferLoading: deferLoading}
	InitScrollControllerBase(&c.ScrollControllerBase, c, ScrollControllerOpts{})
	return c
}

func (c *deferringScrollController) CreateScrollPosition(physics ScrollPhysics, context ScrollContext, oldPosition ScrollPosition) ScrollPosition {
	p := &deferringScrollPosition{deferLoading: c.deferLoading}
	InitScrollPositionBase(&p.ScrollPositionBase, p, physics, context, ScrollPositionOpts{OldPosition: oldPosition, KeepScrollOffset: foundation.Some(true)})
	if !p.HasPixels() {
		p.CorrectPixels(0)
	}
	if p.Activity() == nil {
		p.GoIdle()
	}
	return p
}

// deferringScrollPosition is TestScrollPosition of the Dart test.
type deferringScrollPosition struct {
	ScrollPositionWithSingleContext
	deferLoading bool
}

func (p *deferringScrollPosition) RecommendDeferredLoading(BuildContext) bool { return p.deferLoading }

func TestScrollableStaticMethodsCanTargetASpecificAxis(t *testing.T) {
	ts := gluttertest.New(t)
	horizontal, vertical := newDeferringScrollController(true), newDeferringScrollController(false)
	defer horizontal.Dispose()
	defer vertical.Dispose()
	var foundAxisDirection painting.AxisDirection
	var foundRecommendation bool
	inner := NewScrollable(func(_ BuildContext, p rendering.ViewportOffset) Widget {
		return NewViewport(p, []Widget{newSliverBox(NewBuilder(func(bc BuildContext) Widget {
			axis := foundation.Some(painting.AxisHorizontal)
			foundAxisDirection = ScrollableOf(bc, ScrollableOfOpts{Axis: axis}).AxisDirection()
			foundRecommendation = ScrollableRecommendDeferredLoadingForContext(bc, ScrollableOfOpts{Axis: axis})
			return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1200.0), Height: foundation.Some(1200.0)})
		}))}, ViewportOpts{})
	}, ScrollableOpts{Controller: vertical})
	ts.PumpWidget(ltr(NewScrollable(func(_ BuildContext, p rendering.ViewportOffset) Widget {
		// A viewport needs a bounded cross axis, which a SingleChildScrollView does not.
		return NewViewport(p, []Widget{newSliverBox(NewSizedBox(inner, SizedBoxOpts{Width: foundation.Some(1200.0)}))}, ViewportOpts{AxisDirection: painting.AxisDirectionRight})
	}, ScrollableOpts{AxisDirection: painting.AxisDirectionRight, Controller: horizontal})))
	ts.PumpAndSettle()
	if foundAxisDirection != painting.AxisDirectionRight || !foundRecommendation {
		t.Errorf("axis direction %v, recommendation %v, want right and true", foundAxisDirection, foundRecommendation)
	}
}

// dependenciesCounter counts how often its dependencies changed (TestScrollable
// and TestChild of the Dart test).
type dependenciesCounter struct {
	StatefulWidget
	child  Widget
	onDeps func(bc BuildContext)
}

func (w *dependenciesCounter) CreateState() State { return &dependenciesCounterState{} }

type dependenciesCounterState struct {
	StateOf[*dependenciesCounter]
	changed int
}

func (s *dependenciesCounterState) DidChangeDependencies() {
	s.changed++
	if s.Widget().onDeps != nil {
		s.Widget().onDeps(s.Context())
	}
}

func (s *dependenciesCounterState) Build(bc BuildContext) Widget {
	if s.Widget().child != nil {
		return s.Widget().child
	}
	return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1000.0), Height: foundation.Some(1000.0)})
}

func TestAxisTargetingScrollablesEstablishTheCorrectDependencies(t *testing.T) {
	ts := gluttertest.New(t)
	parentKey := NewGlobalKey[*dependenciesCounterState]()
	childKey := NewGlobalKey[*dependenciesCounterState]()
	frame := func(controller ScrollController) Widget {
		child := &dependenciesCounter{NewStatefulWidgetBase(WidgetOpts{Key: childKey}), nil, func(bc BuildContext) {
			ScrollableOf(bc, ScrollableOfOpts{Axis: foundation.Some(painting.AxisHorizontal)})
		}}
		parent := &dependenciesCounter{NewStatefulWidgetBase(WidgetOpts{Key: parentKey}), child, nil}
		return ltr(NewScrollable(func(_ BuildContext, p rendering.ViewportOffset) Widget {
			return NewViewport(p, []Widget{newSliverBox(parent)}, ViewportOpts{AxisDirection: painting.AxisDirectionRight})
		}, ScrollableOpts{AxisDirection: painting.AxisDirectionRight, Controller: controller}))
	}
	ts.PumpWidget(frame(nil))
	ts.PumpAndSettle()
	if changed(parentKey) != 1 || changed(childKey) != 1 {
		t.Fatalf("dependencies changed %d and %d times, want 1 and 1", changed(parentKey), changed(childKey))
	}

	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(frame(controller)) // adds a controller to the horizontal scroll view
	ts.PumpAndSettle()
	if changed(parentKey) != 1 || changed(childKey) != 2 {
		t.Errorf("dependencies changed %d and %d times, want 1 and 2", changed(parentKey), changed(childKey))
	}
}

func changed(key *GlobalKey[*dependenciesCounterState]) int {
	state, _ := key.CurrentState()
	return state.changed
}
