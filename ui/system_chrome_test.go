package ui

import (
	"reflect"
	"testing"
)

// chromeHost is a host that can control the system's interface and records what it is asked.
type chromeHost struct {
	fakeHost
	log []string
	cb  func(bool)
}

func (h *chromeHost) SetPreferredOrientations(o []DeviceOrientation) {
	h.log = append(h.log, "orientations")
}
func (h *chromeHost) SetEnabledSystemUIMode(m SystemUIMode, o []SystemUIOverlay) {
	h.log = append(h.log, "mode")
}
func (h *chromeHost) SetSystemUIOverlayStyle(SystemUIOverlayStyle) { h.log = append(h.log, "style") }
func (h *chromeHost) SetSystemUIChangeCallback(fn func(visible bool)) {
	h.log = append(h.log, "callback")
	h.cb = fn
}
func (h *chromeHost) PerformHapticFeedback(k HapticFeedbackKind) { h.log = append(h.log, "haptic") }
func (h *chromeHost) PlaySystemSound(k SystemSoundKind)          { h.log = append(h.log, "sound") }

func TestSystemChromeCallsReachTheHost(t *testing.T) {
	h := &chromeHost{}
	d := NewPlatformDispatcher(h)
	d.SetPreferredOrientations([]DeviceOrientation{DeviceOrientationPortraitUp})
	d.SetEnabledSystemUIMode(SystemUIModeEdgeToEdge, nil)
	d.SetSystemUIOverlayStyle(SystemUIOverlayStyle{})
	d.SetSystemUIChangeCallback(func(bool) {})
	d.PerformHapticFeedback(HapticFeedbackVibrate)
	d.PlaySystemSound(SystemSoundClick)
	want := []string{"orientations", "mode", "style", "callback", "haptic", "sound"}
	if !reflect.DeepEqual(h.log, want) {
		t.Errorf("the host heard %v, want %v", h.log, want)
	}
}

func TestSystemChromeCallsAreIgnoredByAHostWithoutThem(t *testing.T) {
	for name, d := range map[string]*PlatformDispatcher{"plain host": NewPlatformDispatcher(&fakeHost{}), "no host": NewPlatformDispatcher(nil)} {
		t.Run(name, func(t *testing.T) {
			d.SetPreferredOrientations(nil)
			d.SetEnabledSystemUIMode(SystemUIModeLeanBack, nil)
			d.SetSystemUIOverlayStyle(SystemUIOverlayStyle{})
			d.SetSystemUIChangeCallback(nil)
			d.PerformHapticFeedback(HapticFeedbackHeavyImpact)
			d.PlaySystemSound(SystemSoundAlert)
		})
	}
}
