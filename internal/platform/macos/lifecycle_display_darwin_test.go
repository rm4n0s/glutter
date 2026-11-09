//go:build darwin && !ios && glutter_display

package macos

import (
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestLifecycleFollowsTheWindowAndTheApp(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		// An accessory app's window is never the key window: the first state is Inactive.
		s.waitFor("Inactive", 5*time.Second, func() bool { return slices.Contains(h.lifecycleHeard(), ui.AppLifecycleStateInactive) })
		s.onUI(func() { w.keyChanged(true) })
		s.onUI(func() { s.p.setAppHidden(true) })
		s.onUI(func() { s.p.setAppHidden(false) })
		s.onUI(func() { w.keyChanged(false) })
		s.onUI(func() { w.keyChanged(false) }) // no change, so no report
		want := []ui.AppLifecycleState{
			ui.AppLifecycleStateInactive, ui.AppLifecycleStateResumed, ui.AppLifecycleStateHidden,
			ui.AppLifecycleStateResumed, ui.AppLifecycleStateInactive,
		}
		if got := h.lifecycleHeard(); !slices.Equal(got, want) {
			t.Errorf("lifecycle = %v, want %v", got, want)
		}
		s.onUI(w.Close)
	})
	got := h.lifecycleHeard()
	if len(got) == 0 || got[len(got)-1] != ui.AppLifecycleStateDetached {
		t.Errorf("lifecycle = %v, want it to end with Detached", got)
	}
}

func TestTerminateAsksTheWindows(t *testing.T) {
	h := &recorder{denyClose: true}
	runApp(t, Options{}, h, func(s *session) {
		s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(func() { s.p.app.Send(sel("terminate:"), s.p.app) })
		s.onUI(func() {
			h.mu.Lock()
			asked := h.closeReq
			h.mu.Unlock()
			if asked != 1 {
				t.Errorf("OnCloseRequest was called %d times, want 1", asked)
			}
			if s.p.windowCount() != 1 {
				t.Errorf("%d windows after a refused quit, want 1", s.p.windowCount())
			}
		})
		h.mu.Lock()
		h.denyClose = false
		h.mu.Unlock()
		s.onUI(func() { s.p.app.Send(sel("terminate:"), s.p.app) })
		// Quitting closes the window, and the end of the last window ends the loop: Run reports Detached
		// before it returns (the body does not end until then).
		s.waitFor("the loop to end", 5*time.Second, func() bool {
			return slices.Contains(h.lifecycleHeard(), ui.AppLifecycleStateDetached)
		})
	})
	h.mu.Lock()
	asked := h.closeReq
	h.mu.Unlock()
	if asked != 2 {
		t.Errorf("OnCloseRequest was called %d times in all, want 2", asked)
	}
}

func TestQuitClosesTheWindowsThatAgreeAndStopsAtTheOneThatRefuses(t *testing.T) {
	h := &recorder{denyViews: map[ui.ViewID]bool{}}
	runApp(t, Options{}, h, func(s *session) {
		cfg := platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}}
		first, second, third := s.newWindow(cfg), s.newWindow(cfg), s.newWindow(cfg)
		s.waitFor("the first frames", 5*time.Second, func() bool { return h.frameCount() >= 3 })
		h.mu.Lock()
		h.denyViews[second.id] = true
		h.mu.Unlock()
		s.onUI(func() { s.p.app.Send(sel("terminate:"), s.p.app) })
		s.onUI(func() {
			if !first.closed || second.closed || third.closed {
				t.Errorf("closed: first %v, second %v, third %v; want only the first", first.closed, second.closed, third.closed)
			}
			if got := s.p.windowCount(); got != 2 {
				t.Errorf("%d windows left, want 2", got)
			}
		})
		s.onUI(second.Close)
		s.onUI(third.Close)
	})
}
