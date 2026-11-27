package desktop_test

import (
	"context"
	"testing"

	godbus "github.com/godbus/dbus/v5"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/internal/dbus/fakebus"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const nameHasOwnerM = "org.freedesktop.DBus.NameHasOwner"

// TestTrayAvailable: the answer of each platform, and on Linux whether a StatusNotifierWatcher runs on the session
// bus (the bus answers NameHasOwner for it); without a session bus the answer is no, not an error.
func TestTrayAvailable(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, bc widgets.BuildContext)
		want  bool
	}{
		{"Linux with a watcher", func(t *testing.T, bc widgets.BuildContext) {
			b := fakebus.New(unique)
			watcherOn(b)
			desktop.UseBus(bc, b)
		}, true},
		{"Linux without a watcher", func(t *testing.T, bc widgets.BuildContext) { desktop.UseBus(bc, fakebus.New(unique)) }, false},
		{"Linux without a session bus", func(t *testing.T, bc widgets.BuildContext) { noBus(bc) }, false},
		{"Windows", func(t *testing.T, bc widgets.BuildContext) { onWindows(t, bc) }, true},
		{"macOS", func(t *testing.T, bc widgets.BuildContext) {
			desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
				t.Error("macOS tried the session bus")
				return nil, errors.New(tagDBusConnectFailed, "must not be called")
			})
			desktop.UseWindows(bc, &macFake{})
		}, true},
		{"Android", func(t *testing.T, bc widgets.BuildContext) { onAndroid(t, bc) }, false},
		{"iOS", func(t *testing.T, bc widgets.BuildContext) {
			desktop.UseBusOpener(bc, func(context.Context) (dbus.Bus, error) {
				t.Error("iOS tried the session bus")
				return nil, errors.New(tagDBusConnectFailed, "must not be called")
			})
			desktop.UseAndroid(bc, &fakeIOS{})
		}, false},
		{"the web", func(t *testing.T, bc widgets.BuildContext) { onTheWeb(t, bc) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			tt.setup(t, bc)
			var got bool
			err := inGoroutine(t, ts, func() error {
				var err error
				got, err = desktop.TrayAvailable(context.Background(), bc)
				return err
			})
			if err != nil {
				t.Fatalf("TrayAvailable: %v", err)
			}
			if got != tt.want {
				t.Errorf("TrayAvailable = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestTrayAvailableAgreesWithTheTray: after a no, the tray reports TrayUnavailable, which an app matches with the
// exported tag; after a yes, the tray registers with the watcher and reports nothing.
func TestTrayAvailableAgreesWithTheTray(t *testing.T) {
	for _, withWatcher := range []bool{true, false} {
		name := "no watcher"
		if withWatcher {
			name = "a watcher"
		}
		t.Run(name, func(t *testing.T) {
			ts, bc := app(t)
			b := fakebus.New(unique)
			if withWatcher {
				watcherOn(b)
			}
			desktop.UseBus(bc, b)
			var available bool
			if err := inGoroutine(t, ts, func() error {
				var err error
				available, err = desktop.TrayAvailable(context.Background(), bc)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if available != withWatcher {
				t.Fatalf("TrayAvailable = %v with a watcher %v", available, withWatcher)
			}
			ts.PumpWidget(trayTree(ui.WindowIcon{Name: "x"}, desktop.SystemTrayOpts{}))
			if available {
				waitFor(t, ts, "the registration", func() bool { return len(registrations(b)) == 1 })
				if err := ts.TakeError(); err != nil {
					t.Errorf("the tray reported %v after a yes", err)
				}
				return
			}
			var err error
			waitFor(t, ts, "the error", func() bool { err = ts.TakeError(); return err != nil })
			if !errors.Is(err, &errors.Error{Tag: desktop.TagTrayUnavailable}) {
				t.Errorf("the tray reported %v, which does not match TagTrayUnavailable", err)
			}
			if appErr, ok := errors.FromError(err); !ok || appErr.Tag != desktop.TagTrayUnavailable {
				t.Errorf("the tray reported %v, want the tag %s", err, desktop.TagTrayUnavailable)
			}
		})
	}
}

// failingNameHasOwner is a bus that cannot answer NameHasOwner. It is a function with a name, so that its frame in a
// route does not depend on inlining.
func failingNameHasOwner(fakebus.Call) ([]any, error) {
	return nil, errors.NewErr(tagDBusCallFailed, &godbus.Error{Name: "org.freedesktop.DBus.Error.AccessDenied"})
}

// blockedOpener is a session bus that never connects while the test runs.
func blockedOpener(release <-chan struct{}) func(context.Context) (dbus.Bus, error) {
	return func(context.Context) (dbus.Bus, error) {
		<-release
		return nil, errors.New(tagDBusConnectFailed, "released")
	}
}

// TestTrayAvailableErrors: each failure, with its route.
func TestTrayAvailableErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, bc widgets.BuildContext)
		ctx   func() context.Context
		route string
		check func(t *testing.T, err error)
	}{
		{"the bus cannot answer", func(t *testing.T, bc widgets.BuildContext) {
			b := fakebus.New(unique)
			b.Handle("org.freedesktop.DBus", "/org/freedesktop/DBus", nameHasOwnerM, failingNameHasOwner)
			desktop.UseBus(bc, b)
		}, context.Background, "TrayAvailable->Bus.Call->failingNameHasOwner." + tagDBusCallFailed, nil},
		{"an answer that is not a boolean", func(t *testing.T, bc widgets.BuildContext) {
			b := fakebus.New(unique)
			b.Handle("org.freedesktop.DBus", "/org/freedesktop/DBus", nameHasOwnerM, func(fakebus.Call) ([]any, error) {
				return []any{"yes"}, nil
			})
			desktop.UseBus(bc, b)
		}, context.Background, "TrayAvailable.TrayAvailabilityInvalid", func(t *testing.T, err error) {
			if appErr, _ := errors.FromError(err); appErr.Metadata["type"] != "string" {
				t.Errorf("metadata %v", appErr.Metadata)
			}
		}},
		{"an answer with no value", func(t *testing.T, bc widgets.BuildContext) {
			b := fakebus.New(unique)
			b.Handle("org.freedesktop.DBus", "/org/freedesktop/DBus", nameHasOwnerM, func(fakebus.Call) ([]any, error) {
				return nil, nil
			})
			desktop.UseBus(bc, b)
		}, context.Background, "TrayAvailable.TrayAvailabilityInvalid", nil},
		{"the context ends while the bus connects", func(t *testing.T, bc widgets.BuildContext) {
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			desktop.UseBusOpener(bc, blockedOpener(release))
		}, func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}, "TrayAvailable->services.connect.TrayUnavailable", func(t *testing.T, err error) {
			if !errors.Is(err, context.Canceled) {
				t.Errorf("the cause of %v is not the context's error", err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, bc := app(t)
			tt.setup(t, bc)
			ctx := tt.ctx()
			err := inGoroutine(t, ts, func() error {
				got, err := desktop.TrayAvailable(ctx, bc)
				if got {
					t.Error("TrayAvailable answered yes with an error")
				}
				return err
			})
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
			if tt.check != nil {
				tt.check(t, err)
			}
		})
	}
}

func TestTrayAvailableOnTheUIThreadFails(t *testing.T) {
	_, bc := app(t)
	_, err := desktop.TrayAvailable(context.Background(), bc)
	appErr := tagged(t, err, "DesktopCallOnUIThread")
	if !appErr.HasRoute("TrayAvailable.DesktopCallOnUIThread") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

func TestTrayAvailableWithoutABinding(t *testing.T) {
	_, err := desktop.TrayAvailable(context.Background(), nil)
	appErr := tagged(t, err, "DesktopWithoutBinding")
	if !appErr.HasRoute("TrayAvailable.DesktopWithoutBinding") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagDBusCallFailed = "DBusCallFailed"
)
