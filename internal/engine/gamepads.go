package engine

import (
	"time"

	"github.com/rm4n0s/glutter/internal/pads"
	"github.com/rm4n0s/glutter/ui"
)

// startGamepads starts the pad manager when the platform has a gamepad service (SPEC §28.2). It runs for the app whether or
// not the app uses package gamepad, because the pads make key events (the buttons as game-button keys and the D-pad as
// arrows), which give every app focus traversal and activation with a pad.
func (h *handler) startGamepads() {
	svc := h.conn.Services()
	if svc == nil {
		return
	}
	service := svc.Gamepads()
	if service == nil {
		return
	}
	h.pads = pads.New(service, pads.Opts{Keys: h.padKey, Logger: h.log, Getenv: h.cfg.Getenv})
	if err := h.pads.Start(); err != nil {
		h.log.Info("there are no gamepads", "err", err)
		h.pads = nil
	}
}

// PadManager implements pads.Host: the manager of the pads of the app, or nil when the platform has none.
func (h *handler) PadManager() *pads.Manager { return h.pads }

// padKey gives the framework the key event that a pad made. A pad is read whether or not the app is in front, so only the
// window that has the focus gets the keys: an app that is behind others does not press its buttons for a pad that someone
// else uses.
func (h *handler) padKey(k pads.Key) {
	w := h.windowOf(h.focusView)
	if w == nil || !w.focused {
		return
	}
	typ := ui.KeyEventTypeUp
	if k.Down {
		typ = ui.KeyEventTypeDown
	}
	dev := ui.KeyEventDeviceTypeGamepad
	if k.DirectionalPad {
		dev = ui.KeyEventDeviceTypeDirectionalPad
	}
	t := k.Time
	if t == 0 {
		t = time.Since(h.runStart)
	}
	d := ui.KeyData{TimeStamp: t, Type: typ, DeviceType: dev, Physical: k.Physical, Logical: k.Logical}
	h.queueInput(w, func() { h.dispatchKey(d) })
}
