package widgets_test

import (
	"slices"
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

// These tests are ported from packages/flutter/test/widgets/draggable_test.dart.
// Not ported: the semantics test, the tests that need Ahem's text metrics (the
// sources and targets are boxes of a size, found by key), the scaled and rotated
// WidgetsApp tests, the MouseRegion cursor test, the root overlay tests, and the
// tests of the deprecated onWillAccept and onAccept.

var (
	srcKey  = foundation.NewValueKey("source")
	tgtKey  = foundation.NewValueKey("target")
	tgt2Key = foundation.NewValueKey("target 2")
)

// dragBox is a SizedBox with a label, found by key.
func dragBox(key foundation.Key, label string, height float64) Widget {
	return NewSizedBox(NewText(label), SizedBoxOpts{Key: key, Width: foundation.Some(200.0), Height: foundation.Some(height)})
}

func dragApp(children ...Widget) Widget {
	return testWidgetsApp(testWidgetsAppOpts{home: NewColumn(children)})
}

func intTarget(key foundation.Key, opts DragTargetOpts[int]) Widget {
	return NewDragTarget(func(BuildContext, []foundation.Opt[int], []any) Widget { return dragBox(key, "Target", 100) }, opts)
}

func textShown(ts *gluttertest.Tester, text string) bool { return ts.Exists(gluttertest.ByText(text)) }

func TestDragAndDropControl(t *testing.T) {
	ts := gluttertest.New(t)
	var accepted []int
	var details []DragTargetDetails[int]
	started, moves := 0, 0
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(1), OnDragStarted: func() { started++ }}),
		intTarget(tgtKey, DragTargetOpts[int]{
			OnMove:              func(DragTargetDetails[int]) { moves++ },
			OnAcceptWithDetails: func(d DragTargetDetails[int]) { accepted = append(accepted, d.Data); details = append(details, d) },
		}),
	))
	if len(accepted) != 0 || textShown(ts, "Dragging") || started != 0 || moves != 0 {
		t.Fatal("something happened before the drag")
	}
	first := ts.Center(gluttertest.ByKey(srcKey))
	g := ts.StartGesture(first, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	if !textShown(ts, "Dragging") || started != 1 || moves != 0 || len(accepted) != 0 {
		t.Fatalf("after the start: dragging=%v started=%d moves=%d accepted=%v", textShown(ts, "Dragging"), started, moves, accepted)
	}
	second := ts.Center(gluttertest.ByKey(tgtKey))
	g.MoveTo(second)
	ts.Pump()
	if moves != 1 || len(accepted) != 0 || !textShown(ts, "Dragging") {
		t.Fatalf("after the move: moves=%d accepted=%v", moves, accepted)
	}
	g.Up()
	ts.Pump()
	if !slices.Equal(accepted, []int{1}) || textShown(ts, "Dragging") || started != 1 || moves != 1 {
		t.Fatalf("after the drop: accepted=%v dragging=%v", accepted, textShown(ts, "Dragging"))
	}
	// The offset is where the feedback is: the pointer, less where it took the child.
	if want := second.Sub(first.Sub(ts.TopLeft(gluttertest.ByKey(srcKey)))); details[0].Offset != want {
		t.Errorf("offset = %v, want %v", details[0].Offset, want)
	}
}

func TestDragAndDropOnLeaveAndOnMove(t *testing.T) {
	ts := gluttertest.New(t)
	left := map[string]int{}
	moved := map[string]int{}
	target := func(key foundation.Key, name string) Widget {
		return NewDragTarget(func(BuildContext, []foundation.Opt[int], []any) Widget { return dragBox(key, name, 100) }, DragTargetOpts[int]{
			OnLeave: func(d foundation.Opt[int]) {
				if v, ok := d.Get(); ok {
					left[name] += v
				}
			},
			OnMove: func(d DragTargetDetails[int]) { moved[name] += d.Data },
		})
	}
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{Data: foundation.Some(1)}),
		target(tgtKey, "Target 1"), target(tgt2Key, "Target 2"),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	one, two := ts.Center(gluttertest.ByKey(tgtKey)), ts.Center(gluttertest.ByKey(tgt2Key))
	step := func(to ui.Offset, wantLeft, wantMoved [2]int) {
		t.Helper()
		g.MoveTo(to)
		ts.Pump()
		gotLeft := [2]int{left["Target 1"], left["Target 2"]}
		gotMoved := [2]int{moved["Target 1"], moved["Target 2"]}
		if gotLeft != wantLeft || gotMoved != wantMoved {
			t.Fatalf("moving to %v: left %v moved %v, want left %v moved %v", to, gotLeft, gotMoved, wantLeft, wantMoved)
		}
	}
	step(one, [2]int{0, 0}, [2]int{1, 0})
	step(two, [2]int{1, 0}, [2]int{1, 1})
	step(one, [2]int{1, 1}, [2]int{2, 1})
	g.Up()
	ts.Pump()
	if left["Target 1"] != 1 || left["Target 2"] != 1 {
		t.Errorf("after the drop: left = %v", left)
	}
}

func TestDragAndDropMoveIsNotCalledWithoutData(t *testing.T) {
	ts := gluttertest.New(t)
	called := false
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{}),
		intTarget(tgtKey, DragTargetOpts[int]{OnMove: func(DragTargetDetails[int]) { called = true }}),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	g.Up()
	ts.Pump()
	if called {
		t.Error("OnMove was called for a drag with no data")
	}
}

func TestDragAndDropAcceptIsNotCalledWithoutData(t *testing.T) {
	ts := gluttertest.New(t)
	accepted := false
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{}),
		intTarget(tgtKey, DragTargetOpts[int]{OnAcceptWithDetails: func(DragTargetDetails[int]) { accepted = true }}),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	g.Up()
	ts.Pump()
	if accepted {
		t.Error("OnAcceptWithDetails was called for a drag with no data")
	}
}

func TestLongPressDraggable(t *testing.T) {
	tests := []struct {
		name      string
		hold      time.Duration
		delay     foundation.Opt[time.Duration]
		wantDrag  bool
		wantAccep bool
	}{
		{"a short press does not drag", 100 * time.Millisecond, foundation.None[time.Duration](), false, false},
		{"a long press drags", gestures.LongPressTimeout + 50*time.Millisecond, foundation.None[time.Duration](), true, true},
		{"a custom delay, short", 400 * time.Millisecond, foundation.Some(time.Second), false, false},
		{"a custom delay, long", 1100 * time.Millisecond, foundation.Some(time.Second), true, true},
		{"a delay shorter than the default", 150 * time.Millisecond, foundation.Some(100 * time.Millisecond), true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var accepted []int
			started := 0
			ts.PumpWidget(dragApp(
				NewLongPressDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), LongPressDraggableOpts[int]{
					DraggableOpts: DraggableOpts[int]{Data: foundation.Some(1), OnDragStarted: func() { started++ }}, Delay: tt.delay}),
				intTarget(tgtKey, DragTargetOpts[int]{OnAcceptWithDetails: func(d DragTargetDetails[int]) { accepted = append(accepted, d.Data) }}),
			))
			g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
			ts.Pump(tt.hold)
			if textShown(ts, "Dragging") != tt.wantDrag || (started == 1) != tt.wantDrag {
				t.Fatalf("dragging = %v, started = %d, want drag %v", textShown(ts, "Dragging"), started, tt.wantDrag)
			}
			g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
			ts.Pump()
			g.Up()
			ts.Pump()
			if (len(accepted) == 1) != tt.wantAccep {
				t.Errorf("accepted = %v, want accepted %v", accepted, tt.wantAccep)
			}
		})
	}
}

func TestDraggableAxes(t *testing.T) {
	tests := []struct {
		name         string
		axis         foundation.Opt[painting.Axis]
		wantFeedback ui.Offset // where the feedback moved by a drag of (+30, +40), relative to its start
		wantUpdates  int
	}{
		{"any direction", foundation.None[painting.Axis](), ui.Offset{Dx: 30, Dy: 40}, 1},
		{"horizontal", foundation.Some(painting.AxisHorizontal), ui.Offset{Dx: 30}, 1},
		{"vertical", foundation.Some(painting.AxisVertical), ui.Offset{Dy: 40}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			feedbackKey := foundation.NewValueKey("feedback")
			updates := 0
			ts.PumpWidget(dragApp(
				NewDraggable(dragBox(feedbackKey, "Dragging", 50), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
					Data: foundation.Some(1), Axis: tt.axis, OnDragUpdate: func(gestures.DragUpdateDetails) { updates++ }}),
			))
			start := ts.Center(gluttertest.ByKey(srcKey))
			g := ts.StartGesture(start, gluttertest.GestureOpts{Pointer: 7})
			ts.Pump()
			before := ts.TopLeft(gluttertest.ByKey(feedbackKey))
			g.MoveTo(start.Add(ui.Offset{Dx: 30, Dy: 40}))
			ts.Pump()
			if got := ts.TopLeft(gluttertest.ByKey(feedbackKey)).Sub(before); got != tt.wantFeedback {
				t.Errorf("the feedback moved by %v, want %v", got, tt.wantFeedback)
			}
			if updates != tt.wantUpdates {
				t.Errorf("OnDragUpdate was called %d times, want %d", updates, tt.wantUpdates)
			}
			g.Up()
			ts.Pump()
		})
	}
}

func TestDraggableUpdateIsOnlyCalledAlongTheAxis(t *testing.T) {
	ts := gluttertest.New(t)
	updates := 0
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(1), Axis: foundation.Some(painting.AxisVertical), OnDragUpdate: func(gestures.DragUpdateDetails) { updates++ }}),
	))
	start := ts.Center(gluttertest.ByKey(srcKey))
	g := ts.StartGesture(start, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(start.Add(ui.Offset{Dx: 50}))
	ts.Pump()
	if updates != 0 {
		t.Errorf("a horizontal move of a vertical draggable called OnDragUpdate %d times", updates)
	}
	g.MoveTo(start.Add(ui.Offset{Dx: 50, Dy: 20}))
	ts.Pump()
	if updates != 1 {
		t.Errorf("a vertical move called OnDragUpdate %d times, want 1", updates)
	}
	g.Up()
	ts.Pump()
}

func TestDraggableEndCallbacks(t *testing.T) {
	tests := []struct {
		name       string
		accepts    bool
		onTarget   bool
		wantCancel int
		wantEnd    int
		wantDone   int
	}{
		{"dropped on an accepting target", true, true, 0, 1, 1},
		{"dropped on a target that does not accept", false, true, 1, 1, 0},
		{"dropped where there is no target", true, false, 1, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			canceled, ended, done := 0, 0, 0
			var endDetails DraggableDetails
			ts.PumpWidget(dragApp(
				NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
					Data:                foundation.Some(1),
					OnDraggableCanceled: func(gestures.Velocity, ui.Offset) { canceled++ },
					OnDragEnd:           func(d DraggableDetails) { ended++; endDetails = d },
					OnDragCompleted:     func() { done++ },
				}),
				intTarget(tgtKey, DragTargetOpts[int]{OnWillAcceptWithDetails: func(DragTargetDetails[int]) bool { return tt.accepts }}),
			))
			g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
			ts.Pump()
			if tt.onTarget {
				g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
			} else {
				g.MoveTo(ui.Offset{Dx: 700, Dy: 500})
			}
			ts.Pump()
			g.Up()
			ts.Pump()
			if canceled != tt.wantCancel || ended != tt.wantEnd || done != tt.wantDone {
				t.Errorf("canceled %d ended %d completed %d, want %d %d %d", canceled, ended, done, tt.wantCancel, tt.wantEnd, tt.wantDone)
			}
			if endDetails.WasAccepted != (tt.wantDone == 1) {
				t.Errorf("WasAccepted = %v", endDetails.WasAccepted)
			}
		})
	}
}

func TestDragAndDropPassesThroughTargetsThatDoNotAccept(t *testing.T) {
	ts := gluttertest.New(t)
	var accepted []string
	// Two targets, one on top of the other: the one above refuses the data.
	over := NewDragTarget(func(BuildContext, []foundation.Opt[int], []any) Widget { return dragBox(tgtKey, "Over", 100) }, DragTargetOpts[int]{
		OnWillAcceptWithDetails: func(DragTargetDetails[int]) bool { return false },
		OnAcceptWithDetails:     func(DragTargetDetails[int]) { accepted = append(accepted, "over") },
	})
	under := NewDragTarget(func(BuildContext, []foundation.Opt[int], []any) Widget { return over }, DragTargetOpts[int]{
		OnAcceptWithDetails: func(DragTargetDetails[int]) { accepted = append(accepted, "under") },
	})
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{Data: foundation.Some(1)}),
		under,
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	g.Up()
	ts.Pump()
	if !slices.Equal(accepted, []string{"under"}) {
		t.Errorf("accepted by %v, want only the target below", accepted)
	}
}

func TestDragTargetBuilderGetsCandidateAndRejectedData(t *testing.T) {
	ts := gluttertest.New(t)
	var candidates [][]foundation.Opt[int]
	var rejected [][]any
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{Data: foundation.Some(7)}),
		NewDragTarget(func(_ BuildContext, c []foundation.Opt[int], r []any) Widget {
			candidates, rejected = append(candidates, c), append(rejected, r)
			return dragBox(tgtKey, "Target", 100)
		}, DragTargetOpts[int]{OnWillAcceptWithDetails: func(d DragTargetDetails[int]) bool { return d.Data == 7 }}),
		NewDragTarget(func(_ BuildContext, c []foundation.Opt[int], r []any) Widget {
			candidates, rejected = append(candidates, c), append(rejected, r)
			return dragBox(tgt2Key, "Target", 100)
		}, DragTargetOpts[int]{OnWillAcceptWithDetails: func(DragTargetDetails[int]) bool { return false }}),
	))
	candidates, rejected = nil, nil
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	if last := candidates[len(candidates)-1]; len(last) != 1 || last[0].Value() != 7 {
		t.Errorf("candidate data over the accepting target = %v", last)
	}
	candidates, rejected = nil, nil
	g.MoveTo(ts.Center(gluttertest.ByKey(tgt2Key)))
	ts.Pump()
	var gotRejected []any
	for _, r := range rejected {
		if len(r) > 0 {
			gotRejected = r
		}
	}
	if len(gotRejected) != 1 || gotRejected[0] != 7 {
		t.Errorf("rejected data over the refusing target = %v", gotRejected)
	}
	g.Up()
	ts.Pump()
}

func TestDragTargetOfObjectAcceptsAnInt(t *testing.T) {
	ts := gluttertest.New(t)
	var got []any
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{Data: foundation.Some(3)}),
		NewDragTarget(func(BuildContext, []foundation.Opt[any], []any) Widget { return dragBox(tgtKey, "Target", 100) },
			DragTargetOpts[any]{OnAcceptWithDetails: func(d DragTargetDetails[any]) { got = append(got, d.Data) }}),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	g.Up()
	ts.Pump()
	if len(got) != 1 || got[0] != 3 {
		t.Errorf("accepted = %v, want [3]", got)
	}
}

func TestDragTargetOfAnotherTypeDoesNotSeeTheDrag(t *testing.T) {
	ts := gluttertest.New(t)
	entered := false
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{Data: foundation.Some(3)}),
		NewDragTarget(func(BuildContext, []foundation.Opt[string], []any) Widget { return dragBox(tgtKey, "Target", 100) },
			DragTargetOpts[string]{OnMove: func(DragTargetDetails[string]) { entered = true }}),
	))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	g.Up()
	ts.Pump()
	if entered {
		t.Error("a DragTarget[string] was told of a drag of an int")
	}
}

func TestDraggableMaxSimultaneousDrags(t *testing.T) {
	ts := gluttertest.New(t)
	started := 0
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(1), MaxSimultaneousDrags: foundation.Some(1), OnDragStarted: func() { started++ }}),
	))
	c := ts.Center(gluttertest.ByKey(srcKey))
	first := ts.StartGesture(c, gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	second := ts.StartGesture(c, gluttertest.GestureOpts{Pointer: 8})
	ts.Pump()
	if started != 1 {
		t.Errorf("%d drags started with MaxSimultaneousDrags 1", started)
	}
	first.Up()
	ts.Pump()
	second.Up()
	ts.Pump()
	// Once the first is over, a new drag may start.
	third := ts.StartGesture(c, gluttertest.GestureOpts{Pointer: 9})
	ts.Pump()
	if started != 2 {
		t.Errorf("%d drags started after the first ended, want 2", started)
	}
	third.Up()
	ts.Pump()
}

func TestDraggableChildWhenDragging(t *testing.T) {
	ts := gluttertest.New(t)
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(1), ChildWhenDragging: dragBox(tgt2Key, "Empty", 100)}),
	))
	if !textShown(ts, "Source") || textShown(ts, "Empty") {
		t.Fatal("before the drag the child is shown")
	}
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	if textShown(ts, "Source") || !textShown(ts, "Empty") || !textShown(ts, "Dragging") {
		t.Error("during the drag the child is replaced and the feedback shown")
	}
	g.Up()
	ts.Pump()
	if !textShown(ts, "Source") || textShown(ts, "Empty") {
		t.Error("after the drag the child is back")
	}
}

func TestDraggableRemovedDuringTheDragFinishesIt(t *testing.T) {
	ts := gluttertest.New(t)
	var accepted []int
	ended := 0
	app := func(withDraggable bool) Widget {
		children := []Widget{intTarget(tgtKey, DragTargetOpts[int]{Key: foundation.NewValueKey("kept"), OnAcceptWithDetails: func(d DragTargetDetails[int]) { accepted = append(accepted, d.Data) }})}
		if withDraggable {
			children = append([]Widget{NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
				Data: foundation.Some(1), OnDragEnd: func(DraggableDetails) { ended++ }})}, children...)
		}
		return dragApp(children...)
	}
	ts.PumpWidget(app(true))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	ts.PumpWidget(app(false))
	g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
	ts.Pump()
	g.Up()
	ts.Pump()
	if !slices.Equal(accepted, []int{1}) {
		t.Errorf("accepted = %v, want the drag to finish after the draggable is gone", accepted)
	}
	if ended != 0 {
		t.Errorf("OnDragEnd was called %d times for a removed widget", ended)
	}
}

func TestDraggablePlaysNiceWithOnTap(t *testing.T) {
	ts := gluttertest.New(t)
	taps := 0
	ts.PumpWidget(dragApp(
		NewGestureDetector(NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{Data: foundation.Some(1)}),
			GestureDetectorOpts{OnTap: func() { taps++ }}),
	))
	ts.Tap(gluttertest.ByKey(srcKey))
	ts.Pump()
	if taps != 1 {
		t.Errorf("taps = %d, want 1", taps)
	}
}

func TestDragAnchorStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy DragAnchorStrategy
		wantAt   func(start ui.Offset, ts *gluttertest.Tester) ui.Offset // top left of the feedback relative to the pointer
	}{
		{"child", nil, func(start ui.Offset, ts *gluttertest.Tester) ui.Offset {
			return ts.TopLeft(gluttertest.ByKey(srcKey)).Sub(start)
		}},
		{"pointer", PointerDragAnchorStrategy, func(ui.Offset, *gluttertest.Tester) ui.Offset { return ui.Offset{} }},
		{"custom", func(Widget, BuildContext, ui.Offset) ui.Offset { return ui.Offset{Dx: 5, Dy: 6} },
			func(ui.Offset, *gluttertest.Tester) ui.Offset { return ui.Offset{Dx: -5, Dy: -6} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			fkey := foundation.NewValueKey("feedback")
			ts.PumpWidget(dragApp(
				NewDraggable(dragBox(fkey, "Dragging", 20), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
					Data: foundation.Some(1), DragAnchorStrategy: tt.strategy}),
			))
			start := ts.Center(gluttertest.ByKey(srcKey))
			want := tt.wantAt(start, ts)
			g := ts.StartGesture(start, gluttertest.GestureOpts{Pointer: 7})
			ts.Pump()
			if got := ts.TopLeft(gluttertest.ByKey(fkey)).Sub(start); got != want {
				t.Errorf("the feedback is at %v from the pointer, want %v", got, want)
			}
			g.Up()
			ts.Pump()
		})
	}
}

func TestDraggableAllowedButtonsFilter(t *testing.T) {
	ts := gluttertest.New(t)
	started := 0
	ts.PumpWidget(dragApp(
		NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
			Data: foundation.Some(1), OnDragStarted: func() { started++ },
			AllowedButtonsFilter: func(b int) bool { return b == gestures.SecondaryButton }}),
	))
	c := ts.Center(gluttertest.ByKey(srcKey))
	g := ts.StartGesture(c, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.PrimaryButton), Pointer: 7})
	ts.Pump()
	g.Up()
	ts.Pump()
	if started != 0 {
		t.Fatal("the primary button started a drag that only allows the secondary one")
	}
	g = ts.StartGesture(c, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.SecondaryButton), Pointer: 8})
	ts.Pump()
	if started != 1 {
		t.Error("the secondary button did not start the drag")
	}
	g.Up()
	ts.Pump()
}

func TestDraggableHitTestBehavior(t *testing.T) {
	tests := []struct {
		name     string
		behavior rendering.HitTestBehavior
		want     int
	}{
		{"defer to child: an empty child is not hit", rendering.HitTestBehaviorUnspecified, 0},
		{"opaque: the area is hit", rendering.HitTestBehaviorOpaque, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			started := 0
			ts.PumpWidget(dragApp(
				NewSizedBox(NewDraggable(NewText("Dragging"), NewSizedBox(nil), DraggableOpts[int]{
					Data: foundation.Some(1), HitTestBehavior: tt.behavior, OnDragStarted: func() { started++ }}),
					SizedBoxOpts{Key: srcKey, Width: foundation.Some(100.0), Height: foundation.Some(100.0)}),
			))
			g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
			ts.Pump()
			g.Up()
			ts.Pump()
			if started != tt.want {
				t.Errorf("drags started = %d, want %d", started, tt.want)
			}
		})
	}
}

func TestDraggableNegativeMaxSimultaneousDragsPanics(t *testing.T) {
	skipUnlessDebug(t)
	requireTag(t, "DraggableMaxSimultaneousDragsNegative", func() {
		NewDraggable(NewText("f"), NewText("c"), DraggableOpts[int]{MaxSimultaneousDrags: foundation.Some(-1)})
	})
}

// Ported from LongPressDraggable.createRecognizer of widgets/drag_target.dart: the drag of a
// LongPressDraggable starts with a selection click, unless hapticFeedbackOnStart is off.
func TestLongPressDraggableHapticFeedbackOnStart(t *testing.T) {
	tests := []struct {
		name string
		hold time.Duration
		off  bool
		want []ui.HapticFeedbackKind
	}{
		{"the drag starts", gestures.LongPressTimeout + 50*time.Millisecond, false, []ui.HapticFeedbackKind{ui.HapticFeedbackSelectionClick}},
		{"the haptic feedback is off", gestures.LongPressTimeout + 50*time.Millisecond, true, nil},
		{"a short press does not start a drag", 100 * time.Millisecond, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			var got []ui.HapticFeedbackKind
			ts.Binding.HapticFeedback().OnPerform = func(k ui.HapticFeedbackKind) { got = append(got, k) }
			ts.PumpWidget(dragApp(
				NewLongPressDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), LongPressDraggableOpts[int]{
					DraggableOpts: DraggableOpts[int]{Data: foundation.Some(1)}, NoHapticFeedbackOnStart: tt.off}),
			))
			g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
			ts.Pump(tt.hold)
			g.Up()
			ts.Pump()
			if !slices.Equal(got, tt.want) {
				t.Errorf("haptic feedback = %v, want %v", got, tt.want)
			}
		})
	}
}
