package desktop

import (
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

// InstallBus makes the app of the binding talk to bus instead of its platform's desktop:
// the session bus on Linux, and the desktop of macOS, Windows, the web, Android and iOS,
// which it drops, so the tray, the notifications and the file dialogs go to bus on every
// platform. It is for gluttertest, which gives the tests a fake desktop
// (gluttertest.Tester.FakeDesktop); the bus type is internal, so an app cannot name it.
//
// Flutter: no counterpart
func InstallBus(b *widgets.WidgetsBinding, bus dbus.Bus) {
	s := servicesOf(b)
	s.mu.Lock()
	s.bus, s.web, s.android, s.windows = bus, nil, nil, nil
	s.mu.Unlock()
}
