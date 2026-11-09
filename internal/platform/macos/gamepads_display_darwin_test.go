//go:build darwin && !ios && glutter_display

package macos

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/internal/platform"
)

// padRecorder is a pad handler that records what it is told.
type padRecorder struct{ added []platform.PadInfo }

func (r *padRecorder) PadAdded(i platform.PadInfo)   { r.added = append(r.added, i) }
func (r *padRecorder) PadRemoved(platform.PadID)     {}
func (r *padRecorder) PadEvents([]platform.PadEvent) {}

// TestGamepadsStartAndStopOnTheMainRunLoop starts the gamepad service of the platform (IOKit's HID manager on the real main run
// loop, SPEC §29.11) and stops it, twice, with the window system running. A machine without pads reports none; one that has
// them reports them as raw pads of "Mac OS X", with their buttons, axes and hats. What a pad does is a manual check.
func TestGamepadsStartAndStopOnTheMainRunLoop(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		var g platform.Gamepads
		pads := &padRecorder{}
		s.onUI(func() {
			svc := s.p.Services()
			if svc.Gamepads() == nil || svc.Gamepads() != s.p.Services().Gamepads() {
				t.Error("the services have no gamepads, or make a second service")
				return
			}
			g = svc.Gamepads()
			if g.MappingPlatform() != "Mac OS X" {
				t.Errorf("MappingPlatform = %q", g.MappingPlatform())
			}
			if err := g.Start(pads); err != nil {
				t.Errorf("Start: %v", err)
			}
		})
		// The run loop turns while the test waits, which is when IOKit reports the pads that are there.
		time.Sleep(300 * time.Millisecond)
		s.onUI(func() {
			for _, p := range pads.added {
				t.Logf("pad %q: %d axes, %d buttons, %d hats, standard %v", p.Name, p.Axes, p.Buttons, p.Hats, p.Standard)
				if p.Axes+p.Buttons+p.Hats == 0 && !p.Standard {
					t.Errorf("pad %q has no inputs", p.Name)
				}
			}
			g.Stop()
			if err := g.Start(pads); err != nil {
				t.Errorf("second Start: %v", err)
			}
			g.Stop()
		})
	})
}

// TestAudioServiceIsProvided checks that the services of the platform have the audio service of Core Audio (SPEC §29.11), once,
// and that it lists the devices of the machine (a Mac with no output lists none, which is no error). Playing and recording are
// tested in package coreaudio.
func TestAudioServiceIsProvided(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		s.onUI(func() {
			a := s.p.Services().Audio()
			if a == nil || a != s.p.Services().Audio() {
				t.Error("the services have no audio, or make a second service")
				return
			}
			for _, kind := range []platform.AudioKind{platform.AudioPlayback, platform.AudioCapture} {
				if _, err := a.Devices(kind); err != nil {
					t.Errorf("Devices(%d): %v", kind, err)
				}
			}
		})
	})
}
