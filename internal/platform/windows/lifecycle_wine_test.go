//go:build windows && glutter_wine

package windows

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// These tests check the lifecycle that the platform reports under Wine (SPEC §25.8).

// lifeStep runs do, waits until the app has reported a state after the ones it had, and calls check
// with the states reported since do; then next.
func lifeStep(t *testing.T, p *Platform, r *rec, name string, do func(), want []ui.AppLifecycleState, next func()) {
	t.Helper()
	n := len(r.lifecycle)
	do()
	var wait func(deadline time.Time)
	wait = func(deadline time.Time) {
		got := r.lifecycle[n:]
		if len(got) >= len(want) {
			// A report made for more than the states wanted is wrong: look a little longer.
			after(p, 150*time.Millisecond, func() {
				if got := r.lifecycle[n:]; !slices.Equal(got, want) {
					t.Errorf("%s: states %v, want %v", name, got, want)
				}
				next()
			})
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("%s: states %v, want %v", name, got, want)
			next()
			return
		}
		after(p, 20*time.Millisecond, func() { wait(deadline) })
	}
	wait(time.Now().Add(5 * time.Second))
}

func TestWineMinimizeAndRestoreGiveHiddenThenResumed(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 320, Height: 240}})
	runFor(t, p, r, 40*time.Second, func(cancel context.CancelFunc) {
		defer func() {}()
		focusWindow(w)
		// The first window of the app, shown and active, is the first state.
		wait := time.Now().Add(5 * time.Second)
		var first func()
		first = func() {
			if len(r.lifecycle) == 0 {
				if time.Now().After(wait) {
					t.Errorf("no lifecycle state after the first window")
					cancel()
					return
				}
				after(p, 20*time.Millisecond, first)
				return
			}
			if !reflect.DeepEqual(r.lifecycle, []ui.AppLifecycleState{ui.AppLifecycleStateResumed}) {
				t.Errorf("first states = %v, want [Resumed]", r.lifecycle)
			}
			seq(
				func(next func()) {
					lifeStep(t, p, r, "minimize", func() { w.SetState(platform.WindowStateMinimized) }, []ui.AppLifecycleState{ui.AppLifecycleStateHidden}, next)
				},
				func(next func()) {
					// A minimized window draws no frame while Hidden.
					nf := len(r.frames)
					w.RequestFrame()
					after(p, 200*time.Millisecond, func() {
						if len(r.frames) != nf {
							t.Errorf("%d frames while the app was Hidden", len(r.frames)-nf)
						}
						next()
					})
				},
				func(next func()) {
					lifeStep(t, p, r, "restore", func() { w.SetState(platform.WindowStateNormal); focusWindow(w) }, []ui.AppLifecycleState{ui.AppLifecycleStateResumed}, next)
				},
				func(next func()) {
					lifeStep(t, p, r, "hide", func() { w.SetHidden(true) }, []ui.AppLifecycleState{ui.AppLifecycleStateHidden}, next)
				},
				func(next func()) {
					lifeStep(t, p, r, "show", func() { w.SetHidden(false) }, []ui.AppLifecycleState{ui.AppLifecycleStateResumed}, next)
				},
				func(next func()) { cancel() },
			)
		}
		first()
	})
	// Run ends: the app is Detached.
	if got := r.lifecycle[len(r.lifecycle)-1]; got != ui.AppLifecycleStateDetached {
		t.Errorf("the last state = %v, want Detached; all: %v", got, r.lifecycle)
	}
}

func TestWineSeveralWindowsAndTheStateOfTheApp(t *testing.T) {
	p, r := newTestPlatform(t)
	a := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	b := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	runFor(t, p, r, 40*time.Second, func(cancel context.CancelFunc) {
		seq(
			func(next func()) {
				lifeStep(t, p, r, "both shown", func() { focusWindow(a) }, []ui.AppLifecycleState{ui.AppLifecycleStateResumed}, next)
			},
			func(next func()) {
				// One window minimized, the other still on the screen: the app is not Hidden.
				lifeStep(t, p, r, "one minimized", func() { b.SetState(platform.WindowStateMinimized) }, nil, next)
			},
			func(next func()) {
				if !reflect.DeepEqual(r.lifecycle, []ui.AppLifecycleState{ui.AppLifecycleStateResumed}) {
					t.Errorf("states = %v: the app stayed Resumed with a window on the screen", r.lifecycle)
				}
				lifeStep(t, p, r, "both minimized", func() { a.SetState(platform.WindowStateMinimized) }, []ui.AppLifecycleState{ui.AppLifecycleStateHidden}, next)
			},
			func(next func()) {
				lifeStep(t, p, r, "one restored", func() { b.SetState(platform.WindowStateNormal); focusWindow(b) }, []ui.AppLifecycleState{ui.AppLifecycleStateResumed}, next)
			},
			func(next func()) { cancel() },
		)
	})
}

func TestWineTheSessionEnds(t *testing.T) {
	p, r := newTestPlatform(t)
	w := newWin(t, p, platform.WindowConfig{Size: ui.Size{Width: 200, Height: 150}})
	start := time.Now()
	runFor(t, p, r, 30*time.Second, func(cancel context.CancelFunc) {
		// The query is answered TRUE, so that the app does not hold the shutdown.
		if got := win32.SendMessage(w.hwnd, win32.WMQueryEndSession, 0, win32.EndSessionLogoff); got != 1 {
			t.Errorf("WM_QUERYENDSESSION answered %d, want 1", got)
		}
		if got := win32.SendMessage(p.service.Load(), win32.WMQueryEndSession, 0, 0); got != 1 {
			t.Errorf("WM_QUERYENDSESSION to the service window answered %d, want 1", got)
		}
		// A session whose end was cancelled changes nothing.
		win32.SendMessage(w.hwnd, win32.WMEndSession, 0, win32.EndSessionLogoff)
		if p.quit || p.ended {
			t.Error("a cancelled end of the session ended the app")
		}
		before := len(r.lifecycle)
		win32.SendMessage(w.hwnd, win32.WMEndSession, 1, win32.EndSessionLogoff)
		if got := r.lifecycle[len(r.lifecycle)-1]; len(r.lifecycle) == before || got != ui.AppLifecycleStateDetached {
			t.Errorf("states %v: Detached was not reported at once", r.lifecycle[before:])
		}
		if len(r.closes) != 0 {
			t.Errorf("close requests = %v: the end of the session is not an exit request", r.closes)
		}
	})
	if time.Since(start) > 10*time.Second {
		t.Error("Run did not end at once")
	}
	if n := countOf(r.lifecycle, ui.AppLifecycleStateDetached); n != 1 {
		t.Errorf("Detached was reported %d times: %v", n, r.lifecycle)
	}
}

func countOf(states []ui.AppLifecycleState, s ui.AppLifecycleState) int {
	n := 0
	for _, x := range states {
		if x == s {
			n++
		}
	}
	return n
}
