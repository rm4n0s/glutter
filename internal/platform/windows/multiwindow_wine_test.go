//go:build windows && glutter_wine

package windows

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests check the several windows of an app under Wine (SPEC §25.4): dialogs with an owner and
// the modal block, the focus between windows, and closing the first window while others stay.

// dialogOf makes a dialog of the owner; modal says whether it blocks the owner.
func dialogOf(t *testing.T, p *Platform, owner *window, modal bool) *window {
	t.Helper()
	return newWin(t, p, platform.WindowConfig{
		Kind: platform.WindowKindDialog, Parent: owner.id, HasParent: true, Modal: modal,
		Size: ui.Size{Width: 200, Height: 120},
	})
}

func TestWineAModalDialogDisablesItsOwnerUntilItCloses(t *testing.T) {
	p, r := newTestPlatform(t)
	owner := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	other := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	if !win32.IsWindowEnabled(owner.hwnd) {
		t.Fatal("the owner starts disabled")
	}
	d1 := dialogOf(t, p, owner, true)
	if win32.IsWindowEnabled(owner.hwnd) {
		t.Error("a modal dialog did not disable its owner")
	}
	if !win32.IsWindowEnabled(other.hwnd) || !win32.IsWindowEnabled(d1.hwnd) {
		t.Error("a modal dialog disabled a window that is not its owner, or itself")
	}
	if got := win32.GetWindowLongPtr(d1.hwnd, win32.GWLPHwndParent); got != owner.hwnd {
		t.Errorf("the owner of the dialog is %#x, want %#x", got, owner.hwnd)
	}
	if ex := uint32(win32.GetWindowLongPtr(d1.hwnd, win32.GWLExStyle)); ex&win32.WSExDlgModalFrame == 0 || ex&win32.WSExAppWindow != 0 {
		t.Errorf("the dialog's extended style %#x: want the dialog frame and no task-bar button", ex)
	}
	if ex := uint32(win32.GetWindowLongPtr(owner.hwnd, win32.GWLExStyle)); ex&win32.WSExDlgModalFrame != 0 {
		t.Errorf("the owner has the dialog frame: %#x", ex)
	}
	// A second dialog of the same owner: the owner stays disabled until both are closed.
	d2 := dialogOf(t, p, owner, true)
	d1.Close()
	if win32.IsWindowEnabled(owner.hwnd) {
		t.Error("the owner was enabled while another modal dialog was open")
	}
	// A dialog that is not modal blocks nothing.
	free := dialogOf(t, p, other, false)
	if !win32.IsWindowEnabled(other.hwnd) {
		t.Error("a dialog that is not modal disabled its owner")
	}
	free.Close()
	d2.Close()
	if !win32.IsWindowEnabled(owner.hwnd) {
		t.Error("the owner is still disabled after its dialogs closed")
	}
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) { cancel() })
}

func TestWineClosingTheOwnerClosesItsDialogsFirst(t *testing.T) {
	p, r := newTestPlatform(t)
	owner := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	other := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	d1 := dialogOf(t, p, owner, true)
	d2 := dialogOf(t, p, owner, true)
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		d1hwnd, d2hwnd := d1.hwnd, d2.hwnd
		owner.Close()
		// The handler is asked to close each dialog before the owner goes (so the framework lets go of
		// the views), the newest first, and the dialogs are destroyed with the owner.
		if want := []ui.ViewID{d2.id, d1.id}; !slices.Equal(r.closes, want) {
			t.Errorf("close requests = %v, want %v", r.closes, want)
		}
		if !owner.closed || !d1.closed || !d2.closed || win32.IsWindow(d1hwnd) || win32.IsWindow(d2hwnd) {
			t.Error("the dialogs were not destroyed with their owner")
		}
		if other.closed || !win32.IsWindow(other.hwnd) {
			t.Error("a window that is not the owner went away")
		}
		if p.quit {
			t.Error("the loop ended with a window still open")
		}
	})
}

func TestWineADialogWhoseHandlerRefusesIsStillDestroyedWithItsOwner(t *testing.T) {
	p, r := newTestPlatform(t)
	r.closeAnswer = false // the app would not close the dialog by itself
	owner := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	d := dialogOf(t, p, owner, true)
	newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		owner.Close()
		if !d.closed {
			t.Error("the dialog outlived its owner")
		}
	})
}

func TestWineClosingTheFirstWindowKeepsTheSecond(t *testing.T) {
	p, r := newTestPlatform(t)
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var framesAfter int
	p.Post(func() {
		seq(
			func(next func()) { after(p, 300*time.Millisecond, next) },
			func(next func()) {
				// The user closes the first window: the handler agrees, and the window goes.
				win32.SendMessage(a.hwnd, win32.WMClose, 0, 0)
				after(p, 300*time.Millisecond, next)
			},
			func(next func()) {
				if !a.closed || b.closed || !win32.IsWindow(b.hwnd) {
					t.Errorf("after the first window closed: first closed = %v, second closed = %v", a.closed, b.closed)
				}
				if p.quit {
					t.Error("Run ends with a window still open")
				}
				nf := len(r.frames)
				b.RequestFrame()
				after(p, 300*time.Millisecond, func() { framesAfter = len(r.frames) - nf; next() })
			},
			func(next func()) {
				win32.SendMessage(b.hwnd, win32.WMClose, 0, 0) // the last window: Run ends by itself
			},
		)
	})
	if err := p.Run(ctx, r); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("Run did not end after the last window closed")
	}
	if framesAfter == 0 {
		t.Error("the second window drew no frame after the first closed")
	}
	if want := []ui.ViewID{a.id, b.id}; !slices.Equal(r.closes, want) {
		t.Errorf("close requests = %v, want %v", r.closes, want)
	}
}

func TestWineTheFocusMovesFromOneWindowToTheOther(t *testing.T) {
	p, r := newTestPlatform(t)
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	place(t, a, 20, 20)
	place(t, b, 420, 20)
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		seq(
			func(next func()) { focusWindow(a); after(p, 300*time.Millisecond, next) },
			func(next func()) { r.focused = nil; focusWindow(b); after(p, 300*time.Millisecond, next) },
			func(next func()) {
				want := []focusChange{{a.id, false}, {b.id, true}}
				if !slices.Equal(r.focused, want) {
					t.Errorf("focus changes going to the second window = %v, want %v", r.focused, want)
				}
				r.focused = nil
				focusWindow(a)
				after(p, 300*time.Millisecond, next)
			},
			func(next func()) {
				want := []focusChange{{b.id, false}, {a.id, true}}
				if !slices.Equal(r.focused, want) {
					t.Errorf("focus changes going back = %v, want %v", r.focused, want)
				}
				cancel()
			},
		)
	})
}

func TestWineEveryRegularWindowHasATaskBarButtonAndADialogHasNone(t *testing.T) {
	p, r := newTestPlatform(t)
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	d := dialogOf(t, p, a, true)
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		// A window with no owner is a task-bar window; an owned one is not (Windows decides by the owner).
		for name, w := range map[string]*window{"first": a, "second": b} {
			if owner := win32.GetWindowLongPtr(w.hwnd, win32.GWLPHwndParent); owner != 0 {
				t.Errorf("the %s window has the owner %#x: it would have no task-bar button", name, owner)
			}
		}
		if owner := win32.GetWindowLongPtr(d.hwnd, win32.GWLPHwndParent); owner != a.hwnd {
			t.Errorf("the dialog's owner is %#x, want %#x", owner, a.hwnd)
		}
	})
}

func TestWineWindowsCascadeAndADialogOpensCenteredOnItsOwner(t *testing.T) {
	p, r := newTestPlatform(t)
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	c := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	max := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}, State: platform.WindowStateMaximized})
	d := dialogOf(t, p, b, true)
	rect := func(w *window) win32.Rect {
		rc, err := win32.GetWindowRect(w.hwnd)
		if err != nil {
			t.Fatal(err)
		}
		return rc
	}
	step := physical(cascadeStep, a.scale())
	ra, rb, rc := rect(a), rect(b), rect(c)
	if rb.Left != ra.Left+step || rb.Top != ra.Top+step || rc.Left != rb.Left+step || rc.Top != rb.Top+step {
		t.Errorf("the windows are at %v, %v, %v: want each a step of %d below and to the right of the one before", ra, rb, rc, step)
	}
	if rm := rect(max); rm.Left == rc.Left+step && rm.Top == rc.Top+step {
		t.Errorf("a window that starts maximized was cascaded: %v", rm)
	}
	// The dialog's center is the center of its owner (the dialog is made after the windows).
	rd := rect(d)
	cx, cy := (rb.Left+rb.Right)/2, (rb.Top+rb.Bottom)/2
	dx, dy := (rd.Left+rd.Right)/2, (rd.Top+rd.Bottom)/2
	if abs32(dx-cx) > 1 || abs32(dy-cy) > 1 {
		t.Errorf("the dialog %v is centered at %d,%d, its owner %v at %d,%d", rd, dx, dy, rb, cx, cy)
	}
	runFor(t, p, r, 20*time.Second, func(cancel context.CancelFunc) { cancel() })
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func TestWineActivatingAnOwnerThatADialogBlocksActivatesTheDialog(t *testing.T) {
	p, r := newTestPlatform(t)
	owner := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	inner := dialogOf(t, p, owner, true)
	innermost := dialogOf(t, p, inner, true) // a dialog of the dialog: both windows below it are blocked
	other := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 300, Height: 200}})
	if win32.IsWindowEnabled(owner.hwnd) || win32.IsWindowEnabled(inner.hwnd) || !win32.IsWindowEnabled(innermost.hwnd) {
		t.Fatal("the owner and the first dialog should be disabled, the innermost one enabled")
	}
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		focusWindow(other)
		after(p, 300*time.Millisecond, func() {
			defer cancel()
			r.focused = nil
			// What the task switcher does to the window that is blocked.
			win32.SendMessage(owner.hwnd, win32.WMActivate, win32.WAActive, 0)
			var gotInner, gotOwner, gotInnermost bool
			for _, f := range r.focused {
				if f.focused {
					gotOwner = gotOwner || f.view == owner.id
					gotInner = gotInner || f.view == inner.id
					gotInnermost = gotInnermost || f.view == innermost.id
				}
			}
			if !gotInnermost || gotOwner || gotInner {
				t.Errorf("focus changes = %v: want the innermost dialog to get the focus and no other window", r.focused)
			}
			// With the dialogs closed, the owner takes input again.
			innermost.Close()
			inner.Close()
			if !win32.IsWindowEnabled(owner.hwnd) {
				t.Error("the owner is still disabled")
			}
		})
	})
}
