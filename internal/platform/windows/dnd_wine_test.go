//go:build windows && glutter_wine

package windows

import (
	"context"
	"slices"
	"testing"
	"time"
	"unsafe"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/ole"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests check drag and drop and the several windows of the app under Wine (SPEC §25.10, §25.4,
// §25.13). A drop target is found the way a program that drags finds it: through the property that
// RegisterDragDrop sets on the window.

func (r *rec) OnDrop(e ui.DropEvent) ui.DropActions {
	r.drops = append(r.drops, e)
	if r.dropAnswer != nil {
		return r.dropAnswer(e)
	}
	return ui.DropActionNone
}

func (r *rec) OnDragEnd(res ui.DragResult) { r.dragEnds = append(r.dragEnds, res) }

// dropTargetProp is the property that RegisterDragDrop sets on a window to find its target.
const dropTargetProp = "OleDropTargetInterface"

// registeredTarget is the IDropTarget that OLE has for the window, found through the property.
func registeredTarget(t *testing.T, w *window) win32.Object {
	t.Helper()
	ptr := win32.GetProp(w.hwnd, dropTargetProp)
	if ptr == 0 {
		t.Fatalf("the window %#x has no drop target registered (property %s)", w.hwnd, dropTargetProp)
	}
	return win32.ObjectFromPtr(ptr)
}

// screenPoint is a place of the client area of the window, as the POINTL of a drop target call: the
// two halves of one register.
func screenPoint(t *testing.T, w *window, x, y int32) uintptr {
	t.Helper()
	pt, ok := win32.ClientToScreen(w.hwnd, win32.Point{X: x, Y: y})
	if !ok {
		t.Fatal("ClientToScreen failed")
	}
	return uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32
}

// callTarget calls a method of the IDropTarget as a program that drags does and returns the HRESULT
// and the effect it was left with.
func dragEnter(t *testing.T, target, data win32.Object, key uint32, pt uintptr, allowed uint32) (win32.HRESULT, uint32) {
	t.Helper()
	effect := allowed
	hr := win32.FromResult(target.Call(win32.IDropTargetDragEnter, data.Ptr(), uintptr(key), pt, uintptr(unsafe.Pointer(&effect))))
	return hr, effect
}

func dragOver(target win32.Object, key uint32, pt uintptr, allowed uint32) (win32.HRESULT, uint32) {
	effect := allowed
	hr := win32.FromResult(target.Call(win32.IDropTargetDragOver, uintptr(key), pt, uintptr(unsafe.Pointer(&effect))))
	return hr, effect
}

func dragDrop(target, data win32.Object, key uint32, pt uintptr, allowed uint32) (win32.HRESULT, uint32) {
	effect := allowed
	hr := win32.FromResult(target.Call(win32.IDropTargetDrop, data.Ptr(), uintptr(key), pt, uintptr(unsafe.Pointer(&effect))))
	return hr, effect
}

// testData is a data object made by the test with the formats given.
func testData(t *testing.T, p *Platform, items ...ole.Item) win32.Object {
	t.Helper()
	obj, err := p.ole.NewDataObject(ole.NewDataSet(items...), p.oleRuntime())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { obj.Release() })
	return obj
}

func TestWineEveryWindowIsADropTargetUntilItCloses(t *testing.T) {
	p, r := newTestPlatform(t)
	if p.ole == nil {
		t.Fatal("OLE objects could not be made")
	}
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		ta, tb := registeredTarget(t, a), registeredTarget(t, b)
		if ta.Ptr() == tb.Ptr() {
			t.Error("two windows share one drop target")
		}
		before := p.ole.Stats()
		if before.Objects < 2 {
			t.Errorf("%d COM objects alive for two registered windows", before.Objects)
		}
		// The window registered twice is refused by OLE.
		if err := win32.RegisterDragDrop(a.hwnd, ta); err == nil {
			t.Error("a second RegisterDragDrop for the same window worked")
		}
		hwndA := a.hwnd
		a.Close()
		if win32.GetProp(hwndA, dropTargetProp) != 0 {
			t.Error("the destroyed window still has a drop target")
		}
		if after := p.ole.Stats(); after.Objects != before.Objects-1 {
			t.Errorf("objects: %d with two windows, %d after one closed; the target should be freed with its window", before.Objects, after.Objects)
		}
		if win32.GetProp(b.hwnd, dropTargetProp) == 0 {
			t.Error("closing the first window took the drop target of the second")
		}
	})
}

func TestWineFilesAndTextHandedToTheRegisteredTargetArriveAsExternalData(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		target := registeredTarget(t, w)
		files := testData(t, p,
			ole.Item{Format: win32.CFHDrop, Data: dropFilesFromPaths([]string{`C:\Users\me\a b.txt`, `D:\x.png`})},
			ole.Item{Format: win32.CFUnicodeText, Data: textBytes(textToClipboard("a b.txt\nx.png"))},
		)
		text := testData(t, p, ole.Item{Format: win32.CFUnicodeText, Data: textBytes(textToClipboard("some\r\ntext"))})
		nothing := testData(t, p, ole.Item{Format: win32.CFDIB, Data: []byte{1, 2, 3}})
		all := uint32(win32.DropEffectCopy | win32.DropEffectMove | win32.DropEffectLink)
		pt := screenPoint(t, w, 40, 30)

		// Files: the framework accepts copy.
		r.dropAnswer = func(ui.DropEvent) ui.DropActions { return ui.DropActionCopy }
		hr, effect := dragEnter(t, target, files, 0, pt, all)
		if hr != win32.SOK || effect != win32.DropEffectCopy {
			t.Errorf("DragEnter = %v, effect %#x; want S_OK and copy", hr, effect)
		}
		// A move with the shift key: the framework is offered move only, and accepts it.
		r.dropAnswer = func(e ui.DropEvent) ui.DropActions { return e.Actions }
		hr, effect = dragOver(target, win32.MKShift, pt, all)
		if hr != win32.SOK || effect != win32.DropEffectMove {
			t.Errorf("DragOver with Shift = %v, effect %#x; want move", hr, effect)
		}
		r.dropAnswer = func(ui.DropEvent) ui.DropActions { return ui.DropActionCopy }
		hr, effect = dragDrop(target, files, 0, pt, all)
		if hr != win32.SOK || effect != win32.DropEffectCopy {
			t.Errorf("Drop = %v, effect %#x; want copy", hr, effect)
		}
		drop := r.drops[len(r.drops)-1]
		wantURIs := []string{"file:///C:/Users/me/a%20b.txt", "file:///D:/x.png"}
		if drop.Phase != ui.DropPhaseDrop || drop.ViewID != w.id || !slices.Equal(drop.URIs, wantURIs) || drop.Text != "a b.txt\nx.png" {
			t.Errorf("drop of files = %+v; want URIs %v", drop, wantURIs)
		}
		if drop.Position.Dx != 40 || drop.Position.Dy != 30 {
			t.Errorf("drop position = %v, want 40,30 (physical pixels of the client area)", drop.Position)
		}
		if !slices.Equal(drop.Types, []string{ole.MIMEURIList, ole.MIMEPlainUTF8}) {
			t.Errorf("types = %v", drop.Types)
		}

		// Text, with a CRLF.
		n := len(r.drops)
		dragEnter(t, target, text, 0, pt, all)
		dragDrop(target, text, 0, pt, all)
		if len(r.drops) != n+2 {
			t.Fatalf("%d events for a text drag, want 2", len(r.drops)-n)
		}
		if d := r.drops[len(r.drops)-1]; d.Text != "some\ntext" || len(d.URIs) != 0 {
			t.Errorf("drop of text = %+v", d)
		}

		// A data object that offers nothing we know: the app hears of the drag, with no types and no data.
		r.dropAnswer = func(ui.DropEvent) ui.DropActions { return ui.DropActionNone }
		n = len(r.drops)
		_, effect = dragEnter(t, target, nothing, 0, pt, all)
		dragDrop(target, nothing, 0, pt, all)
		if effect != win32.DropEffectNone {
			t.Errorf("effect for data we do not know = %#x, want none", effect)
		}
		if d := r.drops[len(r.drops)-1]; len(r.drops) != n+2 || len(d.Types) != 0 || d.Text != "" || len(d.URIs) != 0 {
			t.Errorf("drop of unknown data = %+v", r.drops[n:])
		}

		// A leave ends a drag with no drop.
		n = len(r.drops)
		dragEnter(t, target, text, 0, pt, all)
		target.Call(win32.IDropTargetDragLeave)
		if got := r.drops[n:]; len(got) != 2 || got[1].Phase != ui.DropPhaseLeave {
			t.Errorf("enter and leave = %+v", got)
		}
	})
}

// place puts the window at a place of the screen without changing its size.
func place(t *testing.T, w *window, x, y int32) {
	t.Helper()
	if err := win32.SetWindowPos(w.hwnd, 0, x, y, 0, 0, win32.SWPNoSize|win32.SWPNoZOrder|win32.SWPNoActivate); err != nil {
		t.Fatal(err)
	}
}

// waitFor polls cond on the UI thread, every 20 ms, until it holds or the limit passes; then next.
func waitFor(t *testing.T, p *Platform, what string, limit time.Duration, cond func() bool, next func()) {
	t.Helper()
	deadline := time.Now().Add(limit)
	var poll func()
	poll = func() {
		if cond() {
			next()
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("timed out waiting for %s", what)
			next()
			return
		}
		after(p, 20*time.Millisecond, poll)
	}
	poll()
}

// inputFromAnotherThread plays the steps of a user on a goroutine, so that they go on while the UI
// thread is inside the modal loop of DoDragDrop. The steps are SendInput and SetCursorPos, which work
// from any thread.
func inputFromAnotherThread(gap time.Duration, steps ...func()) {
	go func() {
		for _, s := range steps {
			time.Sleep(gap)
			s()
		}
	}()
}

// moveTo puts the pointer at a place of the client area of the window (safe on any thread).
func moveTo(w *window, x, y int32) func() {
	return func() {
		if pt, ok := win32.ClientToScreen(w.hwnd, win32.Point{X: x, Y: y}); ok {
			win32.SetCursorPos(pt.X, pt.Y)
		}
	}
}

func inputs(flags ...uint32) func() {
	return func() {
		in := make([]win32.Input, len(flags))
		for i, f := range flags {
			in[i] = mouseInput(f)
		}
		win32.SendInput(in)
	}
}

// dragFrom plays a drag of the app: the pointer goes to the place in the window, the left button goes
// down, the drag starts, and the script goes on (on its own thread, while the UI thread is inside the
// loop of DoDragDrop). check runs on the UI thread once the drag has ended and the loop went on a
// little; the run ends after it.
func dragFrom(t *testing.T, p *Platform, r *rec, from *window, req ui.DragRequest, check func(), script ...func()) {
	t.Helper()
	t.Cleanup(func() { sendInput(t) })
	runFor(t, p, r, 40*time.Second, func(cancel context.CancelFunc) {
		seq(
			func(next func()) { moveCursorTo(t, from, 100, 90); after(p, 150*time.Millisecond, next) },
			func(next func()) {
				sendInput(t, win32.MouseEventFLeftDown)
				after(p, 150*time.Millisecond, next)
			},
			func(next func()) {
				if err := from.StartDrag(req); err != nil {
					t.Errorf("StartDrag: %v", err)
					cancel()
					return
				}
				// The loop of the drag runs in the next turn; the script runs meanwhile, on its own thread.
				inputFromAnotherThread(150*time.Millisecond, script...)
				waitFor(t, p, "the end of the drag", 15*time.Second, func() bool { return len(r.dragEnds) > 0 }, next)
			},
			func(next func()) { after(p, 200*time.Millisecond, next) },
			func(next func()) {
				defer cancel()
				check()
				if p.drags().Active() {
					t.Error("the drag is still active")
				}
			},
		)
	})
}

// eventsOf are the drop events of the view.
func (r *rec) eventsOf(view ui.ViewID) []ui.DropEvent {
	var out []ui.DropEvent
	for _, e := range r.drops {
		if e.ViewID == view {
			out = append(out, e)
		}
	}
	return out
}

func phases(events []ui.DropEvent) []ui.DropPhase {
	out := make([]ui.DropPhase, len(events))
	for i, e := range events {
		out[i] = e.Phase
	}
	return out
}

// twoWindows opens two windows side by side.
func twoWindows(t *testing.T) (*Platform, *rec, *window, *window) {
	t.Helper()
	p, r := newTestPlatform(t)
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	place(t, a, 20, 20)
	place(t, b, 420, 20)
	return p, r, a, b
}

func TestWineDragFromOneWindowToTheOtherArrivesAsTheAppsOwnData(t *testing.T) {
	p, r, a, b := twoWindows(t)
	r.dropAnswer = func(e ui.DropEvent) ui.DropActions {
		if e.ViewID == b.id {
			return ui.DropActionCopy
		}
		return ui.DropActionNone
	}
	dragFrom(t, p, r, a, ui.DragRequest{ID: 77, Text: "hello", Actions: ui.DropActionCopy | ui.DropActionMove}, func() {
		if want := (ui.DragResult{ID: 77, Action: ui.DropActionCopy}); len(r.dragEnds) != 1 || r.dragEnds[0] != want {
			t.Errorf("OnDragEnd = %+v, want %+v", r.dragEnds, want)
		}
		onB := r.eventsOf(b.id)
		if len(onB) < 3 || onB[0].Phase != ui.DropPhaseEnter || onB[len(onB)-1].Phase != ui.DropPhaseDrop {
			t.Fatalf("events on the second window: %v", phases(onB))
		}
		drop := onB[len(onB)-1]
		if drop.LocalID != 77 || drop.Text != "hello" {
			t.Errorf("drop = %+v: want the app's own drag 77 with its text", drop)
		}
		if drop.Position.Dx < 100 || drop.Position.Dx > 120 || drop.Position.Dy < 85 || drop.Position.Dy > 105 {
			t.Errorf("drop position = %v, want about 110,95", drop.Position)
		}
		if !slices.Contains(drop.Types, ole.MIMEPlainUTF8) {
			t.Errorf("types = %v", drop.Types)
		}
		for _, e := range onB {
			if slices.ContainsFunc(e.Types, ole.IsMarker) {
				t.Errorf("the marker of the drag is among the types: %v", e.Types)
			}
		}
		// The button went up inside OLE's loop: the window got a synthesized up.
		if countChange(r.pointers, ui.PointerChangeDown) != 1 || countChange(r.pointers, ui.PointerChangeUp) != 1 {
			t.Errorf("pointer changes = %v, want one down and one up", r.changes())
		}
	},
		moveTo(a, 120, 100), moveTo(b, 60, 60), moveTo(b, 100, 90), moveTo(b, 110, 95),
		inputs(win32.MouseEventFLeftUp),
	)
}

func TestWineEscapeCancelsADrag(t *testing.T) {
	p, r, a, b := twoWindows(t)
	r.dropAnswer = func(ui.DropEvent) ui.DropActions { return ui.DropActionCopy }
	dragFrom(t, p, r, a, ui.DragRequest{ID: 5, Text: "x", Actions: ui.DropActionCopy}, func() {
		if want := (ui.DragResult{ID: 5}); len(r.dragEnds) != 1 || r.dragEnds[0] != want {
			t.Errorf("OnDragEnd = %+v, want %+v: a cancelled drag is not accepted", r.dragEnds, want)
		}
		onB := r.eventsOf(b.id)
		if len(onB) == 0 || slices.Contains(phases(onB), ui.DropPhaseDrop) || onB[len(onB)-1].Phase != ui.DropPhaseLeave {
			t.Errorf("events on the second window: %v; want a leave and no drop", phases(onB))
		}
		if slices.Contains(phases(r.eventsOf(a.id)), ui.DropPhaseDrop) {
			t.Error("the first window got a drop")
		}
		// The button went up inside OLE's loop, or was still down when the loop ended: the first up is the
		// one the platform synthesized. (The button then goes up over the second window, which sees a
		// pointer that arrives with it down; that is a second contact.)
		var up *ui.PointerData
		for i := range r.pointers {
			if r.pointers[i].Change == ui.PointerChangeUp {
				up = &r.pointers[i]
				break
			}
		}
		if up == nil || !up.Synthesized || up.ViewID != a.id {
			t.Errorf("pointer changes = %v, first up = %+v: want a synthesized up in the first window", r.changes(), up)
		}
	},
		moveTo(a, 120, 100), moveTo(b, 60, 60), moveTo(b, 100, 90),
		func() { sendScan(0x01, false); sendScan(0x01, true) }, // Escape
		inputs(win32.MouseEventFLeftUp),
	)
}

func TestWineADropTheAppRefusesEndsNotAccepted(t *testing.T) {
	p, r, a, b := twoWindows(t)
	// The framework accepts the drag over no window: the effect stays none, OLE sends a leave, not a drop.
	dragFrom(t, p, r, a, ui.DragRequest{ID: 6, Text: "x", Actions: ui.DropActionCopy}, func() {
		if want := (ui.DragResult{ID: 6}); len(r.dragEnds) != 1 || r.dragEnds[0] != want {
			t.Errorf("OnDragEnd = %+v, want %+v", r.dragEnds, want)
		}
		if onB := r.eventsOf(b.id); slices.Contains(phases(onB), ui.DropPhaseDrop) || len(onB) == 0 || onB[len(onB)-1].Phase != ui.DropPhaseLeave {
			t.Errorf("events on the second window: %v", phases(onB))
		}
	},
		moveTo(b, 60, 60), moveTo(b, 100, 90), inputs(win32.MouseEventFLeftUp),
	)
}

func TestWineADragThatReturnsToItsOwnWindowIsRecognised(t *testing.T) {
	p, r, a, b := twoWindows(t)
	r.dropAnswer = func(ui.DropEvent) ui.DropActions { return ui.DropActionCopy }
	dragFrom(t, p, r, a, ui.DragRequest{ID: 9, Text: "mine", Actions: ui.DropActionCopy}, func() {
		onA := r.eventsOf(a.id)
		if len(onA) == 0 || onA[len(onA)-1].Phase != ui.DropPhaseDrop {
			t.Fatalf("events on the first window: %v; want a drop at the end", phases(onA))
		}
		if drop := onA[len(onA)-1]; drop.LocalID != 9 || drop.Text != "mine" {
			t.Errorf("drop = %+v", drop)
		}
		if len(r.dragEnds) != 1 || r.dragEnds[0] != (ui.DragResult{ID: 9, Action: ui.DropActionCopy}) {
			t.Errorf("OnDragEnd = %+v", r.dragEnds)
		}
		if slices.Contains(phases(r.eventsOf(b.id)), ui.DropPhaseDrop) {
			t.Error("the second window got the drop")
		}
	},
		moveTo(b, 60, 60), moveTo(b, 100, 90), moveTo(a, 200, 120), moveTo(a, 210, 130),
		inputs(win32.MouseEventFLeftUp),
	)
}

func TestWineTheEffectTheFrameworkChoosesIsWhatTheDragEndsWith(t *testing.T) {
	for _, tt := range []struct {
		name    string
		allowed ui.DropActions
		answer  ui.DropActions
		want    ui.DropActions
	}{
		{"move", ui.DropActionCopy | ui.DropActionMove, ui.DropActionMove, ui.DropActionMove},
		{"link", ui.DropActionLink, ui.DropActionLink, ui.DropActionLink},
		{"an action the source does not allow", ui.DropActionCopy, ui.DropActionMove, ui.DropActionNone},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, r, a, b := twoWindows(t)
			r.dropAnswer = func(e ui.DropEvent) ui.DropActions {
				if e.ViewID == b.id {
					return tt.answer
				}
				return ui.DropActionNone
			}
			dragFrom(t, p, r, a, ui.DragRequest{ID: 3, Text: "x", Actions: tt.allowed}, func() {
				if len(r.dragEnds) != 1 || r.dragEnds[0] != (ui.DragResult{ID: 3, Action: tt.want}) {
					t.Errorf("OnDragEnd = %+v, want action %v", r.dragEnds, tt.want)
				}
			},
				moveTo(b, 60, 60), moveTo(b, 100, 90), inputs(win32.MouseEventFLeftUp),
			)
		})
	}
}

func TestWineStartDragErrors(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	t.Cleanup(func() { sendInput(t) })
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		seq(
			func(next func()) {
				// No button is down.
				assertTagRoute(t, w.StartDrag(ui.DragRequest{ID: 1}), "DragNoButton", "window.StartDrag->dragRunner.Start.DragNoButton")
				moveCursorTo(t, w, 50, 50)
				after(p, 100*time.Millisecond, next)
			},
			func(next func()) { sendInput(t, win32.MouseEventFLeftDown); after(p, 150*time.Millisecond, next) },
			func(next func()) {
				if err := w.StartDrag(ui.DragRequest{ID: 2, Text: "x"}); err != nil {
					t.Fatalf("StartDrag: %v", err)
				}
				// The first drag has not run yet.
				assertTagRoute(t, w.StartDrag(ui.DragRequest{ID: 3}), "DragInProgress", "window.StartDrag->dragRunner.Start.DragInProgress")
				// The button is released at once: the loop has nothing to follow and the drag ends, not accepted.
				sendInput(t, win32.MouseEventFLeftUp)
				after(p, 200*time.Millisecond, next)
			},
			func(next func()) {
				if len(r.dragEnds) != 1 || r.dragEnds[0] != (ui.DragResult{ID: 2}) {
					t.Errorf("OnDragEnd = %+v", r.dragEnds)
				}
				// OLE is not available.
				ole := p.ole
				p.ole = nil
				assertTagRoute(t, w.StartDrag(ui.DragRequest{ID: 4}), "DragUnsupported", "window.StartDrag.DragUnsupported")
				p.ole = ole
				// A window that is gone.
				w.Close()
				assertTagRoute(t, w.StartDrag(ui.DragRequest{ID: 5}), "DragUnsupported", "window.StartDrag.DragUnsupported")
				cancel()
			},
		)
	})
}
