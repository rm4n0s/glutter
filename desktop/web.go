package desktop

import (
	"context"
	"slices"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform/web"
)

// webDesktop is what package desktop asks of the browser on the web (SPEC §23.7): the web
// platform's Desktop, which is plain Go over window.open and the Notifications API. The tests of
// the package replace it with a fake. The tray, the file chooser and the session bus do not exist
// on the web, and D-Bus is never tried there.
type webDesktop interface {
	OpenURI(ctx context.Context, uri string) error
	ShowNotification(ctx context.Context, spec web.NotificationSpec) (uint32, error)
	CloseNotification(ctx context.Context, id uint32) error
}

// newWebDesktop returns the desktop of the web page, or nil off the web.
func newWebDesktop() webDesktop {
	if !foundation.IsWeb {
		return nil
	}
	return web.NewDesktop()
}

// showOnWeb shows the notification with the Notifications API. A notification on the web has a title, a
// plain-text body, an icon that is a URL, and a click: the actions become the click when one has the ID
// "default" (the buttons of an action need a service worker, which a Glutter page does not have), the
// Category and AppName have nowhere to go (the browser names the page), Low urgency is silent, and
// Critical and Persistent notifications stay until the user acts. Timeout is kept by the page.
func (c *call) showOnWeb(ctx context.Context, summary string, o NotificationOpts) (NotificationID, error) {
	body := o.Body
	if o.BodyMarkup {
		body = web.StripMarkup(body)
	}
	spec := web.NotificationSpec{
		Title: summary, Body: body, Icon: web.IconURL(o.Icon),
		RequireInteraction: o.Persistent || o.Urgency == NotificationUrgencyCritical,
		Silent:             o.Urgency == NotificationUrgencyLow,
		ReplacesID:         uint32(o.ReplacesID),
	}
	if !o.Persistent {
		spec.Timeout = o.Timeout
	}
	s := c.s
	if onAction := o.OnAction; onAction != nil && slices.ContainsFunc(o.Actions, func(a NotificationAction) bool { return a.ID == "default" }) {
		spec.OnClick = func() { s.postHandler("action", func() { onAction("default") }) }
	}
	if onClosed := o.OnClosed; onClosed != nil {
		spec.OnClose = func(r web.CloseReason) {
			s.postHandler("close", func() { onClosed(NotificationCloseReason(r)) })
		}
	}
	id, err := s.web.ShowNotification(ctx, spec)
	if err != nil {
		return 0, err
	}
	return NotificationID(id), nil
}
