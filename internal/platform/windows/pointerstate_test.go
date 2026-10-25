package windows

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func mouseAt(x, y float64, buttons int) pointerSample {
	return pointerSample{kind: kMouse, device: mouseDevice, x: x, y: y, buttons: buttons}
}

func TestPointerTracker(t *testing.T) {
	tests := []struct {
		name string
		run  func(tr *pointerTracker)
		want []ev
	}{
		{"the first move adds the pointer, then it hovers", func(tr *pointerTracker) {
			tr.move(0, mouseAt(10, 20, 0))
			tr.move(0, mouseAt(15, 25, 0))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 10, y: 20},
			{change: cHover, kind: kMouse, x: 10, y: 20},
			{change: cHover, kind: kMouse, x: 15, y: 25, dx: 5, dy: 5},
		}},
		{"a click is a down and an up", func(tr *pointerTracker) {
			tr.move(0, mouseAt(10, 20, 0))
			tr.down(0, mouseAt(10, 20, buttonPrimary))
			tr.up(0, mouseAt(10, 20, buttonPrimary))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 10, y: 20},
			{change: cHover, kind: kMouse, x: 10, y: 20},
			{change: cDown, kind: kMouse, x: 10, y: 20, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 10, y: 20},
		}},
		{"a button down with no move before adds the pointer first", func(tr *pointerTracker) {
			tr.down(0, mouseAt(3, 4, buttonSecondary))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 3, y: 4},
			{change: cDown, kind: kMouse, x: 3, y: 4, buttons: buttonSecondary},
		}},
		{"a down with no button does nothing", func(tr *pointerTracker) {
			tr.down(0, mouseAt(3, 4, 0))
		}, nil},
		{"two buttons: the first is the down, the second a move, the last up the up", func(tr *pointerTracker) {
			tr.move(0, mouseAt(1, 1, 0))
			tr.down(0, mouseAt(1, 1, buttonPrimary))
			tr.down(0, mouseAt(1, 1, buttonSecondary))
			tr.up(0, mouseAt(1, 1, buttonPrimary))
			tr.up(0, mouseAt(1, 1, buttonSecondary))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cHover, kind: kMouse, x: 1, y: 1},
			{change: cDown, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary | buttonSecondary},
			{change: cMove, kind: kMouse, x: 1, y: 1, buttons: buttonSecondary},
			{change: cUp, kind: kMouse, x: 1, y: 1},
		}},
		{"the side buttons are the back and forward bits", func(tr *pointerTracker) {
			tr.down(0, mouseAt(0, 0, buttonBack))
			tr.down(0, mouseAt(0, 0, buttonForward))
		}, []ev{
			{change: cAdd, kind: kMouse},
			{change: cDown, kind: kMouse, buttons: buttonBack},
			{change: cMove, kind: kMouse, buttons: buttonBack | buttonForward},
		}},
		{"a drag moves with the buttons down", func(tr *pointerTracker) {
			tr.down(0, mouseAt(5, 5, buttonPrimary))
			tr.move(0, mouseAt(-20, 9, buttonPrimary))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 5, y: 5},
			{change: cDown, kind: kMouse, x: 5, y: 5, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: -20, y: 9, dx: -25, dy: 4, buttons: buttonPrimary},
		}},
		{"leaving with a button down lifts it first, synthesized", func(tr *pointerTracker) {
			tr.down(0, mouseAt(5, 5, buttonPrimary))
			tr.leave(0, kMouse, mouseDevice, 7, 8)
		}, []ev{
			{change: cAdd, kind: kMouse, x: 5, y: 5},
			{change: cDown, kind: kMouse, x: 5, y: 5, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 5, y: 5, synth: true},
			{change: cRemove, kind: kMouse, x: 7, y: 8},
		}},
		{"leaving a hover is a remove", func(tr *pointerTracker) {
			tr.move(0, mouseAt(2, 2, 0))
			tr.leave(0, kMouse, mouseDevice, 2, 2)
		}, []ev{
			{change: cAdd, kind: kMouse, x: 2, y: 2},
			{change: cHover, kind: kMouse, x: 2, y: 2},
			{change: cRemove, kind: kMouse, x: 2, y: 2},
		}},
		{"leaving a pointer that was never added is ignored", func(tr *pointerTracker) {
			tr.leave(0, kMouse, mouseDevice, 2, 2)
		}, nil},
		{"the pointer is added again after it was removed", func(tr *pointerTracker) {
			tr.move(0, mouseAt(2, 2, 0))
			tr.leave(0, kMouse, mouseDevice, 2, 2)
			tr.move(0, mouseAt(4, 4, 0))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 2, y: 2},
			{change: cHover, kind: kMouse, x: 2, y: 2},
			{change: cRemove, kind: kMouse, x: 2, y: 2},
			{change: cAdd, kind: kMouse, x: 4, y: 4},
			{change: cHover, kind: kMouse, x: 4, y: 4},
		}},
		{"a move with buttons for a pointer that is not down is its down", func(tr *pointerTracker) {
			tr.move(0, pointerSample{kind: kStylus, device: 2000, x: 15, y: 20, buttons: buttonPrimary})
		}, []ev{
			{change: cAdd, kind: kStylus, device: 2000, x: 15, y: 20},
			{change: cDown, kind: kStylus, device: 2000, x: 15, y: 20, buttons: buttonPrimary},
		}},
		{"a move with no buttons ends a contact whose up was lost, once", func(tr *pointerTracker) {
			tr.down(0, mouseAt(1, 1, buttonPrimary))
			tr.move(0, mouseAt(2, 2, 0))
			tr.move(0, mouseAt(3, 3, 0))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cDown, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 2, y: 2, dx: 1, dy: 1},
			{change: cHover, kind: kMouse, x: 3, y: 3, dx: 1, dy: 1},
		}},
		{"an up of a button that is not down sends a hover and no up", func(tr *pointerTracker) {
			tr.move(0, mouseAt(1, 1, 0))
			tr.up(0, mouseAt(1, 1, buttonPrimary))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cHover, kind: kMouse, x: 1, y: 1},
			{change: cHover, kind: kMouse, x: 1, y: 1},
		}},
		{"an up with no buttons while none is down does nothing", func(tr *pointerTracker) {
			tr.up(0, mouseAt(1, 1, 0))
		}, nil},
		{"the buttons are released without a message: a synthesized up, the pointer stays", func(tr *pointerTracker) {
			tr.down(0, mouseAt(1, 1, buttonPrimary))
			tr.releaseButtons(kMouse, mouseDevice)
			tr.move(0, mouseAt(2, 2, 0))
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cDown, kind: kMouse, x: 1, y: 1, buttons: buttonPrimary},
			{change: cUp, kind: kMouse, x: 1, y: 1, synth: true},
			{change: cHover, kind: kMouse, x: 2, y: 2, dx: 1, dy: 1},
		}},
		{"releasing the buttons of a pointer that is up does nothing", func(tr *pointerTracker) {
			tr.move(0, mouseAt(1, 1, 0))
			tr.releaseButtons(kMouse, mouseDevice)
		}, []ev{
			{change: cAdd, kind: kMouse, x: 1, y: 1},
			{change: cHover, kind: kMouse, x: 1, y: 1},
		}},
		{"a pointer that moves to another view is removed from the first", func(tr *pointerTracker) {
			tr.move(1, mouseAt(5, 5, 0))
			tr.move(2, mouseAt(6, 6, 0))
			tr.leave(1, kMouse, mouseDevice, 5, 5) // the first window hears of it late
		}, []ev{
			{change: cAdd, kind: kMouse, x: 5, y: 5, view: 1},
			{change: cHover, kind: kMouse, x: 5, y: 5, view: 1},
			{change: cRemove, kind: kMouse, x: 5, y: 5, view: 1},
			{change: cAdd, kind: kMouse, x: 6, y: 6, view: 2},
			{change: cHover, kind: kMouse, x: 6, y: 6, view: 2},
		}},
		{"cancel of a pointer that is down is a cancel and a remove", func(tr *pointerTracker) {
			tr.down(0, pointerSample{kind: kTouch, device: 1000, x: 3, y: 3, buttons: buttonPrimary})
			tr.cancel(0, kTouch, 1000)
		}, []ev{
			{change: cAdd, kind: kTouch, device: 1000, x: 3, y: 3},
			{change: cDown, kind: kTouch, device: 1000, x: 3, y: 3, buttons: buttonPrimary},
			{change: cCancel, kind: kTouch, device: 1000, x: 3, y: 3},
			{change: cRemove, kind: kTouch, device: 1000, x: 3, y: 3},
		}},
		{"cancel of a pointer that hovers is a remove", func(tr *pointerTracker) {
			tr.enter(0, pointerSample{kind: kStylus, device: 2000, x: 3, y: 3})
			tr.cancel(0, kStylus, 2000)
		}, []ev{
			{change: cAdd, kind: kStylus, device: 2000, x: 3, y: 3},
			{change: cRemove, kind: kStylus, device: 2000, x: 3, y: 3},
		}},
		{"enter twice adds once", func(tr *pointerTracker) {
			tr.enter(0, pointerSample{kind: kStylus, device: 2000, x: 3, y: 3})
			tr.enter(0, pointerSample{kind: kStylus, device: 2000, x: 4, y: 4})
		}, []ev{{change: cAdd, kind: kStylus, device: 2000, x: 3, y: 3}}},
		{"a mouse and a pen with the same device number are two pointers", func(tr *pointerTracker) {
			tr.move(0, pointerSample{kind: kMouse, device: 5, x: 1, y: 1})
			tr.move(0, pointerSample{kind: kStylus, device: 5, x: 2, y: 2})
		}, []ev{
			{change: cAdd, kind: kMouse, device: 5, x: 1, y: 1},
			{change: cHover, kind: kMouse, device: 5, x: 1, y: 1},
			{change: cAdd, kind: kStylus, device: 5, x: 2, y: 2},
			{change: cHover, kind: kStylus, device: 5, x: 2, y: 2},
		}},
		{"a scroll adds the pointer and is a hover signal", func(tr *pointerTracker) {
			tr.scroll(0, kMouse, mouseDevice, 30, 40, 0, 100)
		}, []ev{
			{change: cAdd, kind: kMouse, x: 30, y: 40},
			{change: cHover, kind: kMouse, x: 30, y: 40, signal: true, scroll: [2]float64{0, 100}},
		}},
		{"a scroll with a button down is a move signal", func(tr *pointerTracker) {
			tr.down(0, mouseAt(30, 40, buttonPrimary))
			tr.scroll(0, kMouse, mouseDevice, 30, 40, -50, 0)
		}, []ev{
			{change: cAdd, kind: kMouse, x: 30, y: 40},
			{change: cDown, kind: kMouse, x: 30, y: 40, buttons: buttonPrimary},
			{change: cMove, kind: kMouse, x: 30, y: 40, buttons: buttonPrimary, signal: true, scroll: [2]float64{-50, 0}},
		}},
		{"cancelling a view cancels its pointers in order", func(tr *pointerTracker) {
			tr.down(3, pointerSample{kind: kTouch, device: 1001, x: 2, y: 2, buttons: buttonPrimary})
			tr.down(3, pointerSample{kind: kTouch, device: 1000, x: 1, y: 1, buttons: buttonPrimary})
			tr.move(4, mouseAt(9, 9, 0))
			tr.cancelView(3)
		}, []ev{
			{change: cAdd, kind: kTouch, device: 1001, x: 2, y: 2, view: 3},
			{change: cDown, kind: kTouch, device: 1001, x: 2, y: 2, buttons: buttonPrimary, view: 3},
			{change: cAdd, kind: kTouch, device: 1000, x: 1, y: 1, view: 3},
			{change: cDown, kind: kTouch, device: 1000, x: 1, y: 1, buttons: buttonPrimary, view: 3},
			{change: cAdd, kind: kMouse, x: 9, y: 9, view: 4},
			{change: cHover, kind: kMouse, x: 9, y: 9, view: 4},
			{change: cCancel, kind: kTouch, device: 1000, x: 1, y: 1, view: 3},
			{change: cRemove, kind: kTouch, device: 1000, x: 1, y: 1, view: 3},
			{change: cCancel, kind: kTouch, device: 1001, x: 2, y: 2, view: 3},
			{change: cRemove, kind: kTouch, device: 1001, x: 2, y: 2, view: 3},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newPointerTracker(tickingClock())
			tt.run(tr)
			sameEvents(t, tr.take(), tt.want)
			if rest := tr.take(); len(rest) != 0 {
				t.Errorf("take gave %d events again", len(rest))
			}
		})
	}
}

func TestPointerTrackerIdentifiersAndTimes(t *testing.T) {
	tr := newPointerTracker(tickingClock())
	tr.down(0, mouseAt(1, 1, buttonPrimary))
	tr.up(0, mouseAt(1, 1, buttonPrimary))
	tr.down(0, mouseAt(1, 1, buttonSecondary))
	tr.up(0, mouseAt(1, 1, buttonSecondary))
	got := tr.take()
	// add, down, up, down, up
	if len(got) != 5 {
		t.Fatalf("%d events", len(got))
	}
	if got[0].PointerIdentifier != 0 {
		t.Errorf("the add has identifier %d", got[0].PointerIdentifier)
	}
	if got[1].PointerIdentifier != 1 || got[2].PointerIdentifier != 1 {
		t.Errorf("the first contact has identifiers %d, %d; want 1", got[1].PointerIdentifier, got[2].PointerIdentifier)
	}
	if got[3].PointerIdentifier != 2 || got[4].PointerIdentifier != 2 {
		t.Errorf("the second contact has identifiers %d, %d; want 2", got[3].PointerIdentifier, got[4].PointerIdentifier)
	}
	for i := 1; i < len(got); i++ {
		if got[i].TimeStamp <= got[i-1].TimeStamp {
			t.Errorf("event %d is not later than the one before", i)
		}
	}
}

func TestPhase(t *testing.T) {
	// SetEventPhaseFromCursorButtonState: the phase from the buttons and the "down" state.
	tests := []struct {
		name    string
		buttons int
		down    bool
		want    ui.PointerChange
	}{
		{"no buttons, not down", 0, false, cHover},
		{"no buttons, down", 0, true, cUp},
		{"buttons, not down", buttonPrimary, false, cDown},
		{"buttons, down", buttonPrimary, true, cMove},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := phase(&pointerState{buttons: tt.buttons, down: tt.down}); got != tt.want {
				t.Errorf("phase = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPointerTrackerMouseButtons(t *testing.T) {
	tr := newPointerTracker(tickingClock())
	if got := tr.mouseButtons(); got != 0 {
		t.Errorf("buttons of a mouse nobody has seen = %d", got)
	}
	steps := []struct {
		name string
		do   func()
		want int
	}{
		{"hovering", func() { tr.move(0, mouseAt(1, 1, 0)) }, 0},
		{"the primary button goes down", func() { tr.down(0, mouseAt(1, 1, buttonPrimary)) }, buttonPrimary},
		{"the secondary one too", func() { tr.down(0, mouseAt(1, 1, buttonSecondary)) }, buttonPrimary | buttonSecondary},
		{"the primary one goes up", func() { tr.up(0, mouseAt(1, 1, buttonPrimary)) }, buttonSecondary},
		{"the last one goes up", func() { tr.up(0, mouseAt(1, 1, buttonSecondary)) }, 0},
	}
	for _, s := range steps {
		s.do()
		if got := tr.mouseButtons(); got != s.want {
			t.Errorf("%s: buttons = %d, want %d", s.name, got, s.want)
		}
	}
}

// The reading of a drop is limited to 5 seconds, as on the other platforms (SPEC §8.11, §25.10).
func TestDropReadBudgetIsFiveSeconds(t *testing.T) {
	if dropReadBudget.Seconds() != 5 {
		t.Errorf("dropReadBudget = %v, want 5s", dropReadBudget)
	}
}
