package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Ported from packages/flutter/test/widgets/animated_switcher_test.dart.

func colorBox(key foundation.Key, color uint32) Widget {
	return NewContainer(nil, ContainerOpts{Key: key, Color: foundation.Some(ui.NewColor(color))})
}

func fadeOpacities(ts *gluttertest.Tester) []float64 {
	var out []float64
	for _, w := range ts.Widgets(gluttertest.ByType[*FadeTransition]()) {
		out = append(out, w.(*FadeTransition).Opacity().Value())
	}
	return out
}

func TestAnimatedSwitcherFadesInANewChild(t *testing.T) {
	ts := gluttertest.New(t)
	one, two, three := foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()
	d := 100 * time.Millisecond
	ts.PumpWidget(NewAnimatedSwitcher(d, colorBox(one, 0x00000000)))
	if got := fadeOpacities(ts); len(got) != 1 || got[0] != 1 {
		t.Fatalf("opacities = %v, want [1]", got)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, colorBox(two, 0xff000000)))
	ts.Pump(50 * time.Millisecond)
	if got := fadeOpacities(ts); len(got) != 2 || got[0] != 0.5 {
		t.Fatalf("opacities = %v, want two with the first at 0.5", got)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, colorBox(three, 0xffff0000)))
	ts.Pump(10 * time.Millisecond)
	got := fadeOpacities(ts)
	want := []float64{0.4, 0.4, 0.1}
	if len(got) != 3 {
		t.Fatalf("opacities = %v, want three", got)
	}
	for i := range want {
		if !closeEnough(got[i], want[i], 0.01) {
			t.Errorf("opacity %d = %v, want %v", i, got[i], want[i])
		}
	}
	ts.PumpAndSettle()
}

func TestAnimatedSwitcherCanHandleBackToBackChanges(t *testing.T) {
	ts := gluttertest.New(t)
	k1, k2, k3 := foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()
	d := 100 * time.Millisecond
	counts := func() [3]int {
		return [3]int{ts.Count(gluttertest.ByKey(k1)), ts.Count(gluttertest.ByKey(k2)), ts.Count(gluttertest.ByKey(k3))}
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, NewContainer(nil, ContainerOpts{Key: k1})))
	if got := counts(); got != [3]int{1, 0, 0} {
		t.Errorf("counts = %v", got)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, NewContainer(nil, ContainerOpts{Key: k2})))
	if got := counts(); got != [3]int{1, 1, 0} {
		t.Errorf("counts = %v", got)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, NewContainer(nil, ContainerOpts{Key: k3})))
	if got := counts(); got != [3]int{1, 0, 1} {
		t.Errorf("counts = %v", got)
	}
}

func TestAnimatedSwitcherDoesntTransitionANewChildOfTheSameType(t *testing.T) {
	ts := gluttertest.New(t)
	d := 100 * time.Millisecond
	ts.PumpWidget(NewAnimatedSwitcher(d, colorBox(nil, 0x00000000)))
	if got := fadeOpacities(ts); len(got) != 1 || got[0] != 1 {
		t.Fatalf("opacities = %v", got)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, colorBox(nil, 0xff000000)))
	ts.Pump(50 * time.Millisecond)
	if got := fadeOpacities(ts); len(got) != 1 || got[0] != 1 {
		t.Fatalf("opacities = %v, want [1]", got)
	}
	ts.PumpAndSettle()
}

func TestAnimatedSwitcherHandlesNilChildren(t *testing.T) {
	ts := gluttertest.New(t)
	d := 100 * time.Millisecond
	ts.PumpWidget(NewAnimatedSwitcher(d, nil))
	if n := ts.Count(gluttertest.ByType[*FadeTransition]()); n != 0 {
		t.Errorf("transitions = %d", n)
	}
	first := func() float64 {
		return ts.Widget(gluttertest.First(gluttertest.ByType[*FadeTransition]())).(*FadeTransition).Opacity().Value()
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, colorBox(nil, 0xff000000)))
	ts.Pump(50 * time.Millisecond)
	if got := first(); got != 0.5 {
		t.Errorf("opacity = %v, want 0.5", got)
	}
	ts.PumpAndSettle()
	ts.PumpWidget(NewAnimatedSwitcher(d, colorBox(nil, 0x00000000)))
	if n := ts.Count(gluttertest.ByType[*FadeTransition]()); n != 1 {
		t.Errorf("transitions = %d, want 1", n)
	}
	if got := first(); got != 1 {
		t.Errorf("opacity = %v, want 1", got)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, nil))
	ts.Pump(50 * time.Millisecond)
	if got := first(); got != 0.5 {
		t.Errorf("opacity = %v, want 0.5", got)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, nil))
	ts.Pump(50 * time.Millisecond)
	if got := first(); got != 0 {
		t.Errorf("opacity = %v, want 0", got)
	}
	ts.PumpAndSettle()
}

func TestAnimatedSwitcherDoesntStartAnyAnimationsAfterDispose(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewAnimatedSwitcher(100*time.Millisecond, colorBox(nil, 0xff000000)))
	ts.Pump(50 * time.Millisecond)
	ts.PumpWidget(colorBox(nil, 0xffff0000))
	if n := ts.PumpAndSettle(); n != 1 {
		t.Errorf("PumpAndSettle = %d frames, want 1", n)
	}
}

func TestAnimatedSwitcherUsesCustomLayout(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(NewAnimatedSwitcher(100*time.Millisecond, colorBox(nil, 0), AnimatedSwitcherOpts{
		LayoutBuilder: func(current Widget, previous []Widget) Widget {
			children := append([]Widget(nil), previous...)
			if current != nil {
				children = append(children, current)
			}
			return NewColumn(children)
		},
	}))
	if n := ts.Count(gluttertest.ByType[*Column]()); n != 1 {
		t.Errorf("columns = %d, want 1", n)
	}
}

func TestAnimatedSwitcherUsesCustomTransitions(t *testing.T) {
	ts := gluttertest.New(t)
	var found []Widget
	layout := func(current Widget, previous []Widget) Widget {
		found = nil
		if current != nil {
			found = append(found, current)
		}
		found = append(found, previous...)
		return NewColumn(found)
	}
	transition := func(child Widget, a animation.Animation[float64]) Widget { return NewSizeTransition(a, child) }
	ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, NewAnimatedSwitcher(100*time.Millisecond, colorBox(nil, 0),
		AnimatedSwitcherOpts{LayoutBuilder: layout, TransitionBuilder: transition})))
	if n := ts.Count(gluttertest.ByType[*Column]()); n != 1 {
		t.Errorf("columns = %d, want 1", n)
	}
	for _, c := range found {
		if _, ok := c.(*KeyedSubtree); !ok {
			t.Errorf("child is %T, want *KeyedSubtree", c)
		}
	}
	ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, NewAnimatedSwitcher(100*time.Millisecond, nil,
		AnimatedSwitcherOpts{LayoutBuilder: layout, TransitionBuilder: transition})))
	ts.Pump(50 * time.Millisecond)
	for _, c := range found {
		if _, ok := c.(*KeyedSubtree); !ok {
			t.Errorf("child is %T, want *KeyedSubtree", c)
		}
		if n := ts.Count(gluttertest.Descendant(gluttertest.ByWidget(c), gluttertest.ByType[*SizeTransition]())); n != 1 {
			t.Errorf("size transitions under a child = %d, want 1", n)
		}
	}
}

func TestAnimatedSwitcherDoesntResetTheStateOfTheChildrenInTransitions(t *testing.T) {
	ts := gluttertest.New(t)
	generation := 0
	newStateful := func(key foundation.Key) Widget {
		return newInitCounter(key, func() { generation++ })
	}
	d := 100 * time.Millisecond
	ts.PumpWidget(NewAnimatedSwitcher(d, newStateful(foundation.NewUniqueKey())))
	if generation != 1 {
		t.Errorf("generation = %d, want 1", generation)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, newStateful(foundation.NewUniqueKey())))
	ts.Pump(50 * time.Millisecond)
	if generation != 2 || ts.Count(gluttertest.ByType[*FadeTransition]()) != 2 {
		t.Errorf("generation = %d", generation)
	}
	ts.PumpWidget(NewAnimatedSwitcher(d, newStateful(foundation.NewUniqueKey())))
	ts.Pump(10 * time.Millisecond)
	if generation != 3 {
		t.Errorf("generation = %d, want 3", generation)
	}
	ts.PumpAndSettle()
	if generation != 3 {
		t.Errorf("generation after settling = %d, want 3", generation)
	}
}

// initCounter is a StatefulWidget that calls onInit in its InitState.
type initCounter struct {
	StatefulWidget
	onInit func()
}

func (w *initCounter) CreateState() State { return &initCounterState{} }

type initCounterState struct{ StateOf[*initCounter] }

func (s *initCounterState) InitState()                { s.Widget().onInit() }
func (s *initCounterState) Build(BuildContext) Widget { return NewContainer(nil) }

// newInitCounter makes a keyed stateful widget that calls onInit once for each
// state it gets.
func newInitCounter(key foundation.Key, onInit func()) Widget {
	return &initCounter{NewStatefulWidgetBase(WidgetOpts{Key: key}), onInit}
}

func TestAnimatedSwitcherUpdatesWidgetsWithoutAnimatingIfTheyAreIsomorphic(t *testing.T) {
	ts := gluttertest.New(t)
	pump := func(child Widget) {
		ts.PumpWidget(NewDirectionality(ui.TextDirectionRTL, NewAnimatedSwitcher(100*time.Millisecond, child)))
	}
	pump(NewText("1"))
	ts.Pump(10 * time.Millisecond)
	transition := ts.Widget(gluttertest.First(gluttertest.ByType[*FadeTransition]())).(*FadeTransition)
	if transition.Opacity().Value() != 1 || ts.Count(gluttertest.ByText("1")) != 1 || ts.Count(gluttertest.ByText("2")) != 0 {
		t.Fatal("the first child is not shown")
	}
	pump(NewText("2"))
	transition = ts.Widget(gluttertest.First(gluttertest.ByType[*FadeTransition]())).(*FadeTransition)
	ts.Pump(20 * time.Millisecond)
	if transition.Opacity().Value() != 1 || ts.Count(gluttertest.ByText("1")) != 0 || ts.Count(gluttertest.ByText("2")) != 1 {
		t.Error("the second child did not replace the first")
	}
}

func TestAnimatedSwitcherUpdatesPreviousChildTransitionsIfTheTransitionBuilderChanges(t *testing.T) {
	ts := gluttertest.New(t)
	one, two, three := foundation.NewUniqueKey(), foundation.NewUniqueKey(), foundation.NewUniqueKey()
	var found []Widget
	layout := func(current Widget, previous []Widget) Widget {
		found = nil
		if current != nil {
			found = append(found, current)
		}
		found = append(found, previous...)
		return NewColumn(found)
	}
	d := 100 * time.Millisecond
	pump := func(child Widget, opts AnimatedSwitcherOpts) {
		opts.LayoutBuilder = layout
		ts.PumpWidget(NewAnimatedSwitcher(d, child, opts))
	}
	pump(colorBox(one, 0xFFFF0000), AnimatedSwitcherOpts{})
	ts.Pump(10 * time.Millisecond)
	pump(colorBox(two, 0xFF00FF00), AnimatedSwitcherOpts{})
	ts.Pump(10 * time.Millisecond)
	pump(colorBox(three, 0xFF0000FF), AnimatedSwitcherOpts{})
	ts.Pump(10 * time.Millisecond)
	if len(found) != 3 {
		t.Fatalf("children = %d, want 3", len(found))
	}
	for _, c := range found {
		if _, ok := c.(*KeyedSubtree); !ok {
			t.Errorf("child is %T", c)
		}
		if n := ts.Count(gluttertest.Descendant(gluttertest.ByWidget(c), gluttertest.ByType[*FadeTransition]())); n != 1 {
			t.Errorf("fade transitions under a child = %d, want 1", n)
		}
	}
	pump(colorBox(nil, 0), AnimatedSwitcherOpts{TransitionBuilder: func(child Widget, a animation.Animation[float64]) Widget {
		return NewScaleTransition(a, child)
	}})
	ts.Pump(10 * time.Millisecond)
	if len(found) != 3 {
		t.Fatalf("children = %d, want 3", len(found))
	}
	for _, c := range found {
		if n := ts.Count(gluttertest.Descendant(gluttertest.ByWidget(c), gluttertest.ByType[*ScaleTransition]())); n != 1 {
			t.Errorf("scale transitions under a child = %d, want 1", n)
		}
	}
}

func TestAnimatedSwitcherDoesNotDuplicateAnimationsIfTheSameChildIsEnteredTwice(t *testing.T) {
	ts := gluttertest.New(t)
	pump := func(key string) {
		ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewAnimatedSwitcher(time.Second, NewText(key, TextOpts{Key: foundation.NewKey(key)}))))
	}
	pump("1")
	pump("2")
	pump("1")
	ts.Pump(time.Second)
	if n := ts.Count(gluttertest.ByText("1")); n != 1 {
		t.Errorf("texts = %d, want 1", n)
	}
}

func TestAnimatedSwitcherDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t)
	wrapShrink := func(child Widget) Widget {
		return NewDirectionality(ui.TextDirectionLTR, NewCenter(NewSizedBoxShrink(child)))
	}
	d := 500 * time.Microsecond
	for _, child := range []Widget{nil, NewText("X", TextOpts{Key: foundation.NewKey("x")}), NewText("Y", TextOpts{Key: foundation.NewKey("y")})} {
		ts.PumpWidget(wrapShrink(NewAnimatedSwitcher(d, child)))
		if size := ts.Size(gluttertest.ByType[*AnimatedSwitcher]()); size != (ui.Size{}) {
			t.Errorf("size = %v", size)
		}
	}
}
