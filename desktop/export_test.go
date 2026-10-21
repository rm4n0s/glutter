package desktop

import (
	"context"

	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

// The seams that the tests of package desktop_test replace, for the app of bc.

// UseBus makes the app talk to bus instead of the session bus.
func UseBus(bc widgets.BuildContext, bus dbus.Bus) {
	s := servicesOf(widgets.WidgetsBindingOf(bc))
	s.mu.Lock()
	s.bus, s.web = bus, nil
	s.mu.Unlock()
}

// UseBusOpener makes the first connection to the session bus call open.
func UseBusOpener(bc widgets.BuildContext, open func(ctx context.Context) (dbus.Bus, error)) {
	s := servicesOf(widgets.WidgetsBindingOf(bc))
	s.mu.Lock()
	s.open, s.bus, s.web = open, nil, nil
	s.mu.Unlock()
}

// UseProgram makes the app start programs (xdg-open) with run.
func UseProgram(bc widgets.BuildContext, run func(ctx context.Context, env []string, name string, args ...string) error) {
	servicesOf(widgets.WidgetsBindingOf(bc)).run = run
}

// UseFlatpak makes the app believe that it runs, or does not run, in a Flatpak sandbox.
func UseFlatpak(bc widgets.BuildContext, in bool) {
	servicesOf(widgets.WidgetsBindingOf(bc)).flatpak = func() bool { return in }
}

// UseWeb makes the app run as on the web with the desktop d, whatever the platform is: URIs and
// notifications go to d, and the session bus is never tried.
func UseWeb(bc widgets.BuildContext, d webDesktop) {
	s := servicesOf(widgets.WidgetsBindingOf(bc))
	s.mu.Lock()
	s.bus, s.web = nil, d
	s.mu.Unlock()
}

// WebDesktop is webDesktop, for the fake of the tests.
type WebDesktop = webDesktop

// FilterName is filterName, for the tests.
var FilterName = filterName
