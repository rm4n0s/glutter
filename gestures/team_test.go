package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// passiveGestureRecognizer is the recognizer of team_test.dart that only
// follows the pointer and lets its team decide.
type passiveGestureRecognizer struct {
	OneSequenceGestureRecognizer
	onGestureAccepted func()
}

func newPassiveGestureRecognizer(b *GestureBinding) *passiveGestureRecognizer {
	r := &passiveGestureRecognizer{}
	r.InitOneSequence(b, r, GestureRecognizerOpts{})
	return r
}

func (r *passiveGestureRecognizer) AddPointer(event *PointerDownEvent) {
	r.StartTrackingPointer(event.Pointer(), event.Transform())
}
func (r *passiveGestureRecognizer) DebugDescription() string { return "passive" }
func (r *passiveGestureRecognizer) DidStopTrackingLastPointer(int) {
	r.Resolve(GestureRejected)
}
func (r *passiveGestureRecognizer) HandleEvent(event PointerEvent) {
	switch event.(type) {
	case *PointerUpEvent, *PointerCancelEvent:
		r.StopTrackingPointer(event.Pointer())
	}
}
func (r *passiveGestureRecognizer) AcceptGesture(int) {
	if r.onGestureAccepted != nil {
		r.onGestureAccepted()
	}
}
func (r *passiveGestureRecognizer) RejectGesture(int) {}

func TestGestureArenaTeam(t *testing.T) {
	for _, tc := range []struct {
		name    string
		captain bool
		delta   ui.Offset
		want    string
	}{
		{"rejection: tap wins without movement", false, ui.Offset{}, "tap"},
		{"rejection: vertical drag wins on a vertical move", false, ui.Offset{Dy: 30}, "vertical-drag-start"},
		{"captain: tap wins without movement", true, ui.Offset{}, "tap"},
		{"captain: the captain accepts on a vertical move", true, ui.Offset{Dy: 30}, "captain accepted gesture"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			team := NewGestureArenaTeam(g.binding.GestureArena)
			horizontalDrag := NewHorizontalDragGestureRecognizer(g.binding)
			horizontalDrag.SetTeam(team)
			verticalDrag := NewVerticalDragGestureRecognizer(g.binding)
			verticalDrag.SetTeam(team)
			tap := NewTapGestureRecognizer(g.binding)
			if horizontalDrag.Team() != team || verticalDrag.Team() != team || tap.Team() != nil {
				t.Fatal("the team of the recognizers is wrong")
			}
			var captain *passiveGestureRecognizer
			var l log
			if tc.captain {
				captain = newPassiveGestureRecognizer(g.binding)
				captain.SetTeam(team)
				team.Captain = captain
				captain.onGestureAccepted = func() { l.add("captain accepted gesture") }
			}
			horizontalDrag.OnStart = func(DragStartDetails) { l.add("horizontal-drag-start") }
			verticalDrag.OnStart = func(DragStartDetails) { l.add("vertical-drag-start") }
			tap.OnTap = func() { l.add("tap") }

			origin := ui.Offset{Dx: 10, Dy: 10}
			pointer := newTestPointer(5)
			down := pointer.down(origin)
			if captain != nil {
				captain.AddPointer(down)
			}
			horizontalDrag.AddPointer(down)
			verticalDrag.AddPointer(down)
			tap.AddPointer(down)
			l.check(t)
			g.closeArena(5)
			l.check(t)
			g.route(down)
			l.check(t)
			g.route(pointer.move(origin.Add(tc.delta)))
			g.route(pointer.up())
			l.check(t, tc.want)

			horizontalDrag.Dispose()
			verticalDrag.Dispose()
			tap.Dispose()
			if captain != nil {
				captain.Dispose()
			}
		})
	}
}
