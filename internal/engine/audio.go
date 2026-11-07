package engine

import "github.com/rm4n0s/glutter/internal/platform"

// Audio implements platform.AudioHost: the audio service of the platform, or nil when it has none (SPEC §28.3).
func (h *handler) Audio() platform.Audio {
	svc := h.conn.Services()
	if svc == nil {
		return nil
	}
	return svc.Audio()
}

// Sensors implements platform.SensorsHost: the motion sensors service of the platform, or nil when it has none (SPEC §28.4).
func (h *handler) Sensors() platform.Sensors {
	svc := h.conn.Services()
	if svc == nil {
		return nil
	}
	return svc.Sensors()
}

// AddReleaseHook implements platform.AudioHost and platform.SensorsHost: fn runs when the app ends, so that the audio devices that the app left open are
// closed before the process leaves. It is called on the UI thread.
func (h *handler) AddReleaseHook(fn func()) { h.releaseHooks = append(h.releaseHooks, fn) }

// runReleaseHooks runs the functions of AddReleaseHook once.
func (h *handler) runReleaseHooks() {
	hooks := h.releaseHooks
	h.releaseHooks = nil
	for _, fn := range hooks {
		fn()
	}
}
