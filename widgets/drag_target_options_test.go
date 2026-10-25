package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// Step 27.8: the slop of a draggable, its cancelling with Escape, the moves of a target
// that repeat, and the hit test of a list under a Listener.

func TestDraggableSlop(t *testing.T) {
	tests := []struct {
		name      string
		slop      foundation.Opt[float64]
		kind      ui.PointerDeviceKind
		move      float64
		wantTap   bool
		wantDrags int
	}{
		{"mouse 3 px with a slop of 4 is a click", foundation.Some(4.0), ui.PointerDeviceKindMouse, 3, true, 0},
		{"mouse 5 px with a slop of 4 is a drag", foundation.Some(4.0), ui.PointerDeviceKindMouse, 5, false, 1},
		{"mouse 3 px with the default slop is a drag", foundation.None[float64](), ui.PointerDeviceKindMouse, 3, false, 1},
		{"touch 10 px with a slop of 12 is a tap", foundation.Some(12.0), ui.PointerDeviceKindTouch, 10, true, 0},
		{"touch 14 px with a slop of 12 is a drag", foundation.Some(12.0), ui.PointerDeviceKindTouch, 14, false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			taps, drags := 0, 0
			row := NewGestureDetector(dragBox(srcKey, "Source", 100), GestureDetectorOpts{OnTap: func() { taps++ }})
			ts.PumpWidget(dragApp(NewDraggable(NewText("Dragging"), row, DraggableOpts[int]{
				Data: foundation.Some(1), Slop: tt.slop, OnDragStarted: func() { drags++ }})))
			c := ts.Center(gluttertest.ByKey(srcKey))
			g := ts.StartGesture(c, gluttertest.GestureOpts{Kind: tt.kind, Buttons: foundation.Some(gestures.PrimaryButton)})
			ts.Pump()
			g.MoveBy(ui.Offset{Dx: tt.move})
			ts.Pump()
			g.Up()
			ts.Pump()
			if (taps == 1) != tt.wantTap || drags != tt.wantDrags {
				t.Errorf("taps = %d, drags = %d; want tap %v and %d drags", taps, drags, tt.wantTap, tt.wantDrags)
			}
		})
	}
}

func TestDraggableCancelOnEscape(t *testing.T) {
	tests := []struct {
		name       string
		option     bool
		key        services.LogicalKeyboardKey
		wantCancel bool
	}{
		{"escape cancels the drag", true, services.LogicalKeyboardKeyEscape, true},
		{"another key does not", true, services.LogicalKeyboardKeyKeyA, false},
		{"without the option escape does not", false, services.LogicalKeyboardKeyEscape, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			canceled, left, ended, completed := 0, 0, 0, 0
			ts.PumpWidget(dragApp(
				NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
					Data: foundation.Some(1), CancelOnEscape: tt.option,
					OnDraggableCanceled: func(gestures.Velocity, ui.Offset) { canceled++ },
					OnDragEnd:           func(DraggableDetails) { ended++ }}),
				intTarget(tgtKey, DragTargetOpts[int]{OnLeave: func(foundation.Opt[int]) { left++ },
					OnAcceptWithDetails: func(DragTargetDetails[int]) { completed++ }}),
			))
			g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
			ts.Pump()
			g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
			ts.Pump()
			if !textShown(ts, "Dragging") {
				t.Fatal("no drag is going on")
			}
			ts.SendKeyDownEvent(tt.key)
			ts.Pump()
			if got := !textShown(ts, "Dragging"); got != tt.wantCancel {
				t.Errorf("feedback removed = %v, want %v", got, tt.wantCancel)
			}
			if (canceled == 1) != tt.wantCancel || (left == 1) != tt.wantCancel || (ended == 1) != tt.wantCancel {
				t.Errorf("canceled=%d left=%d ended=%d, want cancelled = %v", canceled, left, ended, tt.wantCancel)
			}
			// The pointer is still down: moving it or lifting it after the cancel does nothing
			// more, and a drop on the target is not an accept.
			g.MoveBy(ui.Offset{Dx: 3})
			ts.Pump()
			g.Up()
			ts.Pump()
			if tt.wantCancel && (completed != 0 || canceled != 1 || ended != 1) {
				t.Errorf("after the release: completed=%d canceled=%d ended=%d", completed, canceled, ended)
			}
			if !tt.wantCancel && (completed != 1 || textShown(ts, "Dragging")) {
				t.Errorf("a drag that went on must drop on the target: completed=%d", completed)
			}
		})
	}
}

func TestDraggableEscapeHandlerIsRemoved(t *testing.T) {
	ts := gluttertest.New(t)
	canceled := 0
	ts.PumpWidget(dragApp(NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{
		Data: foundation.Some(1), CancelOnEscape: true, OnDraggableCanceled: func(gestures.Velocity, ui.Offset) { canceled++ }})))
	g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
	ts.Pump()
	g.Up()
	ts.Pump()
	// The drag ended with the release: the handler is gone, so Escape is not handled by it.
	if handled := ts.SendKeyDownEvent(services.LogicalKeyboardKeyEscape); handled {
		t.Error("the Escape key was handled after the drag ended")
	}
	if canceled != 1 {
		t.Errorf("canceled = %d, want the one of the drop with no target", canceled)
	}
}

func TestDragTargetRepeatMove(t *testing.T) {
	tests := []struct {
		name       string
		repeat     bool
		overTarget bool
		wantMoves  int // calls of OnMove in five frames with a resting pointer
		wantFrames bool
	}{
		{"over a target with RepeatMove: once per frame", true, true, 5, true},
		{"over a target without it: none", false, true, 0, false},
		{"not over the target: none, and no frame", true, false, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			moves := 0
			ts.PumpWidget(dragApp(
				NewDraggable(NewText("Dragging"), dragBox(srcKey, "Source", 100), DraggableOpts[int]{Data: foundation.Some(1)}),
				intTarget(tgtKey, DragTargetOpts[int]{RepeatMove: tt.repeat, OnMove: func(DragTargetDetails[int]) { moves++ }}),
			))
			g := ts.StartGesture(ts.Center(gluttertest.ByKey(srcKey)), gluttertest.GestureOpts{Pointer: 7})
			ts.Pump()
			if tt.overTarget {
				g.MoveTo(ts.Center(gluttertest.ByKey(tgtKey)))
			} else {
				g.MoveBy(ui.Offset{Dx: 5})
			}
			ts.Pump()
			moves = 0
			sched := ts.Binding.Scheduler()
			for range 5 {
				ts.Pump(16 * time.Millisecond)
			}
			if moves != tt.wantMoves {
				t.Errorf("moves in five frames = %d, want %d", moves, tt.wantMoves)
			}
			if got := sched.HasScheduledFrame() || sched.TransientCallbackCount() > 0; got != tt.wantFrames {
				t.Errorf("a frame is waiting = %v, want %v", got, tt.wantFrames)
			}
			// Leaving the target (or dropping) stops the frames.
			g.Up()
			ts.Pump()
			ts.Pump(16 * time.Millisecond)
			if sched.TransientCallbackCount() > 0 {
				t.Error("a frame callback is left after the drop")
			}
			if ts.Binding.Scheduler().HasScheduledFrame() {
				t.Error("a frame is scheduled after the drop")
			}
		})
	}
}

// The hit test of a list: a Listener above it, with the default deferToChild, gets the press
// on a row, with or without a scrollbar (its MouseRegion) around the list, and a MouseRegion
// that is not opaque reports "not hit" to what is above it, as Flutter's does.
func TestListenerAboveAListGetsThePress(t *testing.T) {
	rows := func() []Widget {
		out := make([]Widget, 20)
		for i := range out {
			out[i] = NewSizedBox(NewColoredBox(ui.NewColor(0xFF336699), nil), SizedBoxOpts{Height: foundation.Some(40.0)})
		}
		return out
	}
	tests := []struct {
		name  string
		build func(c ScrollController) Widget
		want  int
	}{
		{"a plain list", func(c ScrollController) Widget {
			return NewListView(rows(), ListViewOpts{Controller: c})
		}, 1},
		{"a list in a scrollbar", func(c ScrollController) Widget {
			return NewRawScrollbar(NewListView(rows(), ListViewOpts{Controller: c}), RawScrollbarOpts{Controller: c})
		}, 1},
		{"a list in a mouse region", func(c ScrollController) Widget {
			return NewMouseRegion(NewListView(rows(), ListViewOpts{Controller: c}))
		}, 1},
		{"a list in a mouse region that is not opaque (as Flutter)", func(c ScrollController) Widget {
			return NewMouseRegion(NewListView(rows(), ListViewOpts{Controller: c}), MouseRegionOpts{Opaque: foundation.Some(false)})
		}, 0},
		{"a list in a non-opaque mouse region with an opaque hit test behavior", func(c ScrollController) Widget {
			return NewMouseRegion(NewListView(rows(), ListViewOpts{Controller: c}),
				MouseRegionOpts{Opaque: foundation.Some(false), HitTestBehavior: rendering.HitTestBehaviorOpaque})
		}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			c := NewScrollController()
			downs := 0
			ts.PumpWidget(ltr(NewListener(tt.build(c), ListenerOpts{OnPointerDown: func(*gestures.PointerDownEvent) { downs++ }})))
			g := ts.StartGesture(ui.Offset{Dx: 20, Dy: 60}, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.PrimaryButton)})
			g.Up()
			ts.Pump()
			if downs != tt.want {
				t.Errorf("presses at the Listener = %d, want %d", downs, tt.want)
			}
		})
	}
}
