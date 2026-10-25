//go:build windows && glutter_wine

package windows

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests send input to a window under Wine on a headless X server, the way a user's devices
// do (SendInput) and the way the scripted events do (SendMessage), and read what the handler gets
// (SPEC §25.6, §25.13). WM_POINTER messages of touch and pen cannot be made under Wine: their
// conversion is tested over recorded values (pointer_test.go) and the devices are a manual check.

// OnPointer records the batches the platform sends.
func (r *rec) OnPointer(data []ui.PointerData) {
	r.batches = append(r.batches, append([]ui.PointerData(nil), data...))
	r.pointers = append(r.pointers, data...)
}

// pointerSteps runs the steps one after the other on the UI thread, gap apart, after the first
// window was shown, and ends the run after the last one.
func pointerSteps(t *testing.T, p *Platform, r *rec, gap time.Duration, steps ...func()) {
	t.Helper()
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		var run func(i int)
		run = func(i int) {
			if i == len(steps) {
				cancel()
				return
			}
			steps[i]()
			after(p, gap, func() { run(i + 1) })
		}
		run(0)
	})
}

// mouseInput makes one mouse input for SendInput.
func mouseInput(flags uint32) win32.Input {
	return win32.Input{Type: win32.InputMouse, Mouse: win32.MouseInput{Flags: flags}}
}

// moveCursorTo puts the pointer at a place of the client area of the window, in physical pixels.
func moveCursorTo(t *testing.T, w *window, x, y int32) {
	t.Helper()
	pt, ok := win32.ClientToScreen(w.hwnd, win32.Point{X: x, Y: y})
	if !ok || !win32.SetCursorPos(pt.X, pt.Y) {
		t.Fatalf("the pointer could not be moved to %d,%d", x, y)
	}
}

func sendInput(t *testing.T, flags ...uint32) {
	t.Helper()
	// A button that a test leaves down would be down in the next test: the X server of the run is
	// shared. Releasing a button that is up does nothing.
	t.Cleanup(func() {
		win32.SendInput([]win32.Input{mouseInput(win32.MouseEventFLeftUp), mouseInput(win32.MouseEventFRightUp)})
	})
	inputs := make([]win32.Input, len(flags))
	for i, f := range flags {
		inputs[i] = mouseInput(f)
	}
	if n := win32.SendInput(inputs); int(n) != len(inputs) {
		t.Fatalf("SendInput inserted %d of %d inputs", n, len(inputs))
	}
}

// changes lists the changes of the recorded data of the mouse.
func (r *rec) changes() []ui.PointerChange {
	var out []ui.PointerChange
	for _, d := range r.pointers {
		out = append(out, d.Change)
	}
	return out
}

func countChange(data []ui.PointerData, c ui.PointerChange) int {
	n := 0
	for _, d := range data {
		if d.Change == c {
			n++
		}
	}
	return n
}

func TestSendInputClicksThreeTimes(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	click := func() { sendInput(t, win32.MouseEventFLeftDown, win32.MouseEventFLeftUp) }
	pointerSteps(t, p, r, 80*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 90) },
		click, click, click,
		func() {},
	)
	if got := countChange(r.pointers, ui.PointerChangeDown); got != 3 {
		t.Errorf("%d downs, want 3: %v", got, r.changes())
	}
	if got := countChange(r.pointers, ui.PointerChangeUp); got != 3 {
		t.Errorf("%d ups, want 3: %v", got, r.changes())
	}
	if got := countChange(r.pointers, ui.PointerChangeAdd); got != 1 {
		t.Errorf("%d adds, want 1: %v", got, r.changes())
	}
	for _, d := range r.pointers {
		if d.Kind != ui.PointerDeviceKindMouse || d.Device != 0 || d.ViewID != w.id {
			t.Errorf("%+v is not the mouse of the window", d)
		}
		if d.Change == ui.PointerChangeDown && (d.Buttons != buttonPrimary || d.PhysicalX != 100 || d.PhysicalY != 90) {
			t.Errorf("down = %+v: want the primary button at 100,90", d)
		}
	}
	var ids []int
	for _, d := range r.pointers {
		if d.Change == ui.PointerChangeDown {
			ids = append(ids, d.PointerIdentifier)
		}
	}
	if len(ids) != 3 || ids[0] == ids[1] || ids[1] == ids[2] {
		t.Errorf("the three clicks have the identifiers %v: each should have its own", ids)
	}
	if win32.GetCapture() == w.hwnd {
		t.Error("the capture is still taken after the last button went up")
	}
}

func TestSendInputRightButtonAndTwoButtons(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	pointerSteps(t, p, r, 80*time.Millisecond,
		func() { moveCursorTo(t, w, 50, 60) },
		func() { sendInput(t, win32.MouseEventFLeftDown) },
		func() { sendInput(t, win32.MouseEventFRightDown) },
		func() { sendInput(t, win32.MouseEventFLeftUp) },
		func() { sendInput(t, win32.MouseEventFRightUp) },
		func() {},
	)
	var seq []ev
	for _, e := range brief(r.pointers) {
		if e.change != cHover {
			seq = append(seq, ev{change: e.change, buttons: e.buttons})
		}
	}
	want := []ev{
		{change: cAdd},
		{change: cDown, buttons: buttonPrimary},
		{change: cMove, buttons: buttonPrimary | buttonSecondary},
		{change: cMove, buttons: buttonSecondary},
		{change: cUp},
	}
	if len(seq) != len(want) {
		t.Fatalf("got %+v, want %+v", seq, want)
	}
	for i := range want {
		if seq[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, seq[i], want[i])
		}
	}
}

func TestADragThatLeavesTheWindowIsReleasedOutside(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	var outsideX int32
	pointerSteps(t, p, r, 100*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 100) },
		func() { sendInput(t, win32.MouseEventFLeftDown) },
		func() {
			// Past the right edge of the window, on the screen.
			outsideX = w.clientW + 150
			moveCursorTo(t, w, outsideX, 100)
		},
		func() {
			if win32.GetCapture() != w.hwnd {
				t.Error("the window has no capture while the button is down")
			}
		},
		func() { sendInput(t, win32.MouseEventFLeftUp) },
		func() {},
	)
	downs, ups := countChange(r.pointers, ui.PointerChangeDown), countChange(r.pointers, ui.PointerChangeUp)
	if downs != 1 || ups != 1 {
		t.Fatalf("%d downs and %d ups: %v", downs, ups, r.changes())
	}
	var moved, up *ui.PointerData
	for i := range r.pointers {
		d := &r.pointers[i]
		if d.Change == ui.PointerChangeMove && d.PhysicalX >= float64(w.clientW) {
			moved = d
		}
		if d.Change == ui.PointerChangeUp {
			up = d
		}
	}
	if moved == nil || moved.Buttons != buttonPrimary {
		t.Errorf("no move outside the window with the button down: %v", r.changes())
	}
	if up == nil || up.PhysicalX != float64(outsideX) || up.Synthesized {
		t.Errorf("up = %+v (synthesized %v): want a real up where the button was released, %d; %+v", up, up != nil && up.Synthesized, outsideX, brief(r.pointers))
	}
	if win32.GetCapture() == w.hwnd {
		t.Error("the capture was not given back")
	}
	// The drag went on outside the window: the leave that Wine sent at once waited for the up.
	if last := r.pointers[len(r.pointers)-1]; last.Change != ui.PointerChangeRemove && last.Change != ui.PointerChangeUp {
		t.Errorf("the last change is %v", last.Change)
	}
	if got := countChange(r.pointers, ui.PointerChangeRemove); got > 1 {
		t.Errorf("%d removes: %v", got, r.changes())
	}
}

func TestMouseLeaveRemovesThePointer(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	pointerSteps(t, p, r, 150*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 100) },
		func() { moveCursorTo(t, w, 110, 100) },
		func() { moveCursorTo(t, w, w.clientW+100, 100) }, // out of the window
		func() {},
		func() {},
	)
	if countChange(r.pointers, ui.PointerChangeAdd) != 1 || countChange(r.pointers, ui.PointerChangeRemove) != 1 {
		t.Errorf("the pointer was not added once and removed once: %v", r.changes())
	}
	if last := r.pointers[len(r.pointers)-1]; last.Change != ui.PointerChangeRemove {
		t.Errorf("the last change is %v: want the remove", last.Change)
	}
	if w.mouse.tracking {
		t.Error("the window thinks the pointer is still in")
	}
}

func TestMouseLeaveDuringADragWaitsForTheUp(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	var removesBeforeUp int
	pointerSteps(t, p, r, 80*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 100) },
		func() { sendInput(t, win32.MouseEventFLeftDown) },
		func() { win32.SendMessage(w.hwnd, win32.WMMouseLeave, 0, 0) }, // what Wine sends when a captured pointer goes out
		func() { removesBeforeUp = countChange(r.pointers, ui.PointerChangeRemove) },
		func() { sendInput(t, win32.MouseEventFLeftUp) },
		func() {},
	)
	if removesBeforeUp != 0 {
		t.Errorf("the pointer was removed while the button was down: %v", r.changes())
	}
	// The remove follows the real up of the button. (The pointer is in the window, so the next
	// mouse message adds it again.)
	rm := -1
	for i, d := range r.pointers {
		if d.Change == ui.PointerChangeRemove {
			rm = i
		}
	}
	if rm < 1 || r.pointers[rm-1].Change != ui.PointerChangeUp || r.pointers[rm-1].Synthesized {
		t.Errorf("the remove does not follow the up of the button: %v", r.changes())
	}
}

func TestTheCaptureTakenByAnotherWindowReleasesTheButtons(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	other := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	place(t, other, 700, 400) // out of the way: a window opens a step below the one before it
	pointerSteps(t, p, r, 80*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 100) },
		func() { sendInput(t, win32.MouseEventFLeftDown) },
		func() { win32.SetCapture(other.hwnd) },
		func() { win32.ReleaseCapture() },
		func() {},
	)
	var up *ui.PointerData
	for i := range r.pointers {
		if r.pointers[i].Change == ui.PointerChangeUp {
			up = &r.pointers[i]
		}
	}
	if up == nil || !up.Synthesized || up.ViewID != w.id {
		t.Errorf("up = %+v: want a synthesized up in the window that lost the capture; %v", up, r.changes())
	}
	if countChange(r.pointers, ui.PointerChangeRemove) != 0 {
		t.Errorf("the pointer was removed: %v", r.changes())
	}
}

func TestTheWheelScrollsByTheLinesOfTheSystem(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	var lines uint32
	pointerSteps(t, p, r, 80*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 100) },
		func() {
			lines = wheelLines()
			in := mouseInput(win32.MouseEventFWheel)
			in.Mouse.MouseData = uint32(0xFFFFFFFF - 119) // -120: one notch towards the user
			if win32.SendInput([]win32.Input{in}) != 1 {
				t.Error("SendInput failed")
			}
		},
		func() {},
	)
	var sig *ui.PointerData
	for i := range r.pointers {
		if r.pointers[i].SignalKind == ui.PointerSignalKindScroll {
			sig = &r.pointers[i]
		}
	}
	if sig == nil {
		t.Fatalf("no scroll signal: %v", r.changes())
	}
	want := float64(lines) * 100 / 3
	if lines == 0 || lines == win32.WheelPageScroll {
		want = float64(w.clientH)
	}
	if sig.ScrollDeltaX != 0 || !near(sig.ScrollDeltaY, want) || sig.PhysicalX != 100 || sig.PhysicalY != 100 {
		t.Errorf("scroll = %+v: want %v down at 100,100 (%d lines)", sig, want, lines)
	}
}

func TestTheScriptedEventsAreTheMessagesOfARealMouse(t *testing.T) {
	script := "move:60,70;tap:60,70;wheel:60,70,0,300;wheel:60,70,-100,0;wait:50ms"
	p, r := newTestPlatform(t, Options{Getenv: func(k string) string {
		if k == DevEventsEnv {
			return script
		}
		return ""
	}})
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	go func() {
		<-p.devDone
		cancel()
		p.wake()
	}()
	if err := p.Run(ctx, r); err != nil {
		t.Fatal(err)
	}
	scale := w.scale()
	var signals []ui.PointerData
	for _, d := range r.pointers {
		if d.SignalKind == ui.PointerSignalKindScroll {
			signals = append(signals, d)
		}
	}
	if countChange(r.pointers, ui.PointerChangeDown) != 1 || countChange(r.pointers, ui.PointerChangeUp) != 1 {
		t.Errorf("the tap is not one down and one up: %v", r.changes())
	}
	for _, d := range r.pointers {
		if d.Change == ui.PointerChangeDown && (d.PhysicalX != float64(physical(60, scale)) || d.PhysicalY != float64(physical(70, scale))) {
			t.Errorf("the tap is at %v,%v", d.PhysicalX, d.PhysicalY)
		}
	}
	if len(signals) != 2 {
		t.Fatalf("%d scroll signals, want 2", len(signals))
	}
	// 300 logical pixels down, and 100 to the left, in physical pixels (one notch is rounded to
	// 1/120).
	if !nearTo(signals[0].ScrollDeltaY, 300*scale, 1) || signals[0].ScrollDeltaX != 0 {
		t.Errorf("first scroll = %v,%v; want 0,%v", signals[0].ScrollDeltaX, signals[0].ScrollDeltaY, 300*scale)
	}
	if !nearTo(signals[1].ScrollDeltaX, -100*scale, 1) || signals[1].ScrollDeltaY != 0 {
		t.Errorf("second scroll = %v,%v; want %v,0", signals[1].ScrollDeltaX, signals[1].ScrollDeltaY, -100*scale)
	}
	// No leave was asked for by a pointer that is not really there.
	if countChange(r.pointers, ui.PointerChangeRemove) != 0 {
		t.Errorf("the scripted pointer was removed: %v", r.changes())
	}
}

func nearTo(a, b, eps float64) bool { return a-b < eps && b-a < eps }

func TestCursorsOfTheTable(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	tests := []struct {
		cursor platform.MouseCursor
		idc    uintptr // 0: hidden
	}{
		{platform.MouseCursorText, win32.IDCIBeam},
		{platform.MouseCursorClick, win32.IDCHand},
		{platform.MouseCursorBasic, win32.IDCArrow},
		{platform.MouseCursorWait, win32.IDCWait},
		{platform.MouseCursorForbidden, win32.IDCNo},
		{platform.MouseCursorResizeLeftRight, win32.IDCSizeWE},
		{platform.MouseCursorResizeUpDown, win32.IDCSizeNS},
		{platform.MouseCursorResizeUpLeftDownRight, win32.IDCSizeNWSE},
		{platform.MouseCursorMove, win32.IDCSizeAll},
		{platform.MouseCursorNone, 0},
		{platform.MouseCursorText, win32.IDCIBeam},
	}
	var steps []func()
	steps = append(steps, func() { moveCursorTo(t, w, 100, 100) })
	for _, tt := range tests {
		steps = append(steps, func() {
			w.SetCursor(tt.cursor)
			ci, ok := win32.GetCursorInfo()
			if !ok {
				t.Fatal("GetCursorInfo failed")
			}
			if tt.idc == 0 {
				if ci.Cursor != 0 && ci.Flags&1 != 0 {
					t.Errorf("%v: the cursor %#x is shown", tt.cursor, ci.Cursor)
				}
				return
			}
			if want := win32.LoadCursor(0, tt.idc); ci.Cursor != want {
				t.Errorf("%v: the cursor is %#x, want IDC %d = %#x", tt.cursor, ci.Cursor, tt.idc, want)
			}
		})
		steps = append(steps, func() {}) // a turn of the loop
	}
	pointerSteps(t, p, r, 20*time.Millisecond, steps...)
}

func TestWMSetCursorIsAnsweredOverTheClientAreaOnly(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	pointerSteps(t, p, r, 20*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 100) },
		func() {
			w.cursor = platform.MouseCursorClick
			if got := win32.SendMessage(w.hwnd, win32.WMSetCursor, w.hwnd, uintptr(win32.HTClient)|uintptr(win32.WMMouseMove)<<16); got != 1 {
				t.Errorf("WM_SETCURSOR over the client area answered %d, want TRUE", got)
			}
			if ci, _ := win32.GetCursorInfo(); ci.Cursor != win32.LoadCursor(0, win32.IDCHand) {
				t.Errorf("the cursor is %#x after WM_SETCURSOR, want the hand", ci.Cursor)
			}
			w.cursor = platform.MouseCursorText
			win32.SendMessage(w.hwnd, win32.WMSetCursor, w.hwnd, uintptr(win32.HTClient))
			if ci, _ := win32.GetCursorInfo(); ci.Cursor != win32.LoadCursor(0, win32.IDCIBeam) {
				t.Errorf("the cursor is %#x after WM_SETCURSOR for the I-beam", ci.Cursor)
			}
			// Any other part of the window is DefWindowProc's: not answered as handled by us.
			w.cursor = platform.MouseCursorWait
			win32.SendMessage(w.hwnd, win32.WMSetCursor, w.hwnd, uintptr(win32.HTCaption))
			if ci, _ := win32.GetCursorInfo(); ci.Cursor == win32.LoadCursor(0, win32.IDCWait) {
				t.Error("WM_SETCURSOR over the caption set the cursor of the app")
			}
		},
	)
}

func TestDestroyingAWindowCancelsItsPointers(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	keep := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 100, Height: 100}})
	place(t, keep, 700, 400) // out of the way: a window opens a step below the one before it
	pointerSteps(t, p, r, 80*time.Millisecond,
		func() { moveCursorTo(t, w, 100, 100) },
		func() { sendInput(t, win32.MouseEventFLeftDown) },
		func() { w.Close() },
		func() {},
	)
	n := len(r.pointers)
	if n < 2 || r.pointers[n-1].Change != ui.PointerChangeRemove || r.pointers[n-2].Change != ui.PointerChangeCancel {
		t.Errorf("the pointer of the closed window was not cancelled and removed: %v", r.changes())
	}
}
