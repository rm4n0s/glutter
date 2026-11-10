//go:build linux && !android

package linux

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/atspi"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/evdev"
)

// Services are the optional capabilities that the Linux backends share: the
// accessibility bridge (AT-SPI2 over D-Bus, SPEC §8.10), the gamepads (evdev, SPEC §28.2) and the audio (PulseAudio and ALSA, SPEC §28.3).
//
// Flutter: no counterpart
type Services struct {
	bridge *atspi.Bridge
	pads   *evdev.Service
	audio  *audioChoice
}

// NewServices makes the services of a backend. post runs a function on its UI thread.
// The bridge connects to the session bus only when it is started, and only if a bus is
// there: it never starts one.
//
// GLUTTER_GAMEPAD_HID=0 turns the HID drivers of the gamepads off, so that every pad is read through its event device (SPEC §28.2).
// The gamepad service starts asleep: it only reads the pads, until the app uses package gamepad (platform.PadExtras).
//
// Flutter: no counterpart
func NewServices(post func(func())) *Services {
	return &Services{audio: audioFromEnv(), pads: evdev.New(evdev.Opts{Post: post, Logger: slog.Default(), DisableHID: os.Getenv("GLUTTER_GAMEPAD_HID") == "0", Asleep: true}), bridge: atspi.New(atspi.Options{
		AppName: filepath.Base(os.Args[0]),
		Post:    post,
		OpenSession: func(ctx context.Context) (dbus.Bus, error) {
			if !dbus.SessionBusPresent(os.Getenv) {
				return nil, errors.New(tagNoSessionBus, "there is no session bus, and an app does not start one")
			}
			return dbus.Connect(ctx)
		},
	})}
}

// Accessibility returns the AT-SPI2 bridge, or nil when there are no services.
func (s *Services) Accessibility() platform.Accessibility {
	if s == nil {
		return nil
	}
	return bridgeAdapter{s.bridge}
}

// Gamepads returns the evdev gamepad service.
func (s *Services) Gamepads() platform.Gamepads {
	if s == nil || s.pads == nil {
		return nil
	}
	return s.pads
}

// Audio returns the audio service: PulseAudio, then ALSA (SPEC §28.3). Nothing is loaded or connected until it is used.
func (s *Services) Audio() platform.Audio {
	if s == nil || s.audio == nil {
		return nil
	}
	return s.audio
}

// Sensors returns nil: the Linux desktop has no motion sensors service (SPEC §28.4); a pad's sensors are the gamepad service's.
func (s *Services) Sensors() platform.Sensors { return nil }

// bridgeAdapter gives the bridge the types of the platform interface.
type bridgeAdapter struct{ *atspi.Bridge }

// Start follows the screen reader in the background.
func (a bridgeAdapter) Start(ctx context.Context, host platform.AccessibilityHost) {
	a.Bridge.Start(ctx, host)
}

var _ platform.Services = (*Services)(nil)

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagNoSessionBus = "NoSessionBus"
)
