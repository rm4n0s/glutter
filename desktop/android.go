package desktop

import (
	"context"

	"github.com/rm4n0s/glutter/internal/platform/mobile"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/widgets"
)

// androidDesktop is what package desktop asks of Android (SPEC §24.11): the platform's own
// services, which the engine hands out through the dispatcher (mobile.Services, made by
// internal/platform/android over intents and the notification manager). The tests of the package
// replace it with a fake. The tray, the file chooser and the session bus do not exist on Android, and
// D-Bus is never tried there.
type androidDesktop interface {
	OpenURI(ctx context.Context, uri string) error
	ShowNotification(ctx context.Context, spec mobile.NotificationSpec) (uint32, error)
	CloseNotification(ctx context.Context, id uint32) error
}

// newAndroidDesktop returns the services of the Android platform that runs the binding, or nil on
// every other platform. It asks the engine through the dispatcher and does not look at the target
// platform, which a test can set to Android on any machine.
func newAndroidDesktop(b *widgets.WidgetsBinding) androidDesktop {
	svc, _ := b.PlatformDispatcher().PlatformService(mobile.ServiceDesktop).(androidDesktop)
	return svc
}

// androidReason is why the service behind tag does not exist on Android.
func androidReason(tag string) string {
	switch tag {
	case "TrayUnavailable":
		return "Android has no system tray: D-Bus is never tried on Android"
	case "FileChooserUnavailable":
		return "the file chooser is not available on Android: the Storage Access Framework gives content: URIs, not paths, and D-Bus is never tried on Android"
	}
	return "Android has no session bus: D-Bus is never tried on Android"
}

// showOnAndroid posts the notification through the notification manager. A notification there has a
// title, a plain-text body, buttons, a click and a close: the action with the ID "default" is the click
// on the notification (not a button), the other actions are buttons (the system shows three), Low
// urgency is the silent channel and Critical the one that shows over the screen, Persistent and
// Critical notifications have no timeout and stay until the user acts, and Timeout is kept by the
// program. The icon (a theme name or a file the system cannot read) and the Category and AppName have
// nowhere to go: the notification wears the app's icon and name.
func (c *call) showOnAndroid(ctx context.Context, summary string, o NotificationOpts) (NotificationID, error) {
	return c.showOnMobile(ctx, c.s.android, summary, o, false)
}

// showOnMobile is showOnAndroid for the platform services svc, which are Android's or Windows': a
// notification there is a title, a plain-text body, actions, an urgency, a time-out and two callbacks.
// withIcon passes the app's Icon on, for the platform that has a place for it (a balloon of Windows).
func (c *call) showOnMobile(ctx context.Context, svc androidDesktop, summary string, o NotificationOpts, withIcon bool) (NotificationID, error) {
	body := o.Body
	if o.BodyMarkup {
		body = web.StripMarkup(body)
	}
	spec := mobile.NotificationSpec{Title: summary, Body: body, ReplacesID: uint32(o.ReplacesID)}
	if withIcon {
		spec.Icon = o.Icon
	}
	switch o.Urgency {
	case NotificationUrgencyLow:
		spec.Urgency = mobile.UrgencyLow
	case NotificationUrgencyCritical:
		spec.Urgency = mobile.UrgencyCritical
	}
	if !o.Persistent && o.Urgency != NotificationUrgencyCritical {
		spec.Timeout = o.Timeout
	}
	for _, a := range o.Actions {
		spec.Actions = append(spec.Actions, mobile.Action{ID: a.ID, Label: a.Label})
	}
	s := c.s
	if onAction := o.OnAction; onAction != nil {
		spec.OnAction = func(id string) { s.postHandler("action", func() { onAction(id) }) }
	}
	if onClosed := o.OnClosed; onClosed != nil {
		spec.OnClose = func(r mobile.CloseReason) {
			s.postHandler("close", func() { onClosed(NotificationCloseReason(r)) })
		}
	}
	id, err := svc.ShowNotification(ctx, spec)
	if err != nil {
		return 0, err
	}
	return NotificationID(id), nil
}
