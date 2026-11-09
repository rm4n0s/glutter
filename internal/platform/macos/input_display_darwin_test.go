//go:build darwin && !ios && glutter_display

package macos

import (
	"math"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// inputWindow opens a window of 300 x 200 points and waits for its first frame.
func inputWindow(s *session, h *recorder) *window {
	w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
	return w
}

func changes(ds []ui.PointerData) []ui.PointerChange {
	out := make([]ui.PointerChange, len(ds))
	for i, d := range ds {
		out[i] = d.Change
	}
	return out
}

func TestRealMouseEventsMakePointerData(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takePointers()
		s.onUI(func() { w.p.runDevPointer(DevEvent{Kind: "tap", X: 50, Y: 40}) })
		d := h.takePointers()
		want := []ui.PointerChange{ui.PointerChangeAdd, ui.PointerChangeHover, ui.PointerChangeDown, ui.PointerChangeUp}
		got := changes(d)
		if len(got) != len(want) {
			t.Fatalf("changes = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("changes = %v, want %v", got, want)
				break
			}
		}
		if len(d) == 4 {
			// The place is in physical pixels from the top left of the view.
			if math.Abs(d[2].PhysicalX-50*w.scale) > 0.5 || math.Abs(d[2].PhysicalY-40*w.scale) > 0.5 {
				t.Errorf("the click is at %v, %v; want %v, %v", d[2].PhysicalX, d[2].PhysicalY, 50*w.scale, 40*w.scale)
			}
			if d[2].Buttons != buttonPrimary || d[3].Buttons != 0 || d[2].Kind != ui.PointerDeviceKindMouse {
				t.Errorf("down = %+v, up = %+v", d[2], d[3])
			}
			if d[2].ViewID != w.id {
				t.Errorf("view = %v", d[2].ViewID)
			}
		}
		// Closing the window with the mouse in it removes the pointer.
		s.onUI(w.Close)
		if d := h.takePointers(); len(d) != 1 || d[0].Change != ui.PointerChangeRemove {
			t.Errorf("close: %v", changes(d))
		}
	})
}

func TestControlClickIsTheSecondaryButton(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takePointers()
		s.onUI(func() {
			loc := w.windowPoint(20, 20)
			mk := func(typ uint64, flags uint64) objc.ID {
				return class("NSEvent").Send(sel("mouseEventWithType:location:modifierFlags:timestamp:windowNumber:context:eventNumber:clickCount:pressure:"),
					typ, loc, flags, w.p.s.mediaTime(), send[int64](w.ns, "windowNumber"), objc.ID(0), int64(0), int64(1), float32(1))
			}
			w.view.Send(sel("mouseDown:"), mk(eventTypeLeftMouseDown, modControl))
			w.view.Send(sel("mouseUp:"), mk(eventTypeLeftMouseUp, 0)) // the release has no Control any more
		})
		d := h.takePointers()
		var down, up ui.PointerData
		for _, x := range d {
			switch x.Change {
			case ui.PointerChangeDown:
				down = x
			case ui.PointerChangeUp:
				up = x
			}
		}
		if down.Buttons != buttonSecondary {
			t.Errorf("a Control-click presses %d, want the secondary button", down.Buttons)
		}
		if up.Change != ui.PointerChangeUp {
			t.Errorf("no up: %v", changes(d))
		}
		s.onUI(w.Close)
	})
}

func TestNotchedWheelMakesAScrollSignal(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takePointers()
		s.onUI(func() { w.p.runDevPointer(DevEvent{Kind: "wheel", X: 100, Y: 100, DX: 0, DY: 80}) })
		var sig *ui.PointerData
		for _, d := range h.takePointers() {
			if d.SignalKind == ui.PointerSignalKindScroll {
				d := d
				sig = &d
			}
		}
		if sig == nil {
			t.Fatal("no scroll signal")
		}
		// 80 logical pixels are two lines of 40 points; each point is `scale` physical pixels.
		if want := 80 * w.scale; math.Abs(sig.ScrollDeltaY-want) > 1e-6 || sig.ScrollDeltaX != 0 {
			t.Errorf("scroll delta = %v, %v; want 0, %v", sig.ScrollDeltaX, sig.ScrollDeltaY, want)
		}
		s.onUI(w.Close)
	})
}

func TestTrackpadPanMakesAPanZoomGesture(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		h.takePointers()
		events, err := ParseDevEvents("pan:100,100,0,64,48ms")
		if err != nil {
			t.Fatal(err)
		}
		s.onUI(func() {
			for _, ev := range events {
				w.p.runDevPointer(ev) // the waits between the steps are not needed here
			}
		})
		var pz []ui.PointerData
		for _, d := range h.takePointers() {
			if d.Kind == ui.PointerDeviceKindTrackpad && d.Change != ui.PointerChangeAdd {
				pz = append(pz, d)
			}
		}
		if len(pz) < 3 || pz[0].Change != ui.PointerChangePanZoomStart || pz[len(pz)-1].Change != ui.PointerChangePanZoomEnd {
			t.Fatalf("the gesture is %v", changes(pz))
		}
		var updates int
		for _, d := range pz[1 : len(pz)-1] {
			if d.Change != ui.PointerChangePanZoomUpdate {
				t.Errorf("a %v inside the gesture", d.Change)
			}
			updates++
		}
		last := pz[len(pz)-2]
		if math.Abs(last.PanY-64*w.scale) > 1e-6 || last.PanX != 0 {
			t.Errorf("the pan is %v, %v; want 0, %v", last.PanX, last.PanY, 64*w.scale)
		}
		if last.Scale != 1 {
			t.Errorf("scale = %v", last.Scale)
		}
		if updates != 3 {
			t.Errorf("%d updates, want 3 (48 ms in steps of 16)", updates)
		}
		s.onUI(w.Close)
	})
}

func TestSetCursorChangesTheCursorOfTheApp(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		current := func() objc.ID { return class("NSCursor").Send(sel("currentCursor")) }
		named := func(n string) objc.ID { return class("NSCursor").Send(sel(n)) }
		s.onUI(func() {
			tests := []struct {
				c    platform.MouseCursor
				want string
			}{
				{platform.MouseCursorText, "IBeamCursor"},
				{platform.MouseCursorClick, "pointingHandCursor"},
				{platform.MouseCursorGrab, "openHandCursor"},
				{platform.MouseCursorResizeLeftRight, "resizeLeftRightCursor"},
				{platform.MouseCursorWait, "arrowCursor"}, // Flutter has no entry: the arrow
				{platform.MouseCursorBasic, "arrowCursor"},
			}
			for _, tt := range tests {
				w.SetCursor(tt.c)
				if got, want := current(), named(tt.want); got != want {
					t.Errorf("%v: the current cursor is not %s", tt.c, tt.want)
				}
			}
			w.SetCursor(platform.MouseCursorNone)
			if current() == named("arrowCursor") {
				t.Error("the cursor of none is the arrow")
			}
			if size := send[nsSize](send[objc.ID](current(), "image"), "size"); size.Width != 1 || size.Height != 1 {
				t.Errorf("the blank cursor is %vx%v", size.Width, size.Height)
			}
			w.SetCursor(platform.MouseCursorBasic)
		})
		s.onUI(w.Close)
	})
}
