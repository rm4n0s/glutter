//go:build darwin && !ios

package macos

import (
	"context"
	"log/slog"
	"sync"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/notify"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/internal/platform/darwin/openuri"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// desktop is the desktop services of macOS (SPEC §29.9): opening a URI with NSWorkspace, notifications through
// UserNotifications, the status item of the tray and the open and save panels. It is what DesktopProvider returns:
// it implements mobile.Services (open a URI, show and close a notification, as Android's and Windows' Desktop do)
// and the optional platform.TrayHost and platform.FileDialogHost, which package desktop finds by type assertion.
// OpenURI, ShowNotification, CloseNotification and ShowFileDialog may be called from any goroutine but the UI
// thread's; NewTray and the tray's methods belong to the UI thread.
//
// Flutter: no counterpart (Flutter apps use the url_launcher, flutter_local_notifications, tray_manager and
// file_selector plugins)
type desktop struct {
	p   *Platform
	log *slog.Logger
	// notes is the life of the notifications over the center (notify.go).
	notes *notify.Notifier
	// openURL opens a URL on the UI thread; the tests replace it, since opening opens a program.
	openURL func(uri string) (ok bool, reason string)

	// tray is the icon of the app, if it has one (UI thread).
	tray *trayIcon
}

var (
	_ mobile.Services          = (*desktop)(nil)
	_ platform.TrayHost        = (*desktop)(nil)
	_ platform.FileDialogHost  = (*desktop)(nil)
	_ platform.DesktopProvider = (*Platform)(nil)
)

// Desktop returns the desktop services of macOS (SPEC §29.9). It is made at the first call, which may come from any
// goroutine, and lives as long as the platform.
//
// Flutter: no counterpart
func (p *Platform) Desktop() mobile.Services {
	p.deskOnce.Do(func() {
		d := &desktop{p: p, log: p.log}
		d.notes = notify.New(notify.NewCenter(func() *notify.Notifier {
			if p.desk == nil {
				return nil
			}
			return p.desk.notes
		}))
		d.openURL = d.workspaceOpen
		p.desk = d
	})
	return p.desk
}

// DesktopName says which desktop the services are for: package desktop uses it in the error that says that D-Bus is
// not tried.
//
// Flutter: no counterpart
func (d *desktop) DesktopName() string { return "macOS" }

// workspaceOpen opens the URL with NSWorkspace: the program that macOS has for its scheme or its type.
func (d *desktop) workspaceOpen(uri string) (ok bool, reason string) {
	objcx.WithPool(func() {
		s := objcx.NewString(uri)
		defer objcx.Release(s)
		url := class("NSURL").Send(sel("URLWithString:"), s)
		if url == 0 {
			reason = "the text is not a URL that macOS can read"
			return
		}
		ws := class("NSWorkspace").Send(sel("sharedWorkspace"))
		if !send[bool](ws, "openURL:", url) {
			reason = "macOS has no program that opens the URL, or the program refused it"
			return
		}
		ok = true
	})
	return ok, reason
}

// OpenURI opens the URI with NSWorkspace (SPEC §29.9). The checks of openuri.Check come first, because opening a
// file: URI also launches programs. A URI that has no program, or that NSWorkspace refuses, is OpenURIUnavailable
// with the reason.
//
// Flutter: no counterpart (url_launcher_macos: launchUrl)
func (d *desktop) OpenURI(ctx context.Context, uri string) error {
	if err := openuri.Check(uri); err != nil {
		return err
	}
	type result struct {
		ok     bool
		reason string
	}
	done := make(chan result, 1)
	d.p.Post(func() {
		ok, reason := d.openURL(uri)
		done <- result{ok, reason}
	})
	select {
	case r := <-done:
		if !r.ok {
			return errors.New(tagOpenURIUnavailable, "macOS cannot open the URI", "uri", uri, "reason", r.reason)
		}
		return nil
	case <-ctx.Done():
		return errors.NewErr(tagOpenURIUnavailable, ctx.Err()).SetMetadata("uri", uri, "reason", "the context ended before macOS answered")
	}
}

// ShowNotification shows the notification through UserNotifications (SPEC §29.9): the permission is asked at the
// first one, and an app without a bundle that has an identifier (it runs with `go run`) is NotificationsUnavailable.
//
// Flutter: no counterpart (flutter_local_notifications_darwin)
func (d *desktop) ShowNotification(ctx context.Context, spec mobile.NotificationSpec) (uint32, error) {
	return d.notes.Show(ctx, spec)
}

// CloseNotification takes the notification off the screen and calls its OnClose with CloseByApp. Closing one that is
// gone is not an error.
//
// Flutter: no counterpart
func (d *desktop) CloseNotification(_ context.Context, id uint32) error {
	d.notes.Close(id)
	return nil
}

// closeServices ends the services with the platform: the tray icon goes, the timers stop.
func (d *desktop) closeServices() {
	d.notes.Shutdown()
	if d.tray != nil {
		d.tray.Close()
	}
}

// deskState are the fields of the platform for the services.
type deskState struct {
	deskOnce sync.Once
	desk     *desktop
}

// The tag of the error that this file makes (rm4n0s/errors, R5); package desktop matches it by value.
const tagOpenURIUnavailable = "OpenURIUnavailable"
