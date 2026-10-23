//go:build linux

package linux

import (
	"context"
	"log/slog"
	"os"
	"reflect"
	"time"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/platform"
)

// busTimeout is how long the connection to the session bus may take. A bus that
// is there answers at once.
const busTimeout = 3 * time.Second

// SettingsWatcher reads the system settings for a backend and follows their
// changes (SPEC §8.8): from the Settings portal over the session bus, else from the
// environment. Backends add their own source, such as XSETTINGS on X11, with Set.
// Every method belongs to the UI thread.
//
// Flutter: no counterpart
type SettingsWatcher struct {
	// Post runs a function on the UI thread (Platform.Post).
	Post func(func())
	// Apply is called on the UI thread with the settings after each change.
	Apply func(platform.Settings)
	// Open connects to the session bus; nil means dbus.Connect when a session bus is
	// there. Tests replace it.
	Open func(ctx context.Context) (dbus.Bus, error)
	// Getenv reads the environment; nil means os.Getenv.
	Getenv func(string) string

	cancel     context.CancelFunc
	bus        dbus.Bus
	current    platform.Settings
	fromPortal bool
	started    bool
}

// Start connects to the portal and reads the settings once; later calls return the
// settings now. fromPortal says that the portal answered, so that the backend
// knows it need not look elsewhere.
func (w *SettingsWatcher) Start() (settings platform.Settings, fromPortal bool) {
	if w.started {
		return w.current, w.fromPortal
	}
	w.started = true
	getenv := w.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	var bus dbus.Bus
	open := w.Open
	if open == nil && sessionBusPresent(getenv) {
		open = dbus.Connect
	}
	if open != nil {
		openCtx, openCancel := context.WithTimeout(ctx, busTimeout)
		b, err := open(openCtx)
		openCancel()
		if err != nil {
			slog.Debug("no session bus for the system settings", "err", err)
		} else {
			bus = b
		}
	}
	w.bus = bus
	w.current, w.fromPortal = dbus.WatchPortalSettings(ctx, bus, getenv, w.Post, func(s platform.Settings) {
		w.current = s
		w.Apply(s)
	})
	return w.current, w.fromPortal
}

// Current returns the settings now.
func (w *SettingsWatcher) Current() platform.Settings { return w.current }

// Set replaces the settings, for a source other than the portal, and passes them on
// when they changed.
func (w *SettingsWatcher) Set(s platform.Settings) {
	if reflect.DeepEqual(w.current, s) {
		return
	}
	w.current = s
	w.Apply(s)
}

// Replace sets the settings without passing them on, for the first reading of a
// source other than the portal.
func (w *SettingsWatcher) Replace(s platform.Settings) { w.current = s }

// Stop stops following and closes the bus.
func (w *SettingsWatcher) Stop() {
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	if w.bus != nil {
		if err := w.bus.Close(); err != nil {
			slog.Debug("closing the session bus failed", "err", err)
		}
		w.bus = nil
	}
}

// sessionBusPresent is whether a session bus can be reached without starting one.
func sessionBusPresent(getenv func(string) string) bool { return dbus.SessionBusPresent(getenv) }
