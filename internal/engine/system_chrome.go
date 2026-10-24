package engine

import "github.com/rm4n0s/glutter/ui"

var _ ui.SystemChromeHost = (*handler)(nil)

// systemChrome is the window that shows the system's interface around the app, when the
// platform's window can control it (Android's); nil otherwise. There is one view on those
// platforms.
func (h *handler) systemChrome() ui.SystemChromeHost {
	w := h.windowOf(ui.ImplicitViewID)
	if w == nil {
		return nil
	}
	c, _ := w.Window.(ui.SystemChromeHost)
	return c
}

// SetPreferredOrientations limits the orientations of the app (ui.SystemChromeHost).
func (h *handler) SetPreferredOrientations(orientations []ui.DeviceOrientation) {
	if c := h.systemChrome(); c != nil {
		c.SetPreferredOrientations(orientations)
	}
}

// SetEnabledSystemUIMode shows or hides the system's bars (ui.SystemChromeHost).
func (h *handler) SetEnabledSystemUIMode(mode ui.SystemUIMode, overlays []ui.SystemUIOverlay) {
	if c := h.systemChrome(); c != nil {
		c.SetEnabledSystemUIMode(mode, overlays)
	}
}

// SetSystemUIOverlayStyle sets the look of the system's bars (ui.SystemChromeHost).
func (h *handler) SetSystemUIOverlayStyle(style ui.SystemUIOverlayStyle) {
	if c := h.systemChrome(); c != nil {
		c.SetSystemUIOverlayStyle(style)
	}
}

// SetSystemUIChangeCallback sets the function that hears of the system's bars appearing or
// disappearing (ui.SystemChromeHost).
func (h *handler) SetSystemUIChangeCallback(fn func(visible bool)) {
	if c := h.systemChrome(); c != nil {
		c.SetSystemUIChangeCallback(fn)
	}
}

// PerformHapticFeedback gives haptic feedback (ui.SystemChromeHost).
func (h *handler) PerformHapticFeedback(kind ui.HapticFeedbackKind) {
	if c := h.systemChrome(); c != nil {
		c.PerformHapticFeedback(kind)
	}
}

// PlaySystemSound plays a sound of the system (ui.SystemChromeHost).
func (h *handler) PlaySystemSound(kind ui.SystemSoundKind) {
	if c := h.systemChrome(); c != nil {
		c.PlaySystemSound(kind)
	}
}
