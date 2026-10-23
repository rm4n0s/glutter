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

// noBus is an opener whose bus never connects, as an app without a session bus.
func noBus(bc widgets.BuildContext) {
	desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) { return nil, errors.New(tagDBusConnectFailed, "no bus") })
}

// TestConnectFailureTakesTheTagOfTheService: a session bus that cannot be opened is the
// Unavailable tag of the service that asked, made in services.connect, with the bus's
// error as its cause.
func TestConnectFailureTakesTheTagOfTheService(t *testing.T) {
	tests := []struct {
		name  string
		call  func(bc widgets.BuildContext) error
		route string
	}{
		{"OpenFiles", func(bc widgets.BuildContext) error { _, err := desktop.OpenFiles(context.Background(), bc); return err },
			"call.chooseFiles->services.connect.FileChooserUnavailable"},
		{"ShowNotification", func(bc widgets.BuildContext) error {
			_, err := desktop.ShowNotification(context.Background(), bc, "s", desktop.NotificationOpts{})
			return err
		}, "call.show->services.connect.NotificationsUnavailable"},
		{"CloseNotification", func(bc widgets.BuildContext) error { return desktop.CloseNotification(context.Background(), bc, 1) },
			"CloseNotification->services.connect.NotificationsUnavailable"},
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
	if !appErr.HasRoute("tray.run->services.connect.TrayUnavailable") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
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
