package wayland

import (
	"os"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/linux/wayland/wl"
	"github.com/rm4n0s/glutter/ui"
)

// activationTokenEnv is the variable in which a launcher passes an activation
// token to the app it starts (xdg-activation-v1).
const activationTokenEnv = "XDG_ACTIVATION_TOKEN"

var _ platform.ActivationTokenProvider = (*Platform)(nil)

// takeStartupToken reads the activation token that the launcher passed in the
// environment and removes it from there, so that the apps this one starts do not
// inherit it (the token works once).
func (p *Platform) takeStartupToken() {
	p.startupToken = os.Getenv(activationTokenEnv)
	if p.startupToken != "" {
		_ = os.Unsetenv(activationTokenEnv) // nothing to do if it fails: the token is only used once here
	}
}

// useStartupToken activates the first window that the compositor configured with
// the token of the launcher, which ends the startup feedback of the launcher
// (SPEC §8.11). Later windows do nothing.
func (p *Platform) useStartupToken(w *window) {
	token := p.startupToken
	if token == "" {
		return
	}
	p.startupToken = ""
	if act := p.sess.activation; act != nil {
		act.Activate(token, w.surface)
	}
}

// RequestActivationToken asks the compositor for an activation token for the
// window of the view, with the serial of the last input event of the seat and the
// app id of the window: the proof that a user action is behind it. done gets the
// token on the UI thread, or an empty token when there is no such window or the
// compositor has no xdg_activation_v1. The token goes to an app that this app
// starts, in XDG_ACTIVATION_TOKEN, or to activate a window of this app.
//
// Flutter: no counterpart
func (p *Platform) RequestActivationToken(view ui.ViewID, done func(token string)) {
	w, ok := p.windows[view]
	act := p.sess.activation
	if !ok || act == nil {
		done("")
		return
	}
	tok := act.GetActivationToken()
	tok.SetEvents(&wl.XdgActivationTokenV1Events{OnDone: func(token string) {
		tok.Destroy()
		done(token)
	}})
	if p.inputSerial != 0 {
		tok.SetSerial(p.inputSerial, p.sess.seat)
	}
	if w.cfg.AppID != "" {
		tok.SetAppId(w.cfg.AppID)
	}
	tok.SetSurface(w.surface)
	tok.Commit()
}

// activate asks the compositor to show the window and give it the focus, with a
// token for it: xdg-shell has no request for it. The compositor decides; it may
// only mark the window as wanting attention.
func (w *window) activate() {
	if w.p.sess.activation == nil {
		return
	}
	w.p.RequestActivationToken(w.id, func(token string) {
		if token != "" && !w.closed {
			w.p.sess.activation.Activate(token, w.surface)
		}
	})
}

// KeepScreenOn stops the screen from going idle while the window is visible
// (zwp_idle_inhibit_manager_v1: the compositor lifts the inhibitor when the
// surface is hidden). It fails with IdleInhibitUnavailable when the compositor
// has no such protocol.
//
// Flutter: no counterpart
func (w *window) KeepScreenOn(on bool) error {
	if !on {
		if w.inhibitor != nil {
			w.inhibitor.Destroy()
			w.inhibitor = nil
		}
		return nil
	}
	if w.inhibitor != nil {
		return nil
	}
	mgr := w.p.sess.idleInhibit
	if mgr == nil {
		return errors.New("IdleInhibitUnavailable", "the compositor has no zwp_idle_inhibit_manager_v1",
			"interface", wl.ZwpIdleInhibitManagerV1Interface.Name)
	}
	w.inhibitor = mgr.CreateInhibitor(w.surface)
	return nil
}

// contentTypes maps the hint of the app to the one of wp_content_type_v1.
var contentTypes = map[ui.ContentType]wl.WpContentTypeV1Type{
	ui.ContentTypeUnspecified: wl.WpContentTypeV1TypeNone,
	ui.ContentTypePhoto:       wl.WpContentTypeV1TypePhoto,
	ui.ContentTypeVideo:       wl.WpContentTypeV1TypeVideo,
	ui.ContentTypeGame:        wl.WpContentTypeV1TypeGame,
}

// SetContentType tells the compositor what the window shows (wp_content_type_v1),
// which it applies at the next commit. Without the protocol it does nothing.
//
// Flutter: no counterpart
func (w *window) SetContentType(contentType ui.ContentType) {
	mgr := w.p.sess.contentType
	if mgr == nil || (w.contentType == nil && contentType == ui.ContentTypeUnspecified) {
		return
	}
	if w.contentType == nil {
		w.contentType = mgr.GetSurfaceContentType(w.surface)
	}
	t, ok := contentTypes[contentType]
	if !ok {
		t = wl.WpContentTypeV1TypeNone
	}
	w.contentType.SetContentType(t)
}

// xdgToplevel returns the xdg_toplevel of the window: the window's own, or the one
// libdecor made.
func (w *window) xdgToplevel() *wl.XdgToplevel {
	if w.toplevel != nil {
		return w.toplevel
	}
	if w.frame != nil {
		if top := w.frame.Toplevel(); top != 0 {
			return w.p.sess.conn.WrapXdgToplevel(top)
		}
	}
	return nil
}

// SetIcon sets the icon of the window with xdg_toplevel_icon_v1: the name in the
// icon theme and the pictures, which go to the compositor as wl_shm buffers (the
// compositor copies them, but the buffers are kept until the next icon). An empty
// icon removes it. Without the protocol the icon comes from the .desktop file of
// the app, and SetIcon does nothing. It fails with WindowIconInvalid when a picture
// is wrong, and with ShmBufferFailed when the memory for the pictures cannot be made.
//
// Flutter: no counterpart
func (w *window) SetIcon(icon ui.WindowIcon) error {
	if err := icon.Validate(); err != nil {
		return err
	}
	mgr := w.p.sess.iconManager
	top := w.xdgToplevel()
	if mgr == nil || top == nil || w.closed {
		return nil
	}
	if icon.IsZero() {
		mgr.SetIcon(top, nil)
		w.freeIconBuffers()
		return nil
	}
	obj := mgr.CreateIcon()
	if icon.Name != "" {
		obj.SetName(icon.Name)
	}
	buffers := make([]*shmBuffer, 0, len(icon.Images))
	for _, img := range icon.Images {
		pixels := img.RGBA
		buf, err := newShmBufferFormat(w.p.sess.shm, int32(img.Size), int32(img.Size), wl.WlShmFormatArgb8888,
			func(data []byte) { premultiplyBGRA(data, pixels) }, nil)
		if err != nil {
			for _, b := range buffers {
				b.free()
			}
			obj.Destroy()
			return err
		}
		buffers = append(buffers, buf)
		obj.AddBuffer(buf.buffer, int32(max(img.Scale, 1)))
	}
	mgr.SetIcon(top, obj)
	obj.Destroy()
	w.freeIconBuffers()
	w.iconBuffers = buffers
	return nil
}

func (w *window) freeIconBuffers() {
	for _, b := range w.iconBuffers {
		b.free()
	}
	w.iconBuffers = nil
}

// premultiplyBGRA writes the pixels of rgba (straight alpha) into dst as the
// premultiplied ARGB32 of wl_shm, whose bytes in memory are blue, green, red, alpha.
func premultiplyBGRA(dst, rgba []byte) {
	for i := 0; i+3 < len(rgba) && i+3 < len(dst); i += 4 {
		r, g, b, a := uint32(rgba[i]), uint32(rgba[i+1]), uint32(rgba[i+2]), uint32(rgba[i+3])
		dst[i], dst[i+1], dst[i+2], dst[i+3] = byte((b*a+127)/255), byte((g*a+127)/255), byte((r*a+127)/255), byte(a)
	}
}

var _ platform.PortalHandleProvider = (*window)(nil)

// PortalHandle exports the window with zxdg_exporter_v2 and gives done the handle,
// "wayland:" and the compositor's string, which a portal takes as the parent of its
// dialog. The export is made once and kept until the window is destroyed; done is
// called on the UI thread, with an empty string when the compositor has no exporter
// or the window is closed. A window that libdecor decorates is exported through its
// surface like any other.
//
// Flutter: no counterpart
func (w *window) PortalHandle(done func(handle string)) {
	exporter := w.p.sess.exporter
	if exporter == nil || w.closed {
		done("")
		return
	}
	if w.exportHandle != "" {
		done(w.exportHandle)
		return
	}
	w.exportWaiting = append(w.exportWaiting, done)
	if w.exported != nil {
		return // the handle is on its way
	}
	w.exported = exporter.ExportToplevel(w.surface)
	w.exported.SetEvents(&wl.ZxdgExportedV2Events{OnHandle: func(handle string) {
		w.exportHandle = "wayland:" + handle
		waiting := w.exportWaiting
		w.exportWaiting = nil
		for _, f := range waiting {
			f(w.exportHandle)
		}
	}})
}
