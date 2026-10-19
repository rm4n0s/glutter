package gestures

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

type testDrag struct{}

func (testDrag) Update(DragUpdateDetails) {}
func (testDrag) End(DragEndDetails)       {}
func (testDrag) Cancel()                  {}

func TestMultiDragDelayed(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kind     ui.PointerDeviceKind
		supports []ui.PointerDeviceKind
		first    ui.Offset // the first move, before the delay expires
		second   ui.Offset // the move after the delay expired
		wantAt   []bool    // started after the first move, after the delay, after the second move
	}{
		{"moving before the delay rejects", ui.PointerDeviceKindTouch, nil, off(20, 60), off(30, 120), []bool{false, false, false}},
		{"the delay triggers", ui.PointerDeviceKindTouch, nil, off(20, 20), off(30, 70), []bool{false, true, true}},
		{"can filter on the device kind", ui.PointerDeviceKindMouse, []ui.PointerDeviceKind{ui.PointerDeviceKindTouch},
			off(20, 20), off(30, 70), []bool{false, false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newGestureTester(t)
			drag := NewDelayedMultiDragGestureRecognizer(g.binding, DelayedMultiDragGestureRecognizerOpts{
				MultiDragGestureRecognizerOpts: MultiDragGestureRecognizerOpts{SupportedDevices: tc.supports},
			})
			started := false
			drag.OnStart = func(ui.Offset) Drag { started = true; return testDrag{} }
			pointer := newTestPointer(5)
			pointer.kind = tc.kind
			drag.AddPointer(pointer.down(off(10, 10)))
			g.closeArena(5)
			if started {
				t.Fatal("started at once")
			}
			g.flush()
			if started {
				t.Fatal("started after the microtasks")
			}
			g.route(pointer.move(tc.first))
			if started != tc.wantAt[0] {
				t.Errorf("started = %v after the first move", started)
			}
			g.elapse(LongPressTimeout * 2)
			if started != tc.wantAt[1] {
				t.Errorf("started = %v after the delay, want %v", started, tc.wantAt[1])
			}
			g.route(pointer.move(tc.second))
			if started != tc.wantAt[2] {
				t.Errorf("started = %v after the second move, want %v", started, tc.wantAt[2])
			}
			drag.Dispose()
		})
	}
}

func TestMultiDragAllowedButtonsFilterDefault(t *testing.T) {
	g := newGestureTester(t)
	recognizers := []*ImmediateMultiDragGestureRecognizer{
		NewImmediateMultiDragGestureRecognizer(g.binding),
		NewImmediateMultiDragGestureRecognizer(g.binding, MultiDragGestureRecognizerOpts{AllowedButtonsFilter: nil}),
	}
	allowed := NewPointerDownEvent(PointerEventOpts{TimeStamp: 240 * 3600e9})
	rejected := NewPointerDownEvent(PointerEventOpts{TimeStamp: 240 * 3600e9, Buttons: someInt(MiddleMouseButton)})
	for i, r := range recognizers {
		if !r.IsPointerAllowed(allowed) || r.IsPointerAllowed(rejected) {
			t.Errorf("recognizer %d: primary allowed = %v, middle allowed = %v", i, r.IsPointerAllowed(allowed), r.IsPointerAllowed(rejected))
		}
	}
}
