package desktop_test

import (
	"context"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// trayRoute is the path from the function of the tray's state that starts the worker to the function that the worker's
// goroutine runs.
const trayRoute = "systemTrayState.DidChangeDependencies->systemTrayState.DidChangeDependencies.func1->"

// noBus is an opener whose bus never connects, as an app without a session bus.
func noBus(bc widgets.BuildContext) { desktop.UseBusOpener(bc, noBusOpener) }

// noBusOpener is the opener of noBus. It is a function with a name, so that its frame in a route does not depend on
// where the compiler inlines noBus.
func noBusOpener(context.Context) (dbus.Bus, error) {
	return nil, errors.New(tagDBusConnectFailed, "no bus")
}

// TestConnectFailureTakesTheTagOfTheService: a session bus that cannot be opened is the
// Unavailable tag of the service that asked, made in services.connect, with the bus's
// error as its cause.
func TestConnectFailureTakesTheTagOfTheService(t *testing.T) {
	tests := []struct {
		name  string
		call  func(bc widgets.BuildContext) error
		route string
		// cause is the route of the opener's error: the bus is opened on a goroutine of its own, and the route goes
		// on, through errors.Async, with the call that asked for the bus (rm4n0s/errors R7).
		cause string
	}{
		{"OpenFiles", func(bc widgets.BuildContext) error { _, err := desktop.OpenFiles(context.Background(), bc); return err },
			"call.chooseFiles->services.connect.FileChooserUnavailable",
			"call.chooseFiles->services.connect->services.connect.func1->noBusOpener.DBusConnectFailed"},
		{"ShowNotification", func(bc widgets.BuildContext) error {
			_, err := desktop.ShowNotification(context.Background(), bc, "s", desktop.NotificationOpts{})
			return err
		}, "call.show->services.connect.NotificationsUnavailable",
			"call.show->services.connect->services.connect.func1->noBusOpener.DBusConnectFailed"},
		{"CloseNotification", func(bc widgets.BuildContext) error { return desktop.CloseNotification(context.Background(), bc, 1) },
			"CloseNotification->services.connect.NotificationsUnavailable",
			"CloseNotification->services.connect->services.connect.func1->noBusOpener.DBusConnectFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			noBus(bc)
			err := inGoroutine(t, ts, func() error { return tt.call(bc) })
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if !errors.Is(err, &errors.Error{Tag: "DBusConnectFailed"}) {
				t.Errorf("the cause of %v is not DBusConnectFailed", err)
			}
			// errors.All reads the error and its cause: the service's error first, then the bus's own.
			chain := errors.All(err)
			if len(chain) != 2 || chain[0] != appErr || chain[1].Tag != tagDBusConnectFailed {
				t.Fatalf("errors.All gave %v, want the service's error and its cause", chain)
			}
			if !chain[1].HasRoute(tt.cause) {
				t.Errorf("unexpected failure path of the cause: %s", chain[1].Route())
			}
		})
	}
}

// TestConnectFailureOfTheTray: the tray reports TrayUnavailable made in services.connect.
func TestConnectFailureOfTheTray(t *testing.T) {
	ts, bc := app(t)
	noBus(bc)
	ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{}))
	var err error
	waitFor(t, ts, "the error", func() bool { err = ts.TakeError(); return err != nil })
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	// The worker of the tray runs on a goroutine of the state, and the bus is opened on another: errors.Async carries
	// the route across both, from the function of the state that made the tray (rm4n0s/errors R7).
	if !appErr.HasRoute(trayRoute + "tray.run->services.connect.TrayUnavailable") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	chain := errors.All(err)
	if len(chain) != 2 || chain[1].Tag != tagDBusConnectFailed {
		t.Fatalf("errors.All gave %v, want the tray's error and its cause", chain)
	}
	if !chain[1].HasRoute(trayRoute + "tray.run->services.connect->services.connect.func1->noBusOpener.DBusConnectFailed") {
		t.Errorf("unexpected failure path of the cause: %s", chain[1].Route())
	}
}

// TestConnectFailureWhileTheCallerWaits: a context that ends before the bus opens is
// the Unavailable tag of the service, made in services.connect.
func TestConnectFailureWhileTheCallerWaits(t *testing.T) {
	ts, bc := app(t)
	release := make(chan struct{})
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
		cancel() // the caller gives up while the bus is being opened
		<-release
		return nil, errors.New(tagDBusConnectFailed, "no bus")
	})
	err := inGoroutine(t, ts, func() error { _, err := desktop.OpenFiles(ctx, bc); return err })
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !errors.Is(err, &errors.Error{Tag: "FileChooserUnavailable"}) || !appErr.HasRoute("call.chooseFiles->services.connect.FileChooserUnavailable") {
		t.Errorf("got %s via %s", appErr.Tag, appErr.Route())
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDBusConnectFailed = "DBusConnectFailed"
)
