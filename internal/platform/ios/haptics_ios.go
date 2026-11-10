//go:build ios

package ios

import (
	"github.com/rm4n0s/glutter/internal/platform/darwin/dl"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the haptics and the system sounds of iOS (SPEC §30.6), after FlutterPlatformPlugin: UIKit's feedback generators for the
// impacts and the selection, and AudioServices for the vibration and the sounds.

// loadAudioToolbox binds AudioServicesPlaySystemSound.
func loadAudioToolbox(s *system) error {
	at, err := dl.OpenFramework("AudioToolbox")
	if err != nil {
		return err
	}
	return at.Bind(&s.playSound, "AudioServicesPlaySystemSound")
}

// PerformHapticFeedback gives the feedback: an impact of the style, the selection tick, or the system's vibration. A feedback that the
// device cannot make (an iPad has no Taptic Engine) is a no-op for UIKit, as Flutter's is.
//
// Flutter: services/haptic_feedback.dart HapticFeedback
func (w *window) PerformHapticFeedback(kind ui.HapticFeedbackKind) {
	h := hapticOf(kind)
	objcx.WithPool(func() {
		switch h.kind {
		case hapticVibrate:
			w.p.s.playSound(soundVibrate)
		case hapticImpact:
			g := class("UIImpactFeedbackGenerator").Send(sel("alloc")).Send(sel("initWithStyle:"), h.style)
			g.Send(sel("impactOccurred"))
			objcx.Release(g)
		case hapticSelection:
			g := class("UISelectionFeedbackGenerator").Send(sel("alloc")).Send(sel("init"))
			g.Send(sel("selectionChanged"))
			objcx.Release(g)
		}
	})
}

// PlaySystemSound plays the sound: the click of the keyboard, or the tick. The alert has no sound on iOS.
//
// Flutter: services/system_sound.dart SystemSound.play
func (w *window) PlaySystemSound(kind ui.SystemSoundKind) {
	if id, ok := soundOf(kind); ok {
		w.p.s.playSound(id)
	}
}
