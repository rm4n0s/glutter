package desktop

import (
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

// InstallBus makes the app of the binding talk to bus instead of the session bus. It
// is for gluttertest, which gives the tests a fake desktop (gluttertest.Tester.FakeDesktop);
// the bus type is internal, so an app cannot name it.
//
// Flutter: no counterpart
func InstallBus(b *widgets.WidgetsBinding, bus dbus.Bus) {
	s := servicesOf(b)
	s.mu.Lock()
	s.bus, s.web, s.android, s.windows = bus, nil, nil, nil
	s.mu.Unlock()
}
