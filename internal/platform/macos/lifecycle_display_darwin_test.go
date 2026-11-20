//go:build darwin && !ios && glutter_display

package macos

import (
	"slices"
	"testing"
	"time"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"

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

// sendQuitEvent sends the app a quit Apple event, as the Dock (reason 0) or the system at a logout ('logo')
// sends it, and returns the error number of the reply (0 for none). The event goes to this process, which
// the Apple Event Manager dispatches to the handler at once, on the main thread.
func sendQuitEvent(t *testing.T, reason uint32) int32 {
	t.Helper()
	aed := class("NSAppleEventDescriptor")
	target := aed.Send(sel("currentProcessDescriptor"))
	ev := aed.Send(sel("appleEventWithEventClass:eventID:targetDescriptor:returnID:transactionID:"),
		uint32(aeCoreEventClass), uint32(aeQuitApplication), target, int16(-1), int32(0))
	if reason != 0 {
		ev.Send(sel("setAttributeDescriptor:forKeyword:"), aed.Send(sel("descriptorWithEnumCode:"), reason), uint32(aeQuitReason))
	}
	var nserr objc.ID
	const waitForReply = 3 // NSAppleEventSendWaitForReply (kAEWaitReply)
	reply := ev.Send(sel("sendEventWithOptions:timeout:error:"), uint64(waitForReply), float64(5), &nserr)
	if reply == 0 {
		t.Fatalf("the quit event was not sent: %s", describe(nserr))
	}
	errn := reply.Send(sel("paramDescriptorForKeyword:"), uint32(aeErrorNumber))
	if errn == 0 {
		return 0
	}
	return send[int32](errn, "int32Value")
}

// describe is the description of an Objective-C object, for a failure.
func describe(o objc.ID) string {
	if o == 0 {
		return "(nil)"
	}
	return objcx.String(o.Send(sel("description")))
}

// TestQuitEventSaysWhyAndIsCancelledOnlyByARefusal is mardp's issue (step 51.2): a quit Apple event reaches
// the windows with its reason (the system's logout is the end of the session, the Dock's Quit is the app's),
// and its reply says "user cancelled" only when the app refused, so a logout that the app agrees to goes on.
func TestQuitEventSaysWhyAndIsCancelledOnlyByARefusal(t *testing.T) {
	h := &recorder{denyClose: true}
	runApp(t, Options{}, h, func(s *session) {
		s.newWindow(platform.WindowConfig{Size: ui.Size{Width: 200, Height: 100}})
		s.waitFor("the first frame", 5*time.Second, func() bool { return h.frameCount() > 0 })
		s.onUI(func() {
			if got := sendQuitEvent(t, aeReallyLogOut); got != aeUserCanceledErr {
				t.Errorf("the reply to a refused logout has the error %d, want %d", got, aeUserCanceledErr)
			}
			if got := sendQuitEvent(t, 0); got != aeUserCanceledErr {
				t.Errorf("the reply to a refused quit has the error %d, want %d", got, aeUserCanceledErr)
			}
			// Cmd+Q and the app menu's Quit are terminate:.
			s.p.app.Send(sel("terminate:"), s.p.app)
			if s.p.windowCount() != 1 {
				t.Errorf("%d windows after refused quits, want 1", s.p.windowCount())
			}
		})
		h.mu.Lock()
		want := []ui.AppExitReason{ui.AppExitReasonSessionEnd, ui.AppExitReasonQuit, ui.AppExitReasonQuit}
		if !slices.Equal(h.reasons, want) {
			t.Errorf("reasons = %v, want %v", h.reasons, want)
		}
		h.denyClose = false
		h.mu.Unlock()
		s.onUI(func() {
			if got := sendQuitEvent(t, aeLogOut); got != 0 {
				t.Errorf("the reply to an agreed logout has the error %d, want none", got)
			}
		})
		s.waitFor("the loop to end", 5*time.Second, func() bool {
			return slices.Contains(h.lifecycleHeard(), ui.AppLifecycleStateDetached)
		})
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	if got := h.reasons[len(h.reasons)-1]; got != ui.AppExitReasonSessionEnd {
		t.Errorf("the reason of the agreed logout = %v", got)
	}
}
