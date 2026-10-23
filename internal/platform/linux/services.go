//go:build linux && !android

package linux

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/atspi"
	"github.com/rm4n0s/glutter/internal/platform"
)

// Services are the optional capabilities that the Linux backends share: the
// accessibility bridge (AT-SPI2 over D-Bus, SPEC §8.10).
//
// Flutter: no counterpart
type Services struct {
	bridge *atspi.Bridge
}

// NewServices makes the services of a backend. post runs a function on its UI thread.
// The bridge connects to the session bus only when it is started, and only if a bus is
// there: it never starts one.
//
// Flutter: no counterpart
func NewServices(post func(func())) *Services {
	return &Services{bridge: atspi.New(atspi.Options{
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
