package desktop

import (
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

// InstallBus makes the app of the binding talk to bus instead of its platform's desktop:
// the session bus on Linux, and the desktop of macOS, Windows, the web, Android and iOS,
// which it drops, so the tray, the notifications, the file dialogs and OpenURI go to bus on
// every platform. It also starts no program: OpenURI's fallback to xdg-open fails, so a URI
// that bus does not open is OpenURIUnavailable and never reaches a browser of the machine.
// It is for gluttertest, which gives the tests a fake desktop
// (gluttertest.Tester.FakeDesktop); the bus type is internal, so an app cannot name it.
//
// Flutter: no counterpart
func InstallBus(b *widgets.WidgetsBinding, bus dbus.Bus) {
	s := servicesOf(b)
	s.mu.Lock()
	s.bus, s.web, s.android, s.windows = bus, nil, nil, nil
	s.run = startNoProgram
	s.mu.Unlock()
}

// startNoProgram is the program runner of a desktop that InstallBus installed: a test's
// desktop must not start xdg-open, which would open the developer's browser.
func startNoProgram(_ context.Context, _ []string, name string, _ ...string) error {
	// The caller tags the failure (OpenURIUnavailable); this sentinel is its cause, so that a
	// test can match ProgramNotStarted (SPEC §14.1).
	return &errors.Error{Tag: tagProgramNotStarted, Message: "a desktop installed with InstallBus starts no program",
		Metadata: map[string]any{"program": name}}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const tagProgramNotStarted = "ProgramNotStarted"
