package wl

// WrapBuffer returns a *WlBuffer for a wl_buffer that another library created on
// this connection, such as libwayland-cursor for a cursor image, so that it can
// be attached to a surface. The wrapper has no event handler and is not
// registered: the library that made the buffer owns it, and the Go side must not
// destroy it.
//
// Flutter: no counterpart
func (c *Conn) WrapBuffer(ptr uintptr) *WlBuffer {
	b := new(WlBuffer)
	b.conn = c
	b.ptr = ptr
	return b
}

// WrapXdgToplevel returns a *XdgToplevel for an xdg_toplevel that another library
// created on this connection, such as libdecor, so that it can be the argument of
// a request (xdg_wm_dialog_v1.get_xdg_dialog). The wrapper has no event handler
// and is not registered: the library that made the toplevel owns it, and the Go
// side must not destroy it.
//
// Flutter: no counterpart
func (c *Conn) WrapXdgToplevel(ptr uintptr) *XdgToplevel {
	t := new(XdgToplevel)
	t.conn = c
	t.ptr = ptr
	return t
}
