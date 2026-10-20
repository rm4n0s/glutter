package ui

// PortalHost is what the engine implements for the services of the desktop that
// take a window as their parent (the portals of xdg-desktop-portal): the handle of
// the window and the activation token of the app. A host without it gives none.
//
// Flutter: no counterpart
type PortalHost interface {
	// PortalParentHandle asks for the string that a portal wants as its parent
	// window: "wayland:<handle>" (zxdg_exporter_v2) or "x11:<window id in hex>".
	// done is called on the UI thread, with an empty string when the window has no
	// handle (the compositor cannot export it).
	PortalParentHandle(view ViewID, done func(handle string))
	// ActivationToken asks for a token that lets an app that this app starts take
	// the focus (XDG_ACTIVATION_TOKEN, and the activation_token of the portals). done
	// is called on the UI thread, with an empty string when there is none.
	ActivationToken(view ViewID, done func(token string))
	// AppID is the application ID of the app: the Wayland app_id and the X11
	// WM_CLASS, which must match its .desktop file. Notifications and the tray use
	// it to name the app to the desktop. It may be empty.
	AppID() string
}

// PortalParentHandle gives done the handle that the portals take as the parent of
// a dialog over the window of the view; see PortalHost. Without a host it is called
// at once with an empty handle.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) PortalParentHandle(view ViewID, done func(handle string)) {
	if h, ok := d.host.(PortalHost); ok {
		h.PortalParentHandle(view, done)
		return
	}
	done("")
}

// ActivationToken gives done an activation token for an app that this app starts
// from the window of the view; see PortalHost. Without a host it is called at once
// with an empty token.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) ActivationToken(view ViewID, done func(token string)) {
	if h, ok := d.host.(PortalHost); ok {
		h.ActivationToken(view, done)
		return
	}
	done("")
}

// AppID is the application ID of the app (RunAppOpts.AppID), or an empty string when
// the host does not know it.
//
// Flutter: no counterpart
func (d *PlatformDispatcher) AppID() string {
	if h, ok := d.host.(PortalHost); ok {
		return h.AppID()
	}
	return ""
}
