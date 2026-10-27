package gestures

import (
	"slices"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// Tests for the microtasks of the gesture binding (step 29.17). Dart runs a
// microtask after the handler of one event and before the next, so what the
// gesture arena schedules (the default win) runs before the next pointer event
// even when several events are handled in one turn of the event loop. No
// Flutter test is ported: Flutter has no counterpart for the queue.

// microtaskFixture is a binding with a tap recognizer on every pointer that goes
// down; it logs the events and the callbacks in the order they happen.
type microtaskFixture struct {
	*bindingTester
	log []string
	tap *TapGestureRecognizer
}

func newMicrotaskFixture(t *testing.T) *microtaskFixture {
	f := &microtaskFixture{bindingTester: newBindingTester(t, 1)}
	f.tap = NewTapGestureRecognizer(f.binding)
	f.tap.OnTapDown = func(TapDownDetails) { f.log = append(f.log, "tapDown") }
	f.tap.OnTapUp = func(TapUpDetails) { f.log = append(f.log, "tapUp") }
	f.tap.OnTap = func() { f.log = append(f.log, "tap") }
	f.tap.OnTapCancel = func() { f.log = append(f.log, "tapCancel") }
	f.onEvent = func(e PointerEvent) {
		f.log = append(f.log, foundation.RuntimeTypeName(e))
		if d, ok := e.(*PointerDownEvent); ok {
			f.tap.AddPointer(d)
		}
	}
	t.Cleanup(f.tap.Dispose)
	return f
}

func pdata(change ui.PointerChange, id int, x float64) ui.PointerData {
	return ui.PointerData{Change: change, PointerIdentifier: id, PhysicalX: x, PhysicalY: 10}
}

func TestBindingMicrotasksRunBeforeTheNextPointerEvent(t *testing.T) {
	const down, move, up = ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp
	tests := []struct {
		name string
		// events are dispatched in one packet, so the binding handles them in
		// one turn; nothing posted runs in between.
		events []ui.PointerData
		// wantTurn is the log before the posted functions run, wantAfter the
		// log after they ran.
		wantTurn  []string
		wantAfter []string
	}{
		{
			// Emetos's reproduction: a pointer 0 that goes down, moves 150 pixels,
			// goes up and down again, all before the loop turns.
			"same id: press, move 150 px, release, press",
			[]ui.PointerData{pdata(down, 0, 0), pdata(move, 0, 150), pdata(up, 0, 150), pdata(down, 0, 0)},
			[]string{
				"PointerDownEvent", "tapDown", // the first arena is won by default ...
				"PointerMoveEvent", "tapCancel", // ... so the move past the slop cancels the tap ...
				"PointerUpEvent",
				"PointerDownEvent", // ... before the second press is handled
			},
			[]string{
				"PointerDownEvent", "tapDown", "PointerMoveEvent", "tapCancel", "PointerUpEvent",
				"PointerDownEvent", "tapDown",
			},
		},
		{
			"new ids: press, move 150 px, release, press",
			[]ui.PointerData{pdata(down, 1, 0), pdata(move, 1, 150), pdata(up, 1, 150), pdata(down, 2, 0)},
			[]string{
				"PointerDownEvent", "tapDown",
				"PointerMoveEvent", "tapCancel", "PointerUpEvent",
				"PointerDownEvent",
			},
			[]string{
				"PointerDownEvent", "tapDown", "PointerMoveEvent", "tapCancel", "PointerUpEvent",
				"PointerDownEvent", "tapDown",
			},
		},
		{
			"two quick taps with new ids",
			[]ui.PointerData{pdata(down, 1, 0), pdata(up, 1, 0), pdata(down, 2, 0), pdata(up, 2, 0)},
			[]string{
				"PointerDownEvent", "tapDown", "PointerUpEvent", "tapUp", "tap",
				"PointerDownEvent", "tapDown", "PointerUpEvent", "tapUp", "tap",
			},
			nil,
		},
		{
			"a lone tap wins by default on the next turn",
			[]ui.PointerData{pdata(down, 1, 0)},
			[]string{"PointerDownEvent"},
			[]string{"PointerDownEvent", "tapDown"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newMicrotaskFixture(t)
			f.d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: tc.events})
			if !slices.Equal(f.log, tc.wantTurn) {
				t.Errorf("in the turn of the events:\n got %v\nwant %v", f.log, tc.wantTurn)
			}
			f.host.flush()
			want := tc.wantAfter
			if want == nil {
				want = tc.wantTurn
			}
			if !slices.Equal(f.log, want) {
				t.Errorf("after the next turn:\n got %v\nwant %v", f.log, want)
			}
		})
	}
}

func TestBindingMicrotaskQueue(t *testing.T) {
	const down, up = ui.PointerChangeDown, ui.PointerChangeUp
	tests := []struct {
		name string
		// setup schedules microtasks before the events are dispatched.
		setup func(b *GestureBinding, log func(string))
		want  []string
	}{
		{
			"a microtask that schedules another runs both before the next event",
			func(b *GestureBinding, log func(string)) {
				b.scheduleMicrotask(func() {
					log("first")
					b.scheduleMicrotask(func() { log("nested") })
				})
				b.scheduleMicrotask(func() { log("second") })
			},
			[]string{"first", "second", "nested", "PointerDownEvent", "PointerUpEvent"},
		},
		{
			"the tasks are first in first out",
			func(b *GestureBinding, log func(string)) {
				for _, n := range []string{"a", "b", "c"} {
					b.scheduleMicrotask(func() { log(n) })
				}
			},
			[]string{"a", "b", "c", "PointerDownEvent", "PointerUpEvent"},
		},
		{
			"nothing is queued",
			func(*GestureBinding, func(string)) {},
			[]string{"PointerDownEvent", "PointerUpEvent"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bt := newBindingTester(t, 1)
			var log []string
			bt.onEvent = func(e PointerEvent) { log = append(log, foundation.RuntimeTypeName(e)) }
			tc.setup(bt.binding, func(s string) { log = append(log, s) })
			bt.d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{pdata(down, 1, 0), pdata(up, 1, 0)}})
			if !slices.Equal(log, tc.want) {
				t.Errorf("order = %v, want %v", log, tc.want)
			}
			bt.host.flush()
			if len(bt.binding.microtasks) != 0 {
				t.Errorf("%d microtasks are left", len(bt.binding.microtasks))
			}
		})
	}
}

func TestBindingMicrotaskRunsOnTheNextTurnWithoutAnEvent(t *testing.T) {
	bt := newBindingTester(t, 1)
	ran := 0
	bt.binding.scheduleMicrotask(func() { ran++ })
	bt.binding.scheduleMicrotask(func() { ran++ })
	if ran != 0 {
		t.Fatalf("a microtask ran in the turn that scheduled it")
	}
	if len(bt.host.posted) != 1 {
		t.Fatalf("%d functions were posted, want one for the queue", len(bt.host.posted))
	}
	bt.host.flush()
	if ran != 2 {
		t.Errorf("%d microtasks ran on the next turn, want 2", ran)
	}
	// A function that is posted is still a post, and it runs after the
	// microtasks that were queued before it.
	var order []string
	bt.binding.scheduleMicrotask(func() { order = append(order, "microtask") })
	bt.binding.Post(func() { order = append(order, "post") })
	bt.host.flush()
	if !slices.Equal(order, []string{"microtask", "post"}) {
		t.Errorf("order = %v", order)
	}
}

func TestBindingMicrotaskPanicIsReported(t *testing.T) {
	bt := newBindingTester(t, 1)
	var errs reports
	bt.binding.SetErrorReporter(errs.report)
	ran := false
	bt.binding.scheduleMicrotask(func() { panic("bad microtask") })
	bt.binding.scheduleMicrotask(func() { ran = true })
	bt.host.flush()
	if !ran {
		t.Error("the microtask after the failed one did not run")
	}
	appErr := errs.checkOne(t, "while running a microtask of the gesture binding")
	if appErr.Metadata["panic"] != "bad microtask" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
}
