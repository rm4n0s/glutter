package desktop

import (
	"context"
	"net/url"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/dbus"
	"github.com/rm4n0s/glutter/widgets"
)

const openURIMethod = "org.freedesktop.portal.OpenURI.OpenURI"

// OpenURI opens a URI with the application that the desktop chooses for it: a web
// page in the browser, a file in its viewer, a mailto: link in the mail program. It
// goes through the OpenURI portal (org.freedesktop.portal.OpenURI), which makes the
// app chooser of the desktop the parent of the window of bc, and falls back to
// xdg-open when there is no portal. Without either it fails with OpenURIUnavailable.
// If the user cancels the portal's app chooser, OpenURI returns nil.
//
// On the web it is window.open(uri, "_blank", "noopener"): the browser lets a page open a window only
// soon after a click or a key press, so call it from the handler of one (on a goroutine, as always); a
// javascript: or vbscript: URI fails with OpenURISchemeBlocked, and the browser's pop-up blocker is not
// reported.
//
// It waits for the desktop, so it must be called from a goroutine and not on the UI
// thread (it fails with DesktopCallOnUIThread there); ctx ends the wait. A URI without
// a scheme fails with OpenURIInvalid.
//
// Flutter: no counterpart (Flutter apps use the url_launcher plugin)
func OpenURI(ctx context.Context, bc widgets.BuildContext, uri string) error {
	if err := validateURI(uri); err != nil {
		return err
	}
	c, err := begin(ctx, bc)
	if err != nil {
		return err
	}
	if c.s.web != nil {
		return c.s.web.OpenURI(ctx, uri)
	}
	return c.openURI(ctx, uri)
}

// validateURI checks that the text is a URI with a scheme.
func validateURI(uri string) error {
	if strings.ContainsAny(uri, "\x00\r\n") {
		return errors.New("OpenURIInvalid", "a URI cannot hold a control character", "uri", uri)
	}
	u, err := url.Parse(uri)
	if err != nil {
		return errors.NewErr("OpenURIInvalid", err).SetMetadata("uri", uri)
	}
	if u.Scheme == "" {
		return errors.New("OpenURIInvalid", "a URI needs a scheme, such as https: or file:", "uri", uri)
	}
	return nil
}

// openURI tries the portal, then xdg-open.
func (c *call) openURI(ctx context.Context, uri string) error {
	portalErr := c.openURIWithPortal(ctx, uri)
	if portalErr == nil {
		return nil
	}
	if ctx.Err() != nil {
		return portalErr
	}
	var env []string
	if c.token != "" {
		env = append(env, "XDG_ACTIVATION_TOKEN="+c.token)
	}
	if err := c.s.run(ctx, env, "xdg-open", uri); err != nil {
		portalTag := ""
		if pe, ok := errors.FromError(portalErr); ok {
			portalTag = pe.Tag
		}
		return errors.NewErr("OpenURIUnavailable", err).
			SetMetadata("uri", uri, "portalTag", portalTag, "portal", portalErr.Error())
	}
	return nil
}

// openURIWithPortal asks the OpenURI portal and waits for its answer. A user who
// cancels is not an error. A portal that cannot do it fails with the error of the
// bus, or with OpenURIFailed when it answers that the interaction ended badly.
func (c *call) openURIWithPortal(ctx context.Context, uri string) error {
	bus, err := c.s.connect(ctx)
	if err != nil {
		return err
	}
	resp, err := dbus.PortalRequest(ctx, bus, openURIMethod, func(token string) []any {
		return []any{c.handle, uri, c.portalOptions(token, nil)}
	})
	if err != nil {
		return err
	}
	switch resp.Code {
	case dbus.PortalResponseSuccess, dbus.PortalResponseCancelled:
		return nil
	}
	return errors.New("OpenURIFailed", "the OpenURI portal could not open the URI", "uri", uri, "code", uint32(resp.Code))
}
