package engine

import (
	"log/slog"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/pads"
	"github.com/rm4n0s/glutter/ui"
)

// GamepadsEnv is the environment variable that sets the gamepad mode of a run: "auto", "ondemand" or "off". It overrides
// Config.Gamepads, so a user can turn the pads off for one run of any app (SPEC §28.2, "Decided in step 36.1").
const GamepadsEnv = "GLUTTER_GAMEPADS"

// The modes of Config.Gamepads.
const (
	// GamepadsAuto (the empty value is the same) starts the gamepad service with the app, so that a pad moves the focus of every
	// app, but the service only reads: nothing is written to a pad until the app uses package gamepad.
	GamepadsAuto = "auto"
	// GamepadsOnDemand starts nothing until the app first uses package gamepad; until then there are no pad key events.
	GamepadsOnDemand = "ondemand"
	// GamepadsOff never starts the service: there are no pads and no pad key events.
	GamepadsOff = "off"
)

// gamepadModeOf finds the mode of a run: the environment variable wins over the option, and a value of it that is no mode is
// logged and ignored (the option decides). An option that is no mode fails with GamepadsModeUnknown.
func gamepadModeOf(option string, getenv func(string) string, log *slog.Logger) (string, error) {
	mode := GamepadsAuto
	switch option {
	case "", GamepadsAuto, GamepadsOnDemand, GamepadsOff:
		if option != "" {
			mode = option
		}
	default:
		return "", errors.New(tagGamepadsModeUnknown, "the gamepad mode is not auto, ondemand or off", "mode", option)
	}
	switch v := getenv(GamepadsEnv); v {
	case "":
	case GamepadsAuto, GamepadsOnDemand, GamepadsOff:
		mode = v
	default:
		log.Info("the gamepad mode of the environment is not auto, ondemand or off, so it is ignored", "variable", GamepadsEnv, "value", v)
	}
	return mode, nil
}

// startGamepads makes the pad manager when the platform has a gamepad service and the mode allows it (SPEC §28.2). In the
// default mode it also starts it: the pads make key events (the buttons as game-button keys and the D-pad as arrows), which give
// every app focus traversal and activation with a pad, whether or not the app uses package gamepad. The service only reads
// until the app uses the package (UsePads). The mode that waits for the app leaves the manager unstarted, and "off" makes none.
func (h *handler) startGamepads() {
	if h.padMode == GamepadsOff {
		h.log.Info("the gamepads are off", "variable", GamepadsEnv)
		return
	}
	svc := h.conn.Services()
	if svc == nil {
		return
	}
	service := svc.Gamepads()
	if service == nil {
		return
	}
	getenv := h.cfg.Getenv
	h.pads = pads.New(service, pads.Opts{Keys: h.padKey, Logger: h.log, Getenv: getenv})
	if h.padMode == GamepadsOnDemand {
		return
	}
	if err := h.pads.Start(); err != nil {
		h.log.Info("there are no gamepads", "err", err)
		h.pads = nil
	}
}

// UsePads implements pads.Host: package gamepad is used, so the manager that waited starts and the extras of the service wake.
func (h *handler) UsePads() {
	if h.pads == nil {
		return
	}
	if err := h.pads.Wake(); err != nil {
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
